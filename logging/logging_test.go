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

package logging_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/logging"
	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func TestLoggingDefaultLevel(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test",
		nabat.WithIO(io),
		nabat.WithExtension(logging.New()),
	)
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Logger().Info("info log")
		c.Logger().Debug("debug log") // suppressed at info
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	got := stderr.String()
	assert.Contains(t, got, "info log")
	assert.NotContains(t, got, "debug log")
}

func TestLoggingWithVerboseFlag(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test",
		nabat.WithIO(io),
		nabat.WithFlag("verbose", false, nabat.WithPersistent()),
		nabat.WithExtension(logging.New(logging.WithVerboseFlag("verbose"))),
	)
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Logger().Debug("debugging", "id", 1)
		c.Logger().Info("info log")
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.NotContains(t, stderr.String(), "debugging")
	assert.Contains(t, stderr.String(), "info log")

	stderr.Reset()
	require.NoError(t, nabattest.Run(t, app, []string{"run", "--verbose"}))
	assert.Contains(t, stderr.String(), "debugging")
}

func TestLoggingWithLevelFlag(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test",
		nabat.WithIO(io),
		nabat.WithFlag("log-level", "warn", nabat.WithPersistent()),
		nabat.WithExtension(logging.New(
			logging.WithLevel(slog.LevelWarn),
			logging.WithLevelFlag("log-level"),
		)),
	)
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Logger().Info("info")
		c.Logger().Warn("warn")
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.NotContains(t, stderr.String(), "info")
	assert.Contains(t, stderr.String(), "warn")

	stderr.Reset()
	require.NoError(t, nabattest.Run(t, app, []string{"run", "--log-level", "debug"}))
	assert.Contains(t, stderr.String(), "info")
	assert.Contains(t, stderr.String(), "warn")
}

func TestLoggingCustomHandler(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test",
		nabat.WithIO(io),
		nabat.WithExtension(logging.New(logging.WithHandler(slog.NewTextHandler(&logs, nil)))),
	)
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Logger().Info("custom", "id", 7)
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, logs.String(), "custom")
	assert.Contains(t, logs.String(), "id=7")
}

func TestLoggingSetDefault(t *testing.T) {
	oldDefault := slog.Default()
	t.Cleanup(func() { slog.SetDefault(oldDefault) })

	var logs bytes.Buffer
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test",
		nabat.WithIO(io),
		nabat.WithExtension(logging.New(
			logging.WithHandler(slog.NewTextHandler(&logs, nil)),
			logging.WithSetDefault(),
		)),
	)
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		slog.Info("from default", "ok", true)
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, logs.String(), "from default")
}

func TestLoggingWithTimestamp(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test",
		nabat.WithIO(io),
		nabat.WithExtension(logging.New(logging.WithTimestamp())),
	)
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Logger().Info("hello")
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stderr.String(), "hello")
}

func TestLoggingNilHandlerRejected(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithExtension(logging.New(logging.WithHandler(nil))))
	require.Error(t, err)
	require.ErrorIs(t, err, logging.ErrNilHandler)
}

func TestLoggingRejectsNilOption(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithExtension(logging.New(nil)))
	require.Error(t, err)
	require.ErrorIs(t, err, logging.ErrNilOption)
}
