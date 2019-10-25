package modules

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
)

// Worker Default Configuration
const (
	DefaultBackoffDuration = 2 * time.Second
)

var (
	errNoModule = errors.New("missing module (is nil!)")
)

// RunWorker directly runs a generic worker that does not fit to be a Task or MicroTask, such as long running (and possibly mostly idle) sessions. A call to RunWorker blocks until the worker is finished.
func (m *Module) RunWorker(name string, fn func(context.Context) error) error {
	if m == nil {
		log.Errorf(`modules: cannot start worker "%s" with nil module`, name)
		return errNoModule
	}

	atomic.AddInt32(m.workerCnt, 1)
	m.waitGroup.Add(1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.waitGroup.Done()
	}()

	return m.runWorker(name, fn)
}

// StartServiceWorker starts a generic worker, which is automatically restarted in case of an error. A call to StartServiceWorker runs the service-worker in a new goroutine and returns immediately. `backoffDuration` specifies how to long to wait before restarts, multiplied by the number of failed attempts. Pass `0` for the default backoff duration. For custom error remediation functionality, build your own error handling procedure using calls to RunWorker.
func (m *Module) StartServiceWorker(name string, backoffDuration time.Duration, fn func(context.Context) error) {
	if m == nil {
		log.Errorf(`modules: cannot start service worker "%s" with nil module`, name)
		return
	}

	go m.runServiceWorker(name, backoffDuration, fn)
}

func (m *Module) runServiceWorker(name string, backoffDuration time.Duration, fn func(context.Context) error) {
	atomic.AddInt32(m.workerCnt, 1)
	m.waitGroup.Add(1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.waitGroup.Done()
	}()

	if backoffDuration == 0 {
		backoffDuration = DefaultBackoffDuration
	}
	failCnt := 0
	lastFail := time.Now()

	for {
		if m.ShutdownInProgress() {
			return
		}

		err := m.runWorker(name, fn)
		if err != nil {
			// reset fail counter if running without error for some time
			if time.Now().Add(-5 * time.Minute).After(lastFail) {
				failCnt = 0
			}
			// increase fail counter and set last failed time
			failCnt++
			lastFail = time.Now()
			// log error
			sleepFor := time.Duration(failCnt) * backoffDuration
			log.Errorf("%s: service-worker %s failed (%d): %s - restarting in %s", m.Name, name, failCnt, err, sleepFor)
			time.Sleep(sleepFor)
			// loop to restart
		} else {
			// finish
			return
		}
	}
}

func (m *Module) runWorker(name string, fn func(context.Context) error) (err error) {
	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := m.NewPanicError(name, "worker", panicVal)
			me.Report()
			err = me
		}
	}()

	// run
	err = fn(m.Ctx)
	return
}

func (m *Module) runCtrlFnWithTimeout(name string, timeout time.Duration, fn func() error) error {

	stopFnError := make(chan error)
	go func() {
		stopFnError <- m.runCtrlFn(name, fn)
	}()

	// wait for results
	select {
	case err := <-stopFnError:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timed out (%s)", timeout)
	}
}

func (m *Module) runCtrlFn(name string, fn func() error) (err error) {
	if fn == nil {
		return
	}

	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := m.NewPanicError(name, "module-control", panicVal)
			me.Report()
			err = me
		}
	}()

	// run
	err = fn()
	return
}