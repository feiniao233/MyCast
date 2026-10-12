// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"reflect"
	"testing"
	"time"
)

func TestRetentionExpiryRecopyAndDeletion(t *testing.T) {
	now := time.Unix(1000, 0)
	history := NewHistory(5, 100)
	history.now = func() time.Time { return now }
	history.Add("Dedicated first\r\ntext")
	now = now.Add(5 * time.Minute)
	history.Add("Dedicated second")
	if !history.SetRetention(10 * time.Minute) {
		t.Fatal("valid retention rejected")
	}
	now = now.Add(5 * time.Minute)
	history.Expire()
	if got := history.Entries(); !reflect.DeepEqual(got, []string{"Dedicated second"}) {
		t.Fatal("expired entry retained")
	}
	if history.Add("Dedicated second") {
		t.Fatal("newest duplicate reported content change")
	}
	now = now.Add(6 * time.Minute)
	if history.Len() != 1 {
		t.Fatal("recopy did not refresh expiry")
	}
	if history.Delete("dedicated second") {
		t.Fatal("delete incorrectly case folded text")
	}
	if !history.Delete("Dedicated second") || history.Len() != 0 || history.Delete("Dedicated second") {
		t.Fatal("exact deletion failed")
	}
	history.Add("Dedicated after deletion")
	now = now.Add(11 * time.Minute)
	history.Expire()
	if history.Len() != 0 {
		t.Fatal("expiry did not clear without recording")
	}
	if history.SetRetention(-time.Second) || history.SetRetention(25*time.Hour) {
		t.Fatal("unbounded retention accepted")
	}
}

func TestDefaultSessionRetentionAndShorteningExistingEntries(t *testing.T) {
	now := time.Unix(1000, 0)
	history := NewHistory(2, 100)
	history.now = func() time.Time { return now }
	history.Add("Dedicated session text")
	now = now.Add(24 * time.Hour)
	history.Expire()
	if history.Len() != 1 {
		t.Fatal("default session retention expired")
	}
	history.SetRetention(time.Hour)
	if history.Len() != 0 {
		t.Fatal("shorter retention did not remove old entries")
	}
	history.SetRetention(0)
	history.Add("Dedicated new session text")
	now = now.Add(48 * time.Hour)
	if history.Len() != 1 {
		t.Fatal("session retention could not be restored")
	}
}
