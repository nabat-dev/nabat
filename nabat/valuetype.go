// Copyright 2026 The Nabat Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nabat

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// valueKind identifies a supported flag or positional-arg value type.
type valueKind int

const (
	valueString valueKind = iota + 1
	valueBool
	valueInt
	valueInt64
	valueUint
	valueFloat
	valueStringSlice
	valueSelect
	valueMultiSelect
	valueDuration
	valueCount
	valueBoolSlice
)

// valueType describes the expected type and optional constraints.
type valueType struct {
	kind    valueKind
	choices []string
}

func vtString() valueType      { return valueType{kind: valueString} }
func vtBool() valueType        { return valueType{kind: valueBool} }
func vtInt() valueType         { return valueType{kind: valueInt} }
func vtInt64() valueType       { return valueType{kind: valueInt64} }
func vtUint() valueType        { return valueType{kind: valueUint} }
func vtFloat() valueType       { return valueType{kind: valueFloat} }
func vtStringSlice() valueType { return valueType{kind: valueStringSlice} }
func vtSelect(choices ...string) valueType {
	return valueType{kind: valueSelect, choices: append([]string(nil), choices...)}
}

func vtMultiSelect(choices ...string) valueType {
	return valueType{kind: valueMultiSelect, choices: append([]string(nil), choices...)}
}
func vtDuration() valueType { return valueType{kind: valueDuration} }
func vtCount() valueType    { return valueType{kind: valueCount} }
func vtBoolSlice() valueType {
	return valueType{kind: valueBoolSlice}
}

func (v valueType) typeHint() string {
	switch v.kind {
	case valueSelect:
		return strings.Join(v.choices, "|")
	case valueMultiSelect:
		return strings.Join(v.choices, "|") + "..."
	}
	if a := adapterFor(v.kind); a != nil {
		return a.typeHint()
	}
	return ""
}

// valueAdapter centralizes the per-kind behavior previously scattered across
// [parseStringToType], [registerFlagOnCommand], [readFlagTypedValue], and
// [validateDefaultType]. Adding a new value kind means writing one adapter and
// registering it; nothing else needs to change.
type valueAdapter interface {
	typeHint() string
	// parse converts a raw command-line/env string into the typed value. The
	// returned value is what gets stored in [Context.values].
	parse(raw string) (any, error)
	// registerFlag wires the value into a pflag set with the right type.
	registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error
	// readFlag reads the value back from pflag after parsing.
	readFlag(set *pflag.FlagSet, name string) (any, error)
	// checkDefault reports nil when def is the right Go type for this kind, or a
	// descriptive error otherwise (used by [validateDefaultType]).
	checkDefault(def any) error
}

// defaultAs returns def coerced to T, or the zero value of T when def is nil
// or not a T. Adapter [valueAdapter.checkDefault] enforces type correctness
// before [valueAdapter.registerFlag] runs, so the zero-value branch only
// executes when no default was set.
func defaultAs[T any](def any) T {
	v, ok := def.(T)
	if !ok {
		var zero T
		return zero
	}
	return v
}

// stringAdapter handles string and select kinds: both use a pflag string under
// the hood, parse identically, and stash the value as a Go string.
type stringAdapter struct{ hint string }

func (a stringAdapter) typeHint() string { return a.hint }
func (a stringAdapter) parse(raw string) (any, error) {
	return raw, nil
}

func (a stringAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.StringP(name, short, defaultAs[string](def), usage)
	return nil
}

func (a stringAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetString(name)
}

func (a stringAdapter) checkDefault(def any) error {
	if _, ok := def.(string); !ok {
		return fmt.Errorf("default must be string")
	}
	return nil
}

type boolAdapter struct{}

func (boolAdapter) typeHint() string { return "bool" }
func (boolAdapter) parse(raw string) (any, error) {
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("expected bool, got %q", raw)
	}
	return v, nil
}

func (boolAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.BoolP(name, short, defaultAs[bool](def), usage)
	return nil
}

func (boolAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetBool(name)
}

func (boolAdapter) checkDefault(def any) error {
	if _, ok := def.(bool); !ok {
		return fmt.Errorf("default must be bool")
	}
	return nil
}

type intAdapter struct{}

func (intAdapter) typeHint() string { return "int" }
func (intAdapter) parse(raw string) (any, error) {
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("expected int, got %q", raw)
	}
	return v, nil
}

func (intAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.IntP(name, short, defaultAs[int](def), usage)
	return nil
}

func (intAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetInt(name)
}

func (intAdapter) checkDefault(def any) error {
	if _, ok := def.(int); !ok {
		return fmt.Errorf("default must be int")
	}
	return nil
}

type int64Adapter struct{}

func (int64Adapter) typeHint() string { return "int64" }
func (int64Adapter) parse(raw string) (any, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expected int64, got %q", raw)
	}
	return v, nil
}

func (int64Adapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.Int64P(name, short, defaultAs[int64](def), usage)
	return nil
}

func (int64Adapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetInt64(name)
}

func (int64Adapter) checkDefault(def any) error {
	if _, ok := def.(int64); !ok {
		return fmt.Errorf("default must be int64")
	}
	return nil
}

type uintAdapter struct{}

func (uintAdapter) typeHint() string { return "uint" }
func (uintAdapter) parse(raw string) (any, error) {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expected uint, got %q", raw)
	}
	return uint(v), nil
}

func (uintAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.UintP(name, short, defaultAs[uint](def), usage)
	return nil
}

func (uintAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetUint(name)
}

func (uintAdapter) checkDefault(def any) error {
	if _, ok := def.(uint); !ok {
		return fmt.Errorf("default must be uint")
	}
	return nil
}

