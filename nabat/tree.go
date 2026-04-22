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
	"charm.land/lipgloss/v2/tree"

	"nabat.dev/theme"
)

// Enumerator presets for use with [WithTreeEnumerator].
// These are aliases for lipgloss tree enumerators, so callers do not need to
// import the tree package directly.
//
// Available presets:
//
//   - [TreeDefaultEnumerator] — classic box-drawing (├── / └──) (default)
//   - [TreeRoundedEnumerator] — rounded box-drawing (╭── / ╰──)

// TreeDefaultEnumerator returns the classic box-drawing tree enumerator
// (├── / └──). It is the default enumerator used by [Context.Tree].
func TreeDefaultEnumerator() TreeEnumerator { return tree.DefaultEnumerator }

// TreeRoundedEnumerator returns a rounded box-drawing tree enumerator
// (╭── / ╰──).
func TreeRoundedEnumerator() TreeEnumerator { return tree.RoundedEnumerator }

// TreeDefaultIndenter returns the default indentation function.
// It draws a vertical line (│) connecting siblings and blank space for the last
// child.
func TreeDefaultIndenter() TreeIndenter { return tree.DefaultIndenter }

// TreeChildren is the type passed to tree style functions.
type TreeChildren = tree.Children

// TreeEnumerator is the function signature for custom tree enumerators.
type TreeEnumerator = tree.Enumerator

// TreeIndenter is the function signature for custom tree indenters.
type TreeIndenter = tree.Indenter

// TreeStyleFunc is a function that determines the style of a tree node based
// on the children set and the current index.
type TreeStyleFunc = tree.StyleFunc

// TreeNode represents a node in a tree structure.
// Use it to build nested trees without importing the tree package directly.
//
// Tree construction recurses on the calling goroutine's stack; pathologically
// deep trees (tens of thousands of levels) may overflow it. Typical CLI uses
// (filesystem listings, dependency graphs, command trees) are unaffected.
//
// Example:
//
//	nodes := []TreeNode{
//		{Value: ".git"},
//		{Value: "cmd/", Children: []TreeNode{
//			{Value: "root.go"},
//			{Value: "serve.go"},
//		}},
//		{Value: "main.go"},
//	}
//	c.Tree("myproject", nodes)
type TreeNode struct {
	Value    string
	Children []TreeNode
}

type treeConfig struct {
	enumerator      tree.Enumerator
	indenter        tree.Indenter
	rootStyle       *lipgloss.Style
	itemStyle       lipgloss.Style
	itemStyleFn     tree.StyleFunc
	enumStyle       lipgloss.Style
	enumStyleFn     tree.StyleFunc
	indenterStyle   *lipgloss.Style
	indenterStyleFn tree.StyleFunc
	width           int
}

// TreeOption configures the [Context.Tree] output method.
// Pass one or more options to customize the enumerator, indenter, and styling.
//
// Example:
//
//	c.Tree("root", nodes,
//		WithTreeEnumerator(TreeRoundedEnumerator()),
//		WithTreeItemStyle(lipgloss.NewStyle().Bold(true)),
//	)
type TreeOption func(*treeConfig)

// WithTreeEnumerator sets the enumerator used to prefix each tree node.
// Pass one of [TreeDefaultEnumerator] or [TreeRoundedEnumerator], or a
// custom [TreeEnumerator] function.
// The default is [TreeDefaultEnumerator].
//
// Example:
//
//	WithTreeEnumerator(TreeRoundedEnumerator())
func WithTreeEnumerator(e tree.Enumerator) TreeOption {
	return func(c *treeConfig) { c.enumerator = e }
}

// WithTreeIndenter sets the indenter used to draw connectors between sibling
// nodes.
// The default is [TreeDefaultIndenter].
//
// Example:
//
//	WithTreeIndenter(func(children TreeChildren, index int) string {
//		return "→ "
//	})
func WithTreeIndenter(ind tree.Indenter) TreeOption {
	return func(c *treeConfig) { c.indenter = ind }
}

// WithTreeRootStyle sets the [lipgloss] style for the root node.
//
// Example:
//
//	WithTreeRootStyle(lipgloss.NewStyle().Bold(true))
func WithTreeRootStyle(s lipgloss.Style) TreeOption {
	return func(c *treeConfig) { c.rootStyle = &s }
}

// WithTreeItemStyle sets the [lipgloss] style for all tree items.
// The default comes from [Theme.TreeItemStyle].
// Use [WithTreeItemStyleFunc] for per-node control.
//
// Example:
//
//	WithTreeItemStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("252")))
func WithTreeItemStyle(s lipgloss.Style) TreeOption {
	return func(c *treeConfig) { c.itemStyle = s }
}

