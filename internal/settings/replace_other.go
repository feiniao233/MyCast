//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import "os"

func replaceFile(from, to string, exists bool) error {
	if exists {
		return os.Rename(from, to)
	}
	if err := os.Link(from, to); err != nil {
		return err
	}
	return os.Remove(from)
}