type floatAdapter struct{}

func (floatAdapter) typeHint() string { return "float" }
func (floatAdapter) parse(raw string) (any, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("expected float, got %q", raw)
	}
	return v, nil
}

func (floatAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.Float64P(name, short, defaultAs[float64](def), usage)
	return nil
}

func (floatAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetFloat64(name)
}

func (floatAdapter) checkDefault(def any) error {
	if _, ok := def.(float64); !ok {
		return fmt.Errorf("default must be float64")
	}
	return nil
}

type durationAdapter struct{}

func (durationAdapter) typeHint() string { return "duration" }
func (durationAdapter) parse(raw string) (any, error) {
	v, err := time.ParseDuration(raw)
	if err != nil {
		return nil, fmt.Errorf("expected duration (e.g. 30s, 5m), got %q", raw)
	}
	return v, nil
}

func (durationAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	target.DurationP(name, short, defaultAs[time.Duration](def), usage)
	return nil
}

func (durationAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetDuration(name)
}

func (durationAdapter) checkDefault(def any) error {
	if _, ok := def.(time.Duration); !ok {
		return fmt.Errorf("default must be time.Duration")
	}
	return nil
}

type countAdapter struct{}

func (countAdapter) typeHint() string { return "count" }
func (countAdapter) parse(_ string) (any, error) {
	return nil, fmt.Errorf("count is flag-only and cannot be parsed from a string")
}

func (countAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetCount(name)
}

func (countAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, _ any) error {
	target.CountP(name, short, usage)
	return nil
}
func (countAdapter) checkDefault(_ any) error { return nil } // no default for count

type boolSliceAdapter struct{}

func (boolSliceAdapter) typeHint() string { return "bool..." }

func (boolSliceAdapter) parse(raw string) (any, error) {
	parts := splitCSV(raw)
	out := make([]bool, 0, len(parts))
	for _, p := range parts {
		b, err := strconv.ParseBool(p)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (boolSliceAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	d := defaultAs[[]bool](def)
	target.BoolSliceP(name, short, append([]bool(nil), d...), usage)
	return nil
}

func (boolSliceAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetBoolSlice(name)
}

func (boolSliceAdapter) checkDefault(def any) error {
	if _, ok := def.([]bool); !ok {
		return fmt.Errorf("default must be []bool")
	}
	return nil
}

// splitCSV splits raw on commas and trims surrounding whitespace from each
// part. Empty input returns an empty slice. Both string-slice and bool-slice
// adapters use this so env-var values like "foo, bar" behave identically
// regardless of the value kind.
func splitCSV(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// stringSliceAdapter handles string slice and multi-select kinds (both stored
// as []string and registered via StringSliceP).
type stringSliceAdapter struct{ hint string }

func (a stringSliceAdapter) typeHint() string { return a.hint }
func (a stringSliceAdapter) parse(raw string) (any, error) {
	return splitCSV(raw), nil
}

func (a stringSliceAdapter) registerFlag(target *pflag.FlagSet, name, short, usage string, def any) error {
	d := defaultAs[[]string](def)
	target.StringSliceP(name, short, append([]string(nil), d...), usage)
	return nil
}

func (a stringSliceAdapter) readFlag(set *pflag.FlagSet, name string) (any, error) {
	return set.GetStringSlice(name)
}

func (a stringSliceAdapter) checkDefault(def any) error {
	if _, ok := def.([]string); !ok {
		return fmt.Errorf("default must be []string")
	}
	return nil
}

// adapters indexes the per-kind value adapter. Adding a new kind: append to the
// const block above, append to this map, done.
var adapters = map[valueKind]valueAdapter{
	valueString:      stringAdapter{hint: "string"},
	valueSelect:      stringAdapter{hint: ""},
	valueBool:        boolAdapter{},
	valueInt:         intAdapter{},
	valueInt64:       int64Adapter{},
	valueUint:        uintAdapter{},
	valueFloat:       floatAdapter{},
	valueDuration:    durationAdapter{},
	valueCount:       countAdapter{},
	valueBoolSlice:   boolSliceAdapter{},
	valueStringSlice: stringSliceAdapter{hint: "string..."},
	valueMultiSelect: stringSliceAdapter{hint: ""},
}

// adapterFor returns the adapter for kind, or nil when kind is unknown.
func adapterFor(kind valueKind) valueAdapter { return adapters[kind] }

// parseStringToType converts a CLI/env string into the typed value for vt.
// Wraps the kind-specific adapter so callers do not need to switch on kind.
func parseStringToType(raw string, vt valueType) (any, error) {
	a := adapterFor(vt.kind)
	if a == nil {
		return nil, fmt.Errorf("%w for value parsing", ErrInvalidValueType)
	}
	return a.parse(raw)
}

// validateChoice enforces select/multi-select choice membership when applicable.
func validateChoice(vt valueType, value any) error {
	switch vt.kind {
	case valueSelect:
		if len(vt.choices) == 0 {
			return nil
		}
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for select")
		}
		if !slices.Contains(vt.choices, s) {
			return fmt.Errorf("must be one of %v", vt.choices)
		}
	case valueMultiSelect:
		if len(vt.choices) == 0 {
			return nil
		}
		values, ok := value.([]string)
		if !ok {
			return fmt.Errorf("expected []string for multi-select")
		}
		for _, item := range values {
			if !slices.Contains(vt.choices, item) {
				return fmt.Errorf("value %q must be one of %v", item, vt.choices)
			}
		}
	}
	return nil
}

func cloneDefault(v any) any {
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case []bool:
		return append([]bool(nil), t...)
	default:
		return v
	}
}
