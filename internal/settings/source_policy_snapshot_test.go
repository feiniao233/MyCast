// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewSourcePolicyRejectsOldDesktopSnapshotFingerprint(t *testing.T) {
	store, value, now := snapshotFixture(t)
	roots := []string{"synthetic-root"}
	oldData, _ := json.Marshal(struct {
		Roots, Shallow []string
		Shell          bool
	}{roots, nil, true})
	value.Fingerprint = fmt.Sprintf("%x", sha256.Sum256(oldData))
	if err := store.Save(value, true, store.Epoch(), now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(SourceFingerprint(roots, nil, true), now); err == nil {
		t.Fatal("old broad AppsFolder display cache survived source policy change")
	}
}
