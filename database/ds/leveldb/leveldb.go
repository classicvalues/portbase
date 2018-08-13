// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package leveldb

import (
	"fmt"

	"github.com/Safing/safing-core/database/dbutils"
	"github.com/Safing/safing-core/formats/dsd"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/jbenet/goprocess"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type datastore struct {
	DB *leveldb.DB
}

type Options opt.Options

func NewDatastore(path string, opts *Options) (*datastore, error) {
	var nopts opt.Options
	if opts != nil {
		nopts = opt.Options(*opts)
	}
	db, err := leveldb.OpenFile(path, &nopts)
	if err != nil {
		return nil, err
	}

	return &datastore{
		DB: db,
	}, nil
}

// Returns ErrInvalidType if value is not of type []byte.
//
// Note: using sync = false.
// see http://godoc.org/github.com/syndtr/goleveldb/leveldb/opt#WriteOptions
func (d *datastore) Put(key ds.Key, value interface{}) (err error) {
	bytes, err := dbutils.DumpModel(value, dsd.AUTO) // or other dsd format
	if err != nil {
		return err
	}
	return d.DB.Put(key.Bytes(), bytes, nil)
}

func (d *datastore) Get(key ds.Key) (interface{}, error) {
	data, err := d.DB.Get(key.Bytes(), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ds.ErrNotFound
		}
		return nil, err
	}

	model, err := dbutils.NewWrapper(&key, data)
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (d *datastore) Has(key ds.Key) (exists bool, err error) {
	return d.DB.Has(key.Bytes(), nil)
}

func (d *datastore) Delete(key ds.Key) (err error) {
	// leveldb Delete will not return an error if the key doesn't
	// exist (see https://github.com/syndtr/goleveldb/issues/109),
	// so check that the key exists first and if not return an
	// error
	exists, err := d.DB.Has(key.Bytes(), nil)
	if !exists {
		return ds.ErrNotFound
	} else if err != nil {
		return err
	}
	return d.DB.Delete(key.Bytes(), nil)
}

func (d *datastore) Query(q dsq.Query) (dsq.Results, error) {
	return d.QueryNew(q)
}

func (d *datastore) QueryNew(q dsq.Query) (dsq.Results, error) {
	if len(q.Filters) > 0 ||
		len(q.Orders) > 0 ||
		q.Limit > 0 ||
		q.Offset > 0 {
		return d.QueryOrig(q)
	}
	var rnge *util.Range
	if q.Prefix != "" {
		rnge = util.BytesPrefix([]byte(q.Prefix))
	}
	i := d.DB.NewIterator(rnge, nil)
	return dsq.ResultsFromIterator(q, dsq.Iterator{
		Next: func() (dsq.Result, bool) {
			ok := i.Next()
			if !ok {
				return dsq.Result{}, false
			}
			k := string(i.Key())
			e := dsq.Entry{Key: k}

			if !q.KeysOnly {
				buf := make([]byte, len(i.Value()))
				copy(buf, i.Value())

				newKey := ds.RawKey(k)
				wrapper, err := dbutils.NewWrapper(&newKey, buf)
				if err != nil {
					return dsq.Result{Error: fmt.Errorf("failed to create wrapper for %s: %s", k, err)}, false
				}
				e.Value = wrapper
			}
			return dsq.Result{Entry: e}, true
		},
		Close: func() error {
			i.Release()
			return nil
		},
	}), nil
}

func (d *datastore) QueryOrig(q dsq.Query) (dsq.Results, error) {
	// we can use multiple iterators concurrently. see:
	// https://godoc.org/github.com/syndtr/goleveldb/leveldb#DB.NewIterator
	// advance the iterator only if the reader reads
	//
	// run query in own sub-process tied to Results.Process(), so that
	// it waits for us to finish AND so that clients can signal to us
	// that resources should be reclaimed.
	qrb := dsq.NewResultBuilder(q)
	qrb.Process.Go(func(worker goprocess.Process) {
		d.runQuery(worker, qrb)
	})

	// go wait on the worker (without signaling close)
	go qrb.Process.CloseAfterChildren()

	// Now, apply remaining things (filters, order)
	qr := qrb.Results()
	for _, f := range q.Filters {
		qr = dsq.NaiveFilter(qr, f)
	}
	for _, o := range q.Orders {
		qr = dsq.NaiveOrder(qr, o)
	}
	return qr, nil
}

func (d *datastore) runQuery(worker goprocess.Process, qrb *dsq.ResultBuilder) {

	var rnge *util.Range
	if qrb.Query.Prefix != "" {
		rnge = util.BytesPrefix([]byte(qrb.Query.Prefix))
	}
	i := d.DB.NewIterator(rnge, nil)
	defer i.Release()

	// advance iterator for offset
	if qrb.Query.Offset > 0 {
		for j := 0; j < qrb.Query.Offset; j++ {
			i.Next()
		}
	}

	// iterate, and handle limit, too
	for sent := 0; i.Next(); sent++ {
		// end early if we hit the limit
		if qrb.Query.Limit > 0 && sent >= qrb.Query.Limit {
			break
		}

		k := string(i.Key())
		e := dsq.Entry{Key: k}

		if !qrb.Query.KeysOnly {
			buf := make([]byte, len(i.Value()))
			copy(buf, i.Value())

			newKey := ds.RawKey(k)
			wrapper, err := dbutils.NewWrapper(&newKey, buf)
			if err != nil {
				return
			}
			e.Value = wrapper
		}

		select {
		case qrb.Output <- dsq.Result{Entry: e}: // we sent it out
		case <-worker.Closing(): // client told us to end early.
			break
		}
	}

	if err := i.Error(); err != nil {
		select {
		case qrb.Output <- dsq.Result{Error: err}: // client read our error
		case <-worker.Closing(): // client told us to end.
			return
		}
	}
}

// LevelDB needs to be closed.
func (d *datastore) Close() (err error) {
	return d.DB.Close()
}

func (d *datastore) IsThreadSafe() {}

type leveldbBatch struct {
	b  *leveldb.Batch
	db *leveldb.DB
}

func (d *datastore) Batch() (ds.Batch, error) {
	return &leveldbBatch{
		b:  new(leveldb.Batch),
		db: d.DB,
	}, nil
}

func (b *leveldbBatch) Put(key ds.Key, value interface{}) error {
	val, err := dsd.Dump(value, dsd.AUTO)
	if err != nil {
		// return ds.ErrInvalidType
		return err
	}

	b.b.Put(key.Bytes(), val)
	return nil
}

func (b *leveldbBatch) Commit() error {
	return b.db.Write(b.b, nil)
}

func (b *leveldbBatch) Delete(key ds.Key) error {
	b.b.Delete(key.Bytes())
	return nil
}
