// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestScanBudgetAndCancellationBeforeResolution(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 30; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("Fixture%02d.lnk", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := DefaultScanOptions()
	options.MaxEntries = 10
	options.MaxApplications = 5
	apps, problems := IndexApplicationsWithOptions(context.Background(), []string{root}, nil, options)
	if len(apps) > 5 || !hasScanError(problems, ErrScanBudget) {
		t.Fatal("scan budget did not bound applications")
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	apps, problems = IndexApplicationsWithOptions(ctx, []string{root}, func(string) (LaunchIdentity, error) { calls++; cancel(); return LaunchIdentity{}, nil }, DefaultScanOptions())
	if calls != 1 || len(apps) > 1 || !hasScanError(problems, context.Canceled) {
		t.Fatal("cancellation allowed further shortcut resolution")
	}
	checks := 0
	options = DefaultScanOptions()
	options.AllowPath = func(string) error { checks++; return errors.New("synthetic path refused") }
	apps, problems = IndexApplicationsWithOptions(context.Background(), []string{root}, nil, options)
	if len(apps) != 0 || checks != 1 || len(problems) == 0 {
		t.Fatal("path refusal did not stop traversal before descendants")
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	checks = 0
	IndexApplicationsWithOptions(ctx, []string{root}, nil, options)
	if checks != 0 {
		t.Fatal("already cancelled scan checked filesystem paths")
	}
}

func hasScanError(problems []error, want error) bool {
	for _, err := range problems {
		if errors.Is(err, want) {
			return true
		}
	}
	return false
}

func TestScanQueueCoalescesAndRejectsStaleResults(t *testing.T) {
	var queue ScanQueue
	old, start := queue.Request()
	if !start {
		t.Fatal("first worker did not start")
	}
	for i := 0; i < 100; i++ {
		if _, start := queue.Request(); start {
			t.Fatal("overlapping worker started")
		}
	}
	if publish, restart := queue.Finish(old); publish || !restart || queue.Running {
		t.Fatal("old result published or latest request not queued")
	}
	latest, start := queue.Request()
	if !start || latest <= old {
		t.Fatal("latest worker not started")
	}
	if publish, restart := queue.Finish(old); publish || restart || !queue.Running {
		t.Fatal("duplicate old completion affected current worker")
	}
	if publish, restart := queue.Finish(latest); !publish || restart || queue.Running {
		t.Fatal("current result did not publish")
	}
}
