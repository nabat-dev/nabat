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
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"

	"nabat.dev/theme"
)

// Enumerator presets for [WithListEnumerator] (aliases for lipgloss list
// enumerators): [ListBullet] (default), [ListDash], [ListAsterisk],
// [ListNumbered], [ListRoman], [ListAlphabet].
var (
	ListBullet   = list.Bullet
	ListDash     = list.Dash
	ListAsterisk = list.Asterisk
	ListNumbered = list.Arabic
	ListRoman    = list.Roman
	ListAlphabet = list.Alphabet
)

// ListItems is the type passed to list style and enumerator functions.
type ListItems = list.Items

// ListEnumerator is the function signature for custom list enumerators.
type ListEnumerator = list.Enumerator

// ListStyleFunc is a function that determines the style of a list item based
// on the item set and the current index.
type ListStyleFunc = list.StyleFunc

type listConfig struct {
	enumerator  list.Enumerator
	itemStyle   lipgloss.Style
	enumStyle   lipgloss.Style
	itemStyleFn list.StyleFunc
	enumStyleFn list.StyleFunc
}

// ListOption configures [Context.List] (enumerator and styling).
type ListOption func(*listConfig)

// WithListEnumerator sets the enumerator used to prefix each list item.
// Defaults to [ListBullet]; also see ListDash, ListNumbered, ListRoman, and
// ListAlphabet.
//
// Example:
//
//	WithListEnumerator(ListNumbered)
func WithListEnumerator(e list.Enumerator) ListOption {
	return func(c *listConfig) { c.enumerator = e }
}

// WithListItemStyle sets the [lipgloss] style for all list items.
// The default comes from [Theme.ListItemStyle].
// Use [WithListItemStyleFunc] for per-item control.
//
// Example:
//
//	WithListItemStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("252")))
func WithListItemStyle(s lipgloss.Style) ListOption {
	return func(c *listConfig) { c.itemStyle = s }
}

// WithListItemStyleFunc sets a per-item style. When set, it overrides
// [WithListItemStyle].
func WithListItemStyleFunc(fn list.StyleFunc) ListOption {
	return func(c *listConfig) { c.itemStyleFn = fn }
}

// WithListEnumeratorStyle sets the [lipgloss] style for all enumerator markers.
// The default comes from [Theme.ListEnumeratorStyle].
// Use [WithListEnumeratorStyleFunc] for per-item control.
//
// Example:
//
//	WithListEnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("245")))
func WithListEnumeratorStyle(s lipgloss.Style) ListOption {
	return func(c *listConfig) { c.enumStyle = s }
}

// WithListEnumeratorStyleFunc sets a per-item enumerator style. When set, it
// overrides [WithListEnumeratorStyle].
func WithListEnumeratorStyleFunc(fn list.StyleFunc) ListOption {
	return func(c *listConfig) { c.enumStyleFn = fn }
}

// List prints a styled list to the command's output writer. Theme styles apply
// by default; pass [ListOption] values to override.
//
// Example:
//
//	c.List([]string{"Foo", "Bar"}, WithListEnumerator(ListNumbered))
func (c *Context) List(items []string, opts ...ListOption) {
	rt := c.app.Theme()
	cfg := &listConfig{
		enumerator: rt.ListEnumerator(),
		itemStyle:  rt.Style(theme.ListItem),
		enumStyle:  rt.Style(theme.ListEnumerator),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	anyItems := make([]any, 0, len(items))
	for _, item := range items {
		anyItems = append(anyItems, item)
	}

	l := list.New(anyItems...)
	l.Enumerator(cfg.enumerator)

	if cfg.itemStyleFn != nil {
		l.ItemStyleFunc(cfg.itemStyleFn)
	} else {
		l.ItemStyle(cfg.itemStyle)
	}
	if cfg.enumStyleFn != nil {
		l.EnumeratorStyleFunc(cfg.enumStyleFn)
	} else {
		l.EnumeratorStyle(cfg.enumStyle)
	}

	out := writer{w: c.io.Out}
	out.println(l)
}
