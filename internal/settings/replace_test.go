// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstCreationRefusesAppearedDestination(t *testing.T) {
	root := t.TempDir()
	from, to := filepath.Join(root, "prepared.tmp"), filepath.Join(root, "settings.json")
	if err := os.WriteFile(from, []byte("prepared synthetic value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("unrelated synthetic value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(from, to, false); err == nil {
		t.Fatal("appeared destination replaced")
	}
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "unrelated synthetic value" {
		t.Fatal("appeared file changed")
	}
	prepared, err := os.ReadFile(from)
	if err != nil || string(prepared) != "prepared synthetic value" {
		t.Fatal("prepared source lost on refusal")
	}
}
