// SPDX-License-Identifier: AGPL-3.0-or-later

package core

import (
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// History is a bounded, in-memory, text-only clipboard history. It never writes
// content to disk. Methods are safe for concurrent use. Do not copy a History
// after first use. Clearing releases references but does not guarantee secure
// erasure of memory managed by the Go runtime.
type History struct {
	mu            sync.Mutex
	maxEntries    int
	maxEntryBytes int
	entries       []historyEntry
	retention     time.Duration
	now           func() time.Time
}

type historyEntry struct {
	text   string
	copied time.Time
}

// NewHistory creates a newest-first history with the supplied limits. A
// nonpositive limit disables storage. maxEntryBytes measures UTF-8 bytes,
// not characters. A value is accepted whole or rejected; it is never truncated.
func NewHistory(maxEntries, maxEntryBytes int) *History {
	return &History{maxEntries: maxEntries, maxEntryBytes: maxEntryBytes, now: time.Now}
}

// Add stores valid UTF-8 text, moving an exact duplicate to the newest position.
// It reports whether the history changed. Empty text, oversized values, invalid
// UTF-8 and disabled storage leave it unchanged. A newest duplicate refreshes
// its expiry clock without changing content/order or reporting a change.
// Whitespace and line endings are preserved exactly.
func (history *History) Add(text string) bool {
	if text == "" || !utf8.ValidString(text) {
		return false
	}
	history.mu.Lock()
	defer history.mu.Unlock()
	now := history.clock()
	history.expire(now)
	if history.maxEntries <= 0 || history.maxEntryBytes <= 0 || len(text) > history.maxEntryBytes {
		return false
	}
	for index, entry := range history.entries {
		if entry.text != text {
			continue
		}
		if index == 0 {
			history.entries[0].copied = now
			return false
		}
		copy(history.entries[1:index+1], history.entries[:index])
		history.entries[0] = historyEntry{text: entry.text, copied: now}
		return true
	}
	// Clone so a short substring cannot keep an arbitrarily large source
	// string alive and evade the intended memory bound.
	text = strings.Clone(text)
	if len(history.entries) < history.maxEntries {
		history.entries = append(history.entries, historyEntry{})
	}
	copy(history.entries[1:], history.entries[:len(history.entries)-1])
	history.entries[0] = historyEntry{text: text, copied: now}
	return true
}

// Entries returns a detached snapshot in newest-first order.
func (history *History) Entries() []string {
	history.mu.Lock()
	defer history.mu.Unlock()
	history.expire(history.clock())
	entries := make([]string, len(history.entries))
	for i, entry := range history.entries {
		entries[i] = entry.text
	}
	return entries
}

// Len returns the current entry count.
func (history *History) Len() int {
	history.mu.Lock()
	defer history.mu.Unlock()
	history.expire(history.clock())
	return len(history.entries)
}

// Clear removes all entries. Existing snapshots returned by Entries are not
// modified and may still hold content until the caller releases them.
func (history *History) Clear() {
	history.mu.Lock()
	defer history.mu.Unlock()
	for index := range history.entries {
		history.entries[index] = historyEntry{}
	}
	history.entries = nil
}

func (history *History) clock() time.Time {
	if history.now != nil {
		return history.now()
	}
	return time.Now()
}

// SetRetention affects only this in-memory instance. Zero means until exit;
// shorter retention also expires existing entries immediately when applicable.
func (history *History) SetRetention(value time.Duration) bool {
	if value < 0 || value > 24*time.Hour {
		return false
	}
	history.mu.Lock()
	defer history.mu.Unlock()
	history.retention = value
	history.expire(history.clock())
	return true
}

func (history *History) Expire() {
	history.mu.Lock()
	defer history.mu.Unlock()
	history.expire(history.clock())
}
func (history *History) expire(now time.Time) {
	if history.retention == 0 {
		return
	}
	kept := history.entries[:0]
	for _, entry := range history.entries {
		if now.Sub(entry.copied) < history.retention {
			kept = append(kept, entry)
		}
	}
	clear(history.entries[len(kept):])
	history.entries = kept
}

// Delete removes exactly matching text and releases the backing reference.
func (history *History) Delete(text string) bool {
	history.mu.Lock()
	defer history.mu.Unlock()
	history.expire(history.clock())
	for i, entry := range history.entries {
		if entry.text == text {
			copy(history.entries[i:], history.entries[i+1:])
			history.entries[len(history.entries)-1] = historyEntry{}
			history.entries = history.entries[:len(history.entries)-1]
			return true
		}
	}
	return false
}
