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

// Command status shows five patterns for the Nabat Status and Spinner APIs:
// sequential labeled steps (database migrations), parallel priority-sorted
// uploads, health checks with icon overrides, Kubernetes-style event feeds,
// and header-only updates with the simple Spinner.
//
// Run any subcommand to see the live display:
//
//	go run ./examples/status migrate
//	go run ./examples/status upload
//	go run ./examples/status healthcheck
//	go run ./examples/status events
//	go run ./examples/status deploy
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"nabat.dev/nabat"
)

func main() {
	app, err := nabat.New("status",
		nabat.WithDescription("Status and Spinner display demo"),

		// Sequential work: each step starts after the previous one finishes.
		nabat.WithCommand("migrate",
			nabat.WithDescription("Simulate sequential database migrations"),
			nabat.WithRun(runMigrate),
		),

		// Parallel work: uploads run concurrently, sorted by size.
		nabat.WithCommand("upload",
			nabat.WithDescription("Simulate parallel file uploads"),
			nabat.WithRun(runUpload),
		),

		// Mixed results: some rows succeed, some warn, some error, with custom icons.
		nabat.WithCommand("healthcheck",
			nabat.WithDescription("Simulate a multi-service health check"),
			nabat.WithRun(runHealthCheck),
		),

		// Event feed: Kubernetes-style keyed events with Label to dedup.
		nabat.WithCommand("events",
			nabat.WithDescription("Simulate a Kubernetes event feed"),
			nabat.WithRun(runEvents),
		),

		// Header-only: the simple Spinner with SetText updates, no rows.
		nabat.WithCommand("deploy",
			nabat.WithDescription("Simulate a phased deploy (header-only Spinner)"),
			nabat.WithRun(runDeploy),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = app.Run(context.Background()); err != nil {
		os.Exit(1)
	}
}

// runMigrate shows sequential Status rows. Each migration is keyed by version
// and uses Label to show a human-readable name in the first column.
// Column headers are set with WithColumns.
func runMigrate(c *nabat.Context) error {
	type migration struct {
		version string
		name    string
		delay   time.Duration
		fail    bool
	}
	migrations := []migration{
		{"20240101", "create_users_table", 300 * time.Millisecond, false},
		{"20240115", "add_email_index", 200 * time.Millisecond, false},
		{"20240201", "add_audit_log", 400 * time.Millisecond, false},
		{"20240210", "backfill_timestamps", 600 * time.Millisecond, false},
		{"20240301", "rename_status_column", 250 * time.Millisecond, true},
	}

	return c.Status(func(st *nabat.Status) error {
		for _, m := range migrations {
			row := st.Row(m.version).Label(m.name).Set("Applying")

			time.Sleep(m.delay)

			if m.fail {
				row.Set("column rename blocked by constraint").Error()
				return fmt.Errorf("migration %s: column rename blocked by constraint", m.version)
			}
			row.Set("Applied").Success()
		}
		return nil
	},
		nabat.WithTitle("Running migrations"),
		nabat.WithColumns("MIGRATION", "STATUS"),
	)
}

// runUpload shows parallel Status rows with Priority sorting. Larger files get
// a lower priority number so they appear at the top while uploading.
func runUpload(c *nabat.Context) error {
	type file struct {
		name   string
		sizeMB int
		delay  time.Duration
		fail   bool
	}
	files := []file{
		{"report-q1.pdf", 2, 700 * time.Millisecond, false},
		{"report-q2.pdf", 3, 500 * time.Millisecond, false},
		{"report-q3.pdf", 1, 400 * time.Millisecond, false},
		{"archive.tar.gz", 48, 1100 * time.Millisecond, false},
		{"backup-2026.sql", 12, 900 * time.Millisecond, true},
	}

	return c.Status(func(st *nabat.Status) error {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		// Create all rows up front so Priority sorting is stable from the start.
		for i, f := range files {
			st.Row(f.name).
				Label(f.name).
				Set(fmt.Sprintf("%d MB", f.sizeMB), "Queued").
				Priority(i) // Larger index = lower priority.
		}

		for _, f := range files {
			wg.Go(func() {
				row := st.Row(f.name).Set(fmt.Sprintf("%d MB", f.sizeMB), "Uploading")
				time.Sleep(f.delay)
				if f.fail {
					row.Set(fmt.Sprintf("%d MB", f.sizeMB), "connection reset by peer").Error()
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("upload %s: connection reset by peer", f.name)
					}
					mu.Unlock()
					return
				}
				row.Set(fmt.Sprintf("%d MB", f.sizeMB), "Uploaded").Success()
			})
		}
		wg.Wait()
		return firstErr
	},
		nabat.WithTitle("Uploading assets"),
		nabat.WithColumns("FILE", "SIZE", "STATUS"),
	)
}

// runHealthCheck shows mixed-result rows with per-row Icon overrides.
// The caller controls the completion icon via SetCompletion.
func runHealthCheck(c *nabat.Context) error {
	type service struct {
		name    string
		url     string
		delay   time.Duration
		outcome string // "ok", "warn", "error"
	}
	services := []service{
		{"api-gateway", "https://api.example.com/health", 200 * time.Millisecond, "ok"},
		{"auth-service", "https://auth.example.com/health", 350 * time.Millisecond, "ok"},
		{"database", "postgres://db.example.com:5432", 150 * time.Millisecond, "ok"},
		{"cache", "redis://cache.example.com:6379", 400 * time.Millisecond, "warn"},
		{"queue", "amqp://mq.example.com:5672", 500 * time.Millisecond, "error"},
		{"object-store", "https://s3.example.com", 300 * time.Millisecond, "ok"},
	}

	return c.Status(func(st *nabat.Status) error {
		var wg sync.WaitGroup
		var hasWarn, hasError bool
		var mu sync.Mutex

		for _, svc := range services {
			wg.Go(func() {
				row := st.Row(svc.name).Label(svc.name).Set(svc.url, "Checking")
				time.Sleep(svc.delay)
				switch svc.outcome {
				case "ok":
					row.Set(svc.url, "Healthy").Success()
				case "warn":
					row.Set(svc.url, "Degraded: high latency (p99=1.8s)").Icon("?").Warn()
					mu.Lock()
					hasWarn = true
					mu.Unlock()
				case "error":
					row.Set(svc.url, "Unreachable: connection refused").Error()
					mu.Lock()
					hasError = true
					mu.Unlock()
				}
			})
		}
		wg.Wait()

		// Caller controls the header icon explicitly.
		switch {
		case hasError:
			st.SetCompletion(nabat.RowError)
		case hasWarn:
			st.SetCompletion(nabat.RowWarning)
		default:
			st.SetCompletion(nabat.RowSuccess)
		}
		return nil
	},
		nabat.WithTitle("Checking services"),
		nabat.WithColumns("SERVICE", "ENDPOINT", "STATUS"),
	)
}

// runEvents shows the Deployah-like pattern: Kubernetes events arrive as a
// stream; each event has a UID used as the row key. The Label is set to the
// object name so repeated events for the same object update the same row.
// Rows without a terminal state after the feed ends are marked Done via Hide.
func runEvents(c *nabat.Context) error {
	type event struct {
		uid     string
		object  string
		reason  string
		message string
		delay   time.Duration
		state   nabat.RowState
	}

	events := []event{
		{"uid-1", "deployment/api", "Scheduled", "pod assigned to node-1", 200 * time.Millisecond, nabat.RowActive},
		{"uid-2", "deployment/worker", "Scheduled", "pod assigned to node-2", 100 * time.Millisecond, nabat.RowActive},
		{"uid-1", "deployment/api", "Pulled", "image pulled successfully", 400 * time.Millisecond, nabat.RowActive},
		{"uid-3", "deployment/cache", "Failed", "back-off restarting failed container", 300 * time.Millisecond, nabat.RowError},
		{"uid-1", "deployment/api", "Started", "container started", 200 * time.Millisecond, nabat.RowSuccess},
		{"uid-4", "deployment/metrics", "Scheduled", "pod assigned to node-3", 150 * time.Millisecond, nabat.RowActive},
		{"uid-2", "deployment/worker", "Started", "container started", 250 * time.Millisecond, nabat.RowSuccess},
		{"uid-4", "deployment/metrics", "Started", "container started", 300 * time.Millisecond, nabat.RowSuccess},
	}

	return c.Status(func(st *nabat.Status) error {
		for _, ev := range events {
			time.Sleep(ev.delay)
			row := st.Row(ev.uid).Label(ev.object).Set(ev.reason, ev.message)
			switch ev.state {
			case nabat.RowSuccess:
				row.Success()
			case nabat.RowError:
				row.Error()
			case nabat.RowWarning:
				row.Warn()
			}
		}
		return nil
	},
		nabat.WithTitle("Kubernetes events"),
		nabat.WithColumns("OBJECT", "REASON", "MESSAGE"),
	)
}

// runDeploy demonstrates the simple header-only Spinner: a single animated
// line with SetText updates and no rows. Use this when you just need "working"
// feedback without tracking individual items.
func runDeploy(c *nabat.Context) error {
	return c.Spinner(func(sp *nabat.Spinner) error {
		time.Sleep(500 * time.Millisecond)
		sp.SetText("Building image...")
		time.Sleep(800 * time.Millisecond)
		sp.SetText("Pushing image...")
		time.Sleep(600 * time.Millisecond)
		sp.SetText("Rolling out pods...")
		time.Sleep(700 * time.Millisecond)
		sp.SetText("Done")
		return nil
	}, nabat.WithTitle("Connecting to cluster..."))
}
