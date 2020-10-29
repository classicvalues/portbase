// Package config ... (linter fix)
//nolint:dupl
package config

import (
	"sync/atomic"

	"github.com/tevino/abool"
)

// ReleaseLevel is used to define the maturity of a
// configuration setting.
type ReleaseLevel uint8

// Release Level constants
const (
	ReleaseLevelStable       ReleaseLevel = 0
	ReleaseLevelBeta         ReleaseLevel = 1
	ReleaseLevelExperimental ReleaseLevel = 2

	ReleaseLevelNameStable       = "stable"
	ReleaseLevelNameBeta         = "beta"
	ReleaseLevelNameExperimental = "experimental"

	releaseLevelKey = "core/releaseLevel"
)

var (
	releaseLevel           = new(int32)
	releaseLevelOption     *Option
	releaseLevelOptionFlag = abool.New()
)

func init() {
	registerReleaseLevelOption()
}

func registerReleaseLevelOption() {
	releaseLevelOption = &Option{
		Name:           "Release Level",
		Key:            releaseLevelKey,
		Description:    "Controls the amount of available settings. Hidden settings revert to default - unlike with the UI Mode.",
		OptType:        OptTypeString,
		ExpertiseLevel: ExpertiseLevelExpert,
		ReleaseLevel:   ReleaseLevelStable,
		DefaultValue:   ReleaseLevelNameStable,
		Annotations: Annotations{
			DisplayHintAnnotation: DisplayHintOneOf,
			CategoryAnnotation:    "Expertise & Release",
		},
		PossibleValues: []PossibleValue{
			{
				Name:        "Stable",
				Value:       ReleaseLevelNameStable,
				Description: "Only show stable features.",
			},
			{
				Name:        "Beta",
				Value:       ReleaseLevelNameBeta,
				Description: "Show stable and beta features.",
			},
			{
				Name:        "Experimental",
				Value:       ReleaseLevelNameExperimental,
				Description: "Show all features",
			},
		},
	}

	err := Register(releaseLevelOption)
	if err != nil {
		panic(err)
	}

	releaseLevelOptionFlag.Set()
}

func updateReleaseLevel() {
	// get value
	value := releaseLevelOption.activeFallbackValue
	if releaseLevelOption.activeValue != nil {
		value = releaseLevelOption.activeValue
	}
	if releaseLevelOption.activeDefaultValue != nil {
		value = releaseLevelOption.activeDefaultValue
	}
	// set atomic value
	switch value.stringVal {
	case ReleaseLevelNameStable:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelStable))
	case ReleaseLevelNameBeta:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelBeta))
	case ReleaseLevelNameExperimental:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelExperimental))
	default:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelStable))
	}
}

func getReleaseLevel() ReleaseLevel {
	return ReleaseLevel(atomic.LoadInt32(releaseLevel))
}
