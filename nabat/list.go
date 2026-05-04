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

// Enumerator presets for use with [WithListEnumerator].
// These are aliases for lipgloss list enumerators, so callers do not need to
// import the list package directly.
//
// Available presets:
//
//   - [ListBullet] — bullet point (•) (default)
//   - [ListDash] — dash (-)
//   - [ListAsterisk] — asterisk (*)
//   - [ListNumbered] — numbered (1. 2. 3.)
//   - [ListRoman] — Roman numerals (I. II. III.)
//   - [ListAlphabet] — alphabetical (A. B. C.)
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

// ListOption configures the [Context.List] output method.
// Pass one or more options to customize the enumerator and styling.
//
// Example:
//
//	c.List(items,
//		WithListEnumerator(ListNumbered),
//		WithListItemStyle(lipgloss.NewStyle().Bold(true)),
//	)
type ListOption func(*listConfig)

// WithListEnumerator sets the enumerator used to prefix each list item.
// Pass one of the ListBullet, ListDash, ListNumbered, ListRoman, or
// ListAlphabet presets, or a custom [ListEnumerator] function.
// The default is [ListBullet].
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

// WithListItemStyleFunc sets a per-item style function.
// When set, it overrides [WithListItemStyle].
//
// Example:
//
//	WithListItemStyleFunc(func(_ ListItems, i int) lipgloss.Style {
//		if i == 0 {
//			return lipgloss.NewStyle().Bold(true)
//		}
//		return lipgloss.NewStyle()
//	})
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

// WithListEnumeratorStyleFunc sets a per-item enumerator style function.
// When set, it overrides [WithListEnumeratorStyle].
//
// Example:
//
//	WithListEnumeratorStyleFunc(func(_ ListItems, i int) lipgloss.Style {
//		if i == 0 {
//			return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
//		}
//		return lipgloss.NewStyle()
//	})
func WithListEnumeratorStyleFunc(fn list.StyleFunc) ListOption {
	return func(c *listConfig) { c.enumStyleFn = fn }
}

// List prints a styled list to the command's output writer.
// It applies the current [Theme] list styles by default. Pass [ListOption]
// values to override the enumerator or styling.
//
// Example:
//
//	c.List([]string{"Foo", "Bar", "Baz"})
//
//	c.List(items,
//		WithListEnumerator(ListRoman),
//		WithListEnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))),
//	)
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
