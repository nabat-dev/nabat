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

package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"nabat.dev/nabat"
)

// Sentinel errors for [errors.Is] checks.
var (
	// ErrNilOption is returned when a nil [Option] is passed to [New].
	ErrNilOption = errors.New("nabat/logging: option is nil")
	// ErrNilHandler is returned when [WithHandler] receives a nil handler.
	ErrNilHandler = errors.New("nabat/logging: handler is nil")
	// ErrUnknownLevel is returned when [ParseLevel] does not recognize
	// the input string.
	ErrUnknownLevel = errors.New("nabat/logging: unknown log level")
)

type config struct {
	level       slog.Level
	handler     slog.Handler
	verboseFlag string
	levelFlag   string
	setDefault  bool
	timestamp   bool
}

// Option configures the logging extension.
type Option interface {
	applyToConfig(*config) error
}

type optionFn func(*config) error

func (f optionFn) applyToConfig(c *config) error { return f(c) }

// WithLevel sets the base log level (default [slog.LevelInfo]).
func WithLevel(l slog.Level) Option {
	return optionFn(func(c *config) error {
		c.level = l
		return nil
	})
}

// WithHandler installs a custom slog handler. The extension wraps it in a
// dynamic-level adjuster so verbose/level flag changes still apply.
//
// Returns an error if h is nil.
func WithHandler(h slog.Handler) Option {
	return optionFn(func(c *config) error {
		if h == nil {
			return fmt.Errorf("nabat/logging: WithHandler: %w", ErrNilHandler)
		}
		c.handler = h
		return nil
	})
}

// WithVerboseFlag wires a bool flag whose presence flips the level to
// [slog.LevelDebug] for the invocation. The flag must be declared by the user
// via [nabat.WithFlag] (typically on the root command with
// [nabat.WithPersistent]).
func WithVerboseFlag(name string) Option {
	return optionFn(func(c *config) error {
		c.verboseFlag = name
		return nil
	})
}

// WithLevelFlag wires a string flag (debug|info|warn|error) parsed to set the
// per-invocation level. The flag must be declared by the user via
// [nabat.WithFlag].
func WithLevelFlag(name string) Option {
	return optionFn(func(c *config) error {
		c.levelFlag = name
		return nil
	})
}

// WithSetDefault installs the extension's logger as the process-wide
// [slog.Default] via [slog.SetDefault].
func WithSetDefault() Option {
	return optionFn(func(c *config) error {
		c.setDefault = true
		return nil
	})
}

// WithTimestamp enables timestamps on the default styled handler.
// Has no effect when [WithHandler] supplies a custom handler.
func WithTimestamp() Option {
	return optionFn(func(c *config) error {
		c.timestamp = true
		return nil
	})
}

type extension struct {
	cfg config
}

func (e *extension) String() string { return "logging" }

func (e *extension) Init(app nabat.AppSurface) error {
	cfg := e.cfg

	var slogLogger *slog.Logger
	if cfg.handler != nil {
		lv := new(slog.LevelVar)
		lv.Set(cfg.level)
		slogLogger = slog.New(&levelGate{level: lv, inner: cfg.handler})

		if err := app.OnPreRun(func(c *nabat.Context) error {
			lv.Set(resolveLevel(c, cfg))
			return nil
		}); err != nil {
			return err
		}
	} else {
		lv := new(slog.LevelVar)
		lv.Set(cfg.level)
		h := NewHandler(app.IO().ErrOut, HandlerOptions{
			Level:     lv,
			Styles:    FromTheme(app.Theme()),
			Timestamp: cfg.timestamp,
		})
		slogLogger = slog.New(h)

		if err := app.OnPreRun(func(c *nabat.Context) error {
			lv.Set(resolveLevel(c, cfg))
			return nil
		}); err != nil {
			return err
		}
	}

	app.SetLogger(slogLogger)
	if cfg.setDefault {
		slog.SetDefault(slogLogger)
	}
	return nil
}

func resolveLevel(c *nabat.Context, cfg config) slog.Level {
	level := cfg.level
	if cfg.verboseFlag != "" {
		if v, err := nabat.BindAs[bool](c, cfg.verboseFlag); err == nil && c.Explicit(cfg.verboseFlag) && v {
			level = slog.LevelDebug
		}
	}
	if cfg.levelFlag != "" {
		if s, err := nabat.BindAs[string](c, cfg.levelFlag); err == nil && c.Explicit(cfg.levelFlag) && s != "" {
			if parsed, parseErr := ParseLevel(s); parseErr == nil {
				level = parsed
			}
		}
	}
	return level
}

// New returns a [nabat.Extension] that installs an opinionated logger.
func New(opts ...Option) (nabat.Extension, error) {
	cfg := config{level: slog.LevelInfo}
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("nabat/logging: option at index %d: %w", i, ErrNilOption)
		}
		if err := opt.applyToConfig(&cfg); err != nil {
			return nil, fmt.Errorf("nabat/logging: option at index %d: %w", i, err)
		}
	}
	return &extension{cfg: cfg}, nil
}

// ParseLevel parses a log level name (case-insensitive): debug, info, warn, error.
// On failure, the error wraps [ErrUnknownLevel].
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("nabat/logging: parse level %q: %w", s, ErrUnknownLevel)
	}
}

// levelGate wraps a user-provided [slog.Handler] with dynamic level filtering.
type levelGate struct {
	level *slog.LevelVar
	inner slog.Handler
}

func (g *levelGate) Enabled(_ context.Context, level slog.Level) bool {
	return level >= g.level.Level()
}

func (g *levelGate) Handle(ctx context.Context, record slog.Record) error {
	return g.inner.Handle(ctx, record)
}

func (g *levelGate) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelGate{level: g.level, inner: g.inner.WithAttrs(attrs)}
}

func (g *levelGate) WithGroup(name string) slog.Handler {
	return &levelGate{level: g.level, inner: g.inner.WithGroup(name)}
}
