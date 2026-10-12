// SPDX-License-Identifier: AGPL-3.0-or-later

package core

import (
	"strings"
	"sync"
	"unicode/utf8"
)

// History is a bounded, in-memory, text-only clipboard history. It never writes
// content to disk. Methods are safe for concurrent use. Do not copy a History
// after first use. Clearing releases references but does not guarantee secure
// erasure of memory managed by the Go runtime.
type History struct {
	mu            sync.RWMutex
	maxEntries    int
	maxEntryBytes int
	entries       []string
}

// NewHistory creates a newest-first history with the supplied limits. A
// nonpositive limit disables storage. maxEntryBytes measures UTF-8 bytes,
// not characters. A value is accepted whole or rejected; it is never truncated.
func NewHistory(maxEntries, maxEntryBytes int) *History {
	return &History{maxEntries: maxEntries, maxEntryBytes: maxEntryBytes}
}

// Add stores valid UTF-8 text, moving an exact duplicate to the newest position.
// It reports whether the history changed. Empty text, oversized values, invalid
// UTF-8, disabled storage, and repeats of the newest entry leave it unchanged.
// Whitespace and line endings are preserved exactly.
func (history *History) Add(text string) bool {
	if text == "" || !utf8.ValidString(text) {
		return false
	}
	history.mu.Lock()
	defer history.mu.Unlock()
	if history.maxEntries <= 0 || history.maxEntryBytes <= 0 || len(text) > history.maxEntryBytes {
		return false
	}
	for index, entry := range history.entries {
		if entry != text {
			continue
		}
		if index == 0 {
			return false
		}
		copy(history.entries[1:index+1], history.entries[:index])
		history.entries[0] = entry
		return true
	}
	// Clone so a short substring cannot keep an arbitrarily large source
	// string alive and evade the intended memory bound.
	text = strings.Clone(text)
	if len(history.entries) < history.maxEntries {
		history.entries = append(history.entries, "")
	}
	copy(history.entries[1:], history.entries[:len(history.entries)-1])
	history.entries[0] = text
	return true
}

// Entries returns a detached snapshot in newest-first order.
func (history *History) Entries() []string {
	history.mu.RLock()
	defer history.mu.RUnlock()
	entries := make([]string, len(history.entries))
	copy(entries, history.entries)
	return entries
}

// Len returns the current entry count.
func (history *History) Len() int {
	history.mu.RLock()
	defer history.mu.RUnlock()
	return len(history.entries)
}

// Clear removes all entries. Existing snapshots returned by Entries are not
// modified and may still hold content until the caller releases them.
func (history *History) Clear() {
	history.mu.Lock()
	defer history.mu.Unlock()
	for index := range history.entries {
		history.entries[index] = ""
	}
	history.entries = nil
}
