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
	"time"

	"charm.land/huh/v2"
)

type argDef struct {
	name      string
	valueType valueType
	config    fieldConfig
	prompt    promptConfig
}

func (i argDef) validate() error {
	if i.name == "" {
		return fmt.Errorf("nabat: arg name cannot be empty")
	}
	if i.valueType.kind == 0 {
		return fmt.Errorf("%w for arg %q", ErrInvalidValueType, i.name)
	}
	if (i.valueType.kind == valueSelect || i.valueType.kind == valueMultiSelect) && len(i.valueType.choices) == 0 {
		return fmt.Errorf("nabat: arg %q: Select/MultiSelect requires at least one choice", i.name)
	}
	if i.valueType.kind == valueCount {
		return fmt.Errorf("nabat: arg %q: Count is flag-only and cannot be used as a positional arg", i.name)
	}
	return validateDefaultType("arg", i.name, i.valueType, i.config)
}

func (a *App) promptArg(def argDef) (any, error) {
	pc := def.prompt

	switch def.valueType.kind {
	case valueString:
		var out string
		var field huh.Field
		switch pc.mode {
		case widgetMultiline:
			field = buildTextField(&out, pc)
		case widgetFilePicker:
			field = buildFileField(&out, pc)
		default:
			field = buildInputField(&out, pc)
		}
		if err := a.runPromptField(field); err != nil {
			return nil, err
		}
		return out, nil
	case valueBool:
		var out bool
		if err := a.runPromptField(buildConfirmField(&out, pc)); err != nil {
			return nil, err
		}
		return out, nil
	case valueInt:
		var target int
		if err := a.runPromptField(buildNumericInputField(&target, pc)); err != nil {
			return nil, err
		}
		return target, nil
	case valueInt64:
		var target int64
		if err := a.runPromptField(buildNumericInputField(&target, pc)); err != nil {
			return nil, err
		}
		return target, nil
	case valueUint:
		var target uint
		if err := a.runPromptField(buildNumericInputField(&target, pc)); err != nil {
			return nil, err
		}
		return target, nil
	case valueFloat:
		var target float64
		if err := a.runPromptField(buildNumericInputField(&target, pc)); err != nil {
			return nil, err
		}
		return target, nil
	case valueDuration:
		var target time.Duration
		if err := a.runPromptField(buildNumericInputField(&target, pc)); err != nil {
			return nil, err
		}
		return target, nil
	case valueSelect:
		options := make([]huh.Option[string], 0, len(def.valueType.choices))
		for _, choice := range def.valueType.choices {
			options = append(options, huh.NewOption(choice, choice))
		}
		var out string
		f := huh.NewSelect[string]().Title(pc.text).Options(options...).Value(&out)
		if pc.description != "" {
			f = f.Description(pc.description)
		}
		f = f.Filtering(pc.filtering)
		if pc.height > 0 {
			f = f.Height(pc.height)
		}
		if pc.validate != nil {
			fn := pc.validate
			f = f.Validate(func(s string) error { return fn(s) })
		}
		if err := a.runPromptField(f); err != nil {
			return nil, err
		}
		return out, nil
	case valueStringSlice, valueMultiSelect:
		options := make([]huh.Option[string], 0, len(def.valueType.choices))
		for _, choice := range def.valueType.choices {
			options = append(options, huh.NewOption(choice, choice))
		}
		var out []string
		f := huh.NewMultiSelect[string]().Title(pc.text).Options(options...).Value(&out)
		if pc.description != "" {
			f = f.Description(pc.description)
		}
		f = f.Filterable(pc.filtering)
		if pc.height > 0 {
			f = f.Height(pc.height)
		}
		if pc.limit > 0 {
			f = f.Limit(pc.limit)
		}
		if pc.validate != nil {
			fn := pc.validate
			f = f.Validate(func(ss []string) error { return fn(ss) })
		}
		if err := a.runPromptField(f); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w for arg prompt", ErrInvalidValueType)
	}
}

func (a *App) runPromptField(field huh.Field) error {
	form := huh.NewForm(huh.NewGroup(field))
	a.applyHuhTheme(form)
	return form.Run()
}
