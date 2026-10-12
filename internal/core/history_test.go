// SPDX-License-Identifier: AGPL-3.0-or-later

package core

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestHistoryBoundedAndDeduplicated(t *testing.T) {
	history := NewHistory(3, 1024)
	for _, value := range []string{"one", "two", "three", "four"} {
		if !history.Add(value) {
			t.Fatalf("failed to add %q", value)
		}
	}
	if got := history.Entries(); !reflect.DeepEqual(got, []string{"four", "three", "two"}) {
		t.Fatalf("bounded entries = %v", got)
	}
	if !history.Add("two") {
		t.Fatal("moving a duplicate should report a change")
	}
	if got := history.Entries(); !reflect.DeepEqual(got, []string{"two", "four", "three"}) {
		t.Fatalf("deduplicated entries = %v", got)
	}
	if history.Add("two") {
		t.Fatal("repeated newest value should not report a change")
	}
	if history.Len() != 3 {
		t.Fatalf("Len = %d, want 3", history.Len())
	}
	// Dedupe is exact; clipboard content is not case folded or trimmed.
	if !history.Add("TWO") || !history.Add(" two\r\n") {
		t.Fatal("distinct clipboard text was incorrectly deduplicated")
	}
}

func TestHistoryUTF8ByteLimit(t *testing.T) {
	history := NewHistory(5, 6)
	if !history.Add("微信") { // Six UTF-8 bytes, two runes.
		t.Fatal("value exactly at the byte limit was rejected")
	}
	for _, value := range []string{"微信箱", "1234567", "", "\xff", "ok\xc0\x80"} {
		if history.Add(value) {
			t.Errorf("invalid or oversized value %q was accepted", value)
		}
	}
	if got := history.Entries(); !reflect.DeepEqual(got, []string{"微信"}) {
		t.Errorf("rejected input changed history: %v", got)
	}
	if !history.Add(" ") {
		t.Error("meaningful whitespace was rejected")
	}
}

func TestHistorySnapshotAndClear(t *testing.T) {
	history := NewHistory(2, 100)
	history.Add("first")
	history.Add("second")
	snapshot := history.Entries()
	snapshot[0] = "modified"
	if got := history.Entries()[0]; got != "second" {
		t.Fatalf("snapshot mutation changed history to %q", got)
	}
	history.Clear()
	if history.Len() != 0 || len(history.Entries()) != 0 {
		t.Fatal("clear retained entries")
	}
	if snapshot[1] != "first" {
		t.Fatal("clear changed a caller-owned snapshot")
	}
	if !history.Add("after clear") || history.Len() != 1 {
		t.Fatal("history cannot be reused after clear")
	}
	history.Clear()
	history.Clear()
}

func TestHistoryDisabledAndSingleEntry(t *testing.T) {
	for _, history := range []*History{NewHistory(0, 100), NewHistory(-1, 100), NewHistory(10, 0), NewHistory(10, -1), {}} {
		if history.Add("text") || history.Len() != 0 {
			t.Fatal("disabled or zero-value history stored text")
		}
		history.Clear()
	}
	history := NewHistory(1, 100)
	history.Add("one")
	history.Add("two")
	if got := history.Entries(); !reflect.DeepEqual(got, []string{"two"}) {
		t.Errorf("single-entry history = %v", got)
	}
}

func TestHistoryConcurrentAccess(t *testing.T) {
	history := NewHistory(20, 100)
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for index := 0; index < 100; index++ {
				history.Add(fmt.Sprintf("worker %d item %d", worker, index))
				history.Entries()
				history.Len()
				if index%25 == 0 {
					history.Clear()
				}
			}
		}(worker)
	}
	workers.Wait()
	if history.Len() > 20 {
		t.Errorf("concurrent use exceeded capacity: %d", history.Len())
	}
	seen := map[string]bool{}
	for _, entry := range history.Entries() {
		if seen[entry] {
			t.Errorf("duplicate value after concurrent use: %q", entry)
		}
		seen[entry] = true
	}
}
