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

// Command spinnerrows shows four patterns for the Nabat spinner live-row API:
// sequential steps (database migrations), parallel work (file uploads), health
// checks with mixed results, and simple header-only updates.
//
// Run any of the four subcommands to see the live display:
//
//	go run ./examples/spinnerrows migrate
//	go run ./examples/spinnerrows upload
//	go run ./examples/spinnerrows healthcheck
//	go run ./examples/spinnerrows deploy
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
	app, err := nabat.New("spinnerrows",
		nabat.WithDescription("Spinner live rows demo"),

		// Sequential work: each step starts after the previous one finishes.
		nabat.WithCommand("migrate",
			nabat.WithDescription("Simulate sequential database migrations"),
			nabat.WithRun(runMigrate),
		),

		// Parallel work: all items start concurrently, each updating its own row.
		nabat.WithCommand("upload",
			nabat.WithDescription("Simulate parallel file uploads"),
			nabat.WithRun(runUpload),
		),

		// Mixed results: some rows succeed, some warn, some error.
		nabat.WithCommand("healthcheck",
			nabat.WithDescription("Simulate a multi-service health check"),
			nabat.WithRun(runHealthCheck),
		),

		// Header-only updates: the classic spinner with no rows.
		nabat.WithCommand("deploy",
			nabat.WithDescription("Simulate a phased deploy (header-only)"),
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

// runMigrate demonstrates sequential rows: each migration is created, updated
// to "Applying", then marked Success or Error before the next one starts.
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

	return c.Spinner("Running migrations", func(sp *nabat.Spinner) error {
		for _, m := range migrations {
			row := sp.Row(m.version)
			row.Set(m.name, "Applying")

			time.Sleep(m.delay)

			if m.fail {
				row.Set(m.name, "column rename blocked by constraint").Error()
				return fmt.Errorf("migration %s: column rename blocked by constraint", m.version)
			}
			row.Set(m.name, "Applied").Success()
		}
		return nil
	})
}

// runUpload demonstrates parallel rows: all uploads start at the same time and
// update their own row independently.
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

	return c.Spinner("Uploading assets", func(sp *nabat.Spinner) error {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for _, f := range files {
			wg.Go(func() {
				row := sp.Row(f.name)
				row.Set(fmt.Sprintf("%d MB", f.sizeMB), "Uploading")
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
	})
}

// runHealthCheck demonstrates mixed results with Warn state.
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

	return c.Spinner("Checking services", func(sp *nabat.Spinner) error {
		var wg sync.WaitGroup
		for _, svc := range services {
			wg.Go(func() {
				row := sp.Row(svc.name)
				row.Set(svc.url, "Checking")
				time.Sleep(svc.delay)
				switch svc.outcome {
				case "ok":
					row.Set(svc.url, "Healthy").Success()
				case "warn":
					row.Set(svc.url, "Degraded: high latency (p99=1.8s)").Warn()
				case "error":
					row.Set(svc.url, "Unreachable: connection refused").Error()
				}
			})
		}
		wg.Wait()
		return nil
	})
}

// runDeploy demonstrates header-only updates (no rows): the classic spinner
// pattern that was the only option before live rows were added.
func runDeploy(c *nabat.Context) error {
	return c.Spinner("Connecting to cluster...", func(sp *nabat.Spinner) error {
		time.Sleep(500 * time.Millisecond)
		sp.SetText("Building image...")
		time.Sleep(800 * time.Millisecond)
		sp.SetText("Pushing image...")
		time.Sleep(600 * time.Millisecond)
		sp.SetText("Rolling out pods...")
		time.Sleep(700 * time.Millisecond)
		sp.SetText("Done")
		return nil
	})
}
