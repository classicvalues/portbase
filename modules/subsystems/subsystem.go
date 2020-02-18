package subsystems

import (
	"sync"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

// Subsystem describes a subset of modules that represent a part of a service or program to the user.
type Subsystem struct { //nolint:maligned // not worth the effort
	record.Base
	sync.Mutex

	Name        string
	Description string
	module      *modules.Module

	Status        *ModuleStatus
	Dependencies  []*ModuleStatus
	FailureStatus uint8

	ToggleOptionKey string // empty == forced on
	toggleOption    *config.Option
	toggleValue     func() bool
	ExpertiseLevel  uint8 // copied from toggleOption
	ReleaseLevel    uint8 // copied from toggleOption

	ConfigKeySpace string
}

// ModuleStatus describes the status of a module.
type ModuleStatus struct {
	Name   string
	module *modules.Module

	// status mgmt
	Enabled bool
	Status  uint8

	// failure status
	FailureStatus uint8
	FailureID     string
	FailureMsg    string
}

// Save saves the Subsystem Status to the database.
func (sub *Subsystem) Save() {
	if databaseKeySpace != "" {
		// sub.SetKey() // FIXME
		err := db.Put(sub)
		if err != nil {
			log.Errorf("subsystems: could not save subsystem status to database: %s", err)
		}
	}
}

func statusFromModule(module *modules.Module) *ModuleStatus {
	status := &ModuleStatus{
		Name:    module.Name,
		module:  module,
		Enabled: module.Enabled(),
		Status:  module.Status(),
	}
	status.FailureStatus, status.FailureID, status.FailureMsg = module.FailureStatus()

	return status
}

func compareAndUpdateStatus(module *modules.Module, status *ModuleStatus) (changed bool) {
	// check if enabled
	enabled := module.Enabled()
	if status.Enabled != enabled {
		status.Enabled = enabled
		changed = true
	}

	// check status
	statusLvl := module.Status()
	if status.Status != statusLvl {
		status.Status = statusLvl
		changed = true
	}

	// check failure status
	failureStatus, failureID, failureMsg := module.FailureStatus()
	if status.FailureStatus != failureStatus ||
		status.FailureID != failureID {
		status.FailureStatus = failureStatus
		status.FailureID = failureID
		status.FailureMsg = failureMsg
		changed = true
	}

	return
}

func (sub *Subsystem) makeSummary() {
	// find worst failing module
	worstFailing := &ModuleStatus{}
	if sub.Status.FailureStatus > worstFailing.FailureStatus {
		worstFailing = sub.Status
	}
	for _, depStatus := range sub.Dependencies {
		if depStatus.FailureStatus > worstFailing.FailureStatus {
			worstFailing = depStatus
		}
	}

	if worstFailing != nil {
		sub.FailureStatus = worstFailing.FailureStatus
	} else {
		sub.FailureStatus = 0
	}
}
