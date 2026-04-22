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
	"reflect"
)

// Bind copies resolved positional arg and flag values into target, which must be
// a non-nil pointer to a struct.
//
// For each exported field tagged `nabat:"name"`, Bind assigns the resolved value
// for name when one exists. Fields without a tag are skipped. Anonymous embedded
// structs without a nabat tag are walked recursively.
//
// Value fields (int, string, [time.Duration], …) are always assigned when a
// value exists in the context, including registered defaults when the user did
// not supply a value.
//
// Pointer fields (*int, *string, *[time.Duration], …) encode optionality: the
// field
// stays nil when the value was resolved only from a default or when the name was
// never resolved. When the user supplied the value via CLI, environment variable,
// or interactive prompt, Bind allocates a pointer and stores the resolved value.
//
// Errors:
//   - "nabat: bind target: context is nil" when c is nil
//   - "nabat: bind target must be a non-nil pointer to struct" when target is
//     not a non-nil *struct
//   - "nabat: bind target must be a pointer to struct" when target is not a
//     pointer to struct
//   - "nabat: bind field ...: field is not settable" for unexported or
//     non-settable fields
//   - "nabat: bind field ...: expected ..., got <nil>" when the resolved value is
//     nil
//   - "nabat: bind field ...: expected T, got U" when the resolved type does not
//     assign to the field
//   - "nabat: bind field ...: tag ... does not match any declared arg or flag"
//     when the tag names no field on the current command (typo or stale struct)
func (c *Context) Bind(target any) error {
	if c == nil {
		return fmt.Errorf("nabat: bind target: context is nil")
	}

	rv := reflect.ValueOf(target)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("nabat: bind target must be a non-nil pointer to struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("nabat: bind target must be a pointer to struct")
	}

	return c.bindStruct(rv, rv.Type(), "", c.declaredBindNames())
}

// BindAs returns the resolved value for a single arg or flag name. It is a
// convenience for tests and one-off reads; prefer binding into a struct with
// [Context.Bind] in command handlers.
//
// Errors:
//   - "nabat: BindAs: context is nil" when c is nil
//   - "nabat: BindAs: %q has no resolved value" when the name was not resolved
//   - "nabat: BindAs %q: value type ... does not match requested type ..." on type
//     mismatch
func BindAs[T any](c *Context, name string) (T, error) {
	var zero T
	if c == nil {
		return zero, fmt.Errorf("nabat: BindAs: context is nil")
	}
	v, ok := c.values[name]
	if !ok {
		return zero, fmt.Errorf("nabat: BindAs: %q has no resolved value", name)
	}
	tv, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("nabat: BindAs %q: value type %T does not match requested type %T", name, v, zero)
	}
	return tv, nil
}

func (c *Context) bindStruct(value reflect.Value, typ reflect.Type, prefix string, declared map[string]bool) error {
	for i := range typ.NumField() {
		structField := typ.Field(i)
		fieldValue := value.Field(i)

		// Walk embedded structs when they are not explicitly tagged.
		if structField.Anonymous && structField.Tag.Get("nabat") == "" {
			switch structField.Type.Kind() {
			case reflect.Struct:
				if err := c.bindStruct(fieldValue, structField.Type, joinFieldPath(prefix, structField.Name), declared); err != nil {
					return err
				}
				continue
			case reflect.Pointer:
				if structField.Type.Elem().Kind() == reflect.Struct {
					if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(structField.Type.Elem()))
					}
					if err := c.bindStruct(fieldValue.Elem(), structField.Type.Elem(), joinFieldPath(prefix, structField.Name), declared); err != nil {
						return err
					}
					continue
				}
			}
		}

		name := structField.Tag.Get("nabat")
		if name == "" {
			continue
		}
		fieldPath := joinFieldPath(prefix, structField.Name)

		if !fieldValue.CanSet() {
			return fmt.Errorf("nabat: bind field %s: field is not settable", fieldPath)
		}

		resolved, ok := c.values[name]
		if !ok {
			if len(declared) > 0 && !declared[name] {
				return fmt.Errorf("nabat: bind field %s: tag %q does not match any declared arg or flag", fieldPath, name)
			}
			continue // optional declared field not provided — struct field keeps its zero value
		}

		resolvedValue := reflect.ValueOf(resolved)
		if !resolvedValue.IsValid() {
			return fmt.Errorf("nabat: bind field %s: expected %s, got <nil>", fieldPath, fieldValue.Type())
		}

		if fieldValue.Kind() == reflect.Pointer {
			if !c.set[name] {
				// Optional pointer: leave nil when the value came only from a default or implicit resolution.
				continue
			}
			if err := assignPointer(fieldPath, fieldValue, resolvedValue); err != nil {
				return err
			}
			continue
		}

		if !resolvedValue.Type().AssignableTo(fieldValue.Type()) {
			return fmt.Errorf("nabat: bind field %s: expected %s, got %s", fieldPath, fieldValue.Type(), resolvedValue.Type())
		}

		fieldValue.Set(resolvedValue)
	}

	return nil
}

func assignPointer(
	fieldPath string,
	fieldValue reflect.Value,
	resolvedValue reflect.Value,
) error {
	elemType := fieldValue.Type().Elem()
	if elemType.Kind() == reflect.Struct {
		return fmt.Errorf("nabat: bind field %s: pointer to struct is not supported; use a value field", fieldPath)
	}
	rv := resolvedValue
	if !rv.Type().AssignableTo(elemType) {
		if rv.CanConvert(elemType) {
			rv = rv.Convert(elemType)
		} else {
			return fmt.Errorf("nabat: bind field %s: expected %s, got %s", fieldPath, fieldValue.Type(), resolvedValue.Type())
		}
	}
	ptr := reflect.New(elemType)
	ptr.Elem().Set(rv)
	fieldValue.Set(ptr)
	return nil
}

// declaredBindNames returns the declared positional arg and flag names once.
// When the context has no app or command (e.g. tests with a hand-built Context),
// validation is skipped by returning an empty map and Bind treats all names as
// declared.
func (c *Context) declaredBindNames() map[string]bool {
	names := map[string]bool{}
	if c.app == nil || c.cmd == nil {
		return names
	}
	meta := c.app.meta[c.cmd]
	if meta == nil {
		return names
	}
	for _, a := range meta.args {
		names[a.name] = true
	}
	for _, fl := range c.app.collectFlagDefsForCommand(c.cmd) {
		names[fl.name] = true
	}
	return names
}

func joinFieldPath(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}
