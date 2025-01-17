package config

import (
	"errors"
	"fmt"
	"math"
	"reflect"
)

type valueCache struct {
	stringVal      string
	stringArrayVal []string
	intVal         int64
	boolVal        bool
}

func (vc *valueCache) getData(opt *Option) interface{} {
	switch opt.OptType {
	case OptTypeBool:
		return vc.boolVal
	case OptTypeInt:
		return vc.intVal
	case OptTypeString:
		return vc.stringVal
	case OptTypeStringArray:
		return vc.stringArrayVal
	case optTypeAny:
		return nil
	default:
		return nil
	}
}

// isAllowedPossibleValue checks if value is defined as a PossibleValue
// in opt. If there are not possible values defined value is considered
// allowed and nil is returned. isAllowedPossibleValue ensure the actual
// value is an allowed primitiv value by using reflection to convert
// value and each PossibleValue to a comparable primitiv if possible.
// In case of complex value types isAllowedPossibleValue uses
// reflect.DeepEqual as a fallback.
func isAllowedPossibleValue(opt *Option, value interface{}) error {
	if opt.PossibleValues == nil {
		return nil
	}

	for _, val := range opt.PossibleValues {
		compareAgainst := val.Value
		valueType := reflect.TypeOf(value)

		// loading int's from the configuration JSON does not preserve the correct type
		// as we get float64 instead. Make sure to convert them before.
		if reflect.TypeOf(val.Value).ConvertibleTo(valueType) {
			compareAgainst = reflect.ValueOf(val.Value).Convert(valueType).Interface()
		}
		if compareAgainst == value {
			return nil
		}

		if reflect.DeepEqual(val.Value, value) {
			return nil
		}
	}

	return fmt.Errorf("value is not allowed")
}

// validateValue ensures that value matches the expected type of option.
// It does not create a copy of the value!
func validateValue(option *Option, value interface{}) (*valueCache, error) { //nolint:gocyclo
	if option.OptType != OptTypeStringArray {
		if err := isAllowedPossibleValue(option, value); err != nil {
			return nil, fmt.Errorf("validation of option %s failed for %v: %w", option.Key, value, err)
		}
	}

	reflect.TypeOf(value).ConvertibleTo(reflect.TypeOf(""))

	switch v := value.(type) {
	case string:
		if option.OptType != OptTypeString {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), option.Key, v)
		}
		if option.compiledRegex != nil {
			if !option.compiledRegex.MatchString(v) {
				return nil, fmt.Errorf("validation of option %s failed: string \"%s\" did not match validation regex for option", option.Key, v)
			}
		}
		return &valueCache{stringVal: v}, nil
	case []interface{}:
		vConverted := make([]string, len(v))
		for pos, entry := range v {
			s, ok := entry.(string)
			if !ok {
				return nil, fmt.Errorf("validation of option %s failed: element %+v at index %d is not a string", option.Key, entry, pos)
			}
			vConverted[pos] = s
		}
		// continue to next case
		return validateValue(option, vConverted)
	case []string:
		if option.OptType != OptTypeStringArray {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), option.Key, v)
		}
		if option.compiledRegex != nil {
			for pos, entry := range v {
				if !option.compiledRegex.MatchString(entry) {
					return nil, fmt.Errorf("validation of option %s failed: string \"%s\" at index %d did not match validation regex", option.Key, entry, pos)
				}

				if err := isAllowedPossibleValue(option, entry); err != nil {
					return nil, fmt.Errorf("validation of option %s failed: string %q at index %d is not allowed", option.Key, entry, pos)
				}
			}
		}
		return &valueCache{stringArrayVal: v}, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, float32, float64:
		// uint64 is omitted, as it does not fit in a int64
		if option.OptType != OptTypeInt {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), option.Key, v)
		}
		if option.compiledRegex != nil {
			// we need to use %v here so we handle float and int correctly.
			if !option.compiledRegex.MatchString(fmt.Sprintf("%v", v)) {
				return nil, fmt.Errorf("validation of option %s failed: number \"%d\" did not match validation regex", option.Key, v)
			}
		}
		switch v := value.(type) {
		case int:
			return &valueCache{intVal: int64(v)}, nil
		case int8:
			return &valueCache{intVal: int64(v)}, nil
		case int16:
			return &valueCache{intVal: int64(v)}, nil
		case int32:
			return &valueCache{intVal: int64(v)}, nil
		case int64:
			return &valueCache{intVal: v}, nil
		case uint:
			return &valueCache{intVal: int64(v)}, nil
		case uint8:
			return &valueCache{intVal: int64(v)}, nil
		case uint16:
			return &valueCache{intVal: int64(v)}, nil
		case uint32:
			return &valueCache{intVal: int64(v)}, nil
		case float32:
			// convert if float has no decimals
			if math.Remainder(float64(v), 1) == 0 {
				return &valueCache{intVal: int64(v)}, nil
			}
			return nil, fmt.Errorf("failed to convert float32 to int64 for option %s, got value %+v", option.Key, v)
		case float64:
			// convert if float has no decimals
			if math.Remainder(v, 1) == 0 {
				return &valueCache{intVal: int64(v)}, nil
			}
			return nil, fmt.Errorf("failed to convert float64 to int64 for option %s, got value %+v", option.Key, v)
		default:
			return nil, errors.New("internal error")
		}
	case bool:
		if option.OptType != OptTypeBool {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), option.Key, v)
		}
		return &valueCache{boolVal: v}, nil
	default:
		return nil, fmt.Errorf("invalid option value type for option %s: %T", option.Key, value)
	}
}