// WithTreeItemStyleFunc sets a per-node item style function.
// When set, it overrides [WithTreeItemStyle].
//
// Example:
//
//	WithTreeItemStyleFunc(func(_ TreeChildren, i int) lipgloss.Style {
//		if i == 0 {
//			return lipgloss.NewStyle().Bold(true)
//		}
//		return lipgloss.NewStyle()
//	})
func WithTreeItemStyleFunc(fn tree.StyleFunc) TreeOption {
	return func(c *treeConfig) { c.itemStyleFn = fn }
}

// WithTreeEnumeratorStyle sets the [lipgloss] style for all enumerator markers
// (├──, └──, etc.).
// The default comes from [Theme.TreeEnumeratorStyle].
// Use [WithTreeEnumeratorStyleFunc] for per-node control.
//
// Example:
//
//	WithTreeEnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("245")))
func WithTreeEnumeratorStyle(s lipgloss.Style) TreeOption {
	return func(c *treeConfig) { c.enumStyle = s }
}

// WithTreeEnumeratorStyleFunc sets a per-node enumerator style function.
// When set, it overrides [WithTreeEnumeratorStyle].
//
// Example:
//
//	WithTreeEnumeratorStyleFunc(func(_ TreeChildren, i int) lipgloss.Style {
//		if i == 0 {
//			return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
//		}
//		return lipgloss.NewStyle()
//	})
func WithTreeEnumeratorStyleFunc(fn tree.StyleFunc) TreeOption {
	return func(c *treeConfig) { c.enumStyleFn = fn }
}

// WithTreeIndenterStyle sets the [lipgloss] style for all indentation connectors
// (│, etc.).
//
// Example:
//
//	WithTreeIndenterStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("245")))
func WithTreeIndenterStyle(s lipgloss.Style) TreeOption {
	return func(c *treeConfig) { c.indenterStyle = &s }
}

// WithTreeIndenterStyleFunc sets a per-node indenter style function.
// When set, it overrides [WithTreeIndenterStyle].
//
// Example:
//
//	WithTreeIndenterStyleFunc(func(_ TreeChildren, i int) lipgloss.Style {
//		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
//	})
func WithTreeIndenterStyleFunc(fn tree.StyleFunc) TreeOption {
	return func(c *treeConfig) { c.indenterStyleFn = fn }
}

// WithTreeWidth sets a fixed total width for the tree in columns.
// Items will be padded to account for the entire width.
// A value of 0 or less is ignored.
//
// Example:
//
//	WithTreeWidth(80)
func WithTreeWidth(w int) TreeOption {
	return func(c *treeConfig) { c.width = w }
}

// Tree prints a styled tree to the command's output writer.
// It applies the current [Theme] tree styles by default. Pass [TreeOption]
// values to override the enumerator, indenter, or styling.
//
// Example:
//
//	c.Tree("myproject", []TreeNode{
//		{Value: ".git"},
//		{Value: "cmd/", Children: []TreeNode{
//			{Value: "root.go"},
//			{Value: "serve.go"},
//		}},
//		{Value: "main.go"},
//	})
//
//	c.Tree("root", nodes,
//		WithTreeEnumerator(TreeRoundedEnumerator()),
//		WithTreeRootStyle(lipgloss.NewStyle().Bold(true)),
//	)
func (c *Context) Tree(root string, children []TreeNode, opts ...TreeOption) {
	rt := c.app.Theme()
	cfg := &treeConfig{
		enumerator: tree.DefaultEnumerator,
		indenter:   tree.DefaultIndenter,
		itemStyle:  rt.Style(theme.TreeItem),
		enumStyle:  rt.Style(theme.TreeEnumerator),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	t := tree.Root(root)
	buildChildren(t, children)

	t.Enumerator(cfg.enumerator)
	t.Indenter(cfg.indenter)

	if cfg.rootStyle != nil {
		t.RootStyle(*cfg.rootStyle)
	}

	if cfg.itemStyleFn != nil {
		t.ItemStyleFunc(cfg.itemStyleFn)
	} else {
		t.ItemStyle(cfg.itemStyle)
	}

	if cfg.enumStyleFn != nil {
		t.EnumeratorStyleFunc(cfg.enumStyleFn)
	} else {
		t.EnumeratorStyle(cfg.enumStyle)
	}

	if cfg.indenterStyleFn != nil {
		t.IndenterStyleFunc(cfg.indenterStyleFn)
	} else if cfg.indenterStyle != nil {
		t.IndenterStyle(*cfg.indenterStyle)
	}

	if cfg.width > 0 {
		t.Width(cfg.width)
	}

	out := writer{w: c.io.Out}
	out.println(t)
}

func buildChildren(t *tree.Tree, nodes []TreeNode) {
	for _, node := range nodes {
		if len(node.Children) == 0 {
			t.Child(node.Value)
		} else {
			sub := tree.Root(node.Value)
			buildChildren(sub, node.Children)
			t.Child(sub)
		}
	}
}
