//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/feiniao233/MyCast/internal/core"
)

// IncludeApplicationTarget excludes confirmed ordinary documents/directories,
// never names or arbitrary command arguments. Executable/CLI, unknown file
// associations and unavailable targets remain discoverable. Special/virtual
// links bypass this callback in core. Locality is checked before target stat.
func IncludeApplicationTarget(launch core.LaunchIdentity) bool {
	if !launch.Independent && filepath.IsAbs(launch.Target) && strings.EqualFold(filepath.Ext(launch.Target), ".exe") {
		state, attrs := localApplicationPresence(launch.Target)
		return state != applicationMissing && !(state == applicationPresent && attrs&0x10 != 0)
	}
	return includeApplicationTarget(launch.Target, LocalAppPath, os.Lstat)
}

func includeApplicationTarget(target string, allow func(string) error, stat func(string) (fs.FileInfo, error)) bool {
	if target == "" || !filepath.IsAbs(target) {
		return true
	}
	// These extensions describe documents, not a particular program name.
	// Unknown associations (scripts, profiles, protocols) are not guessed.
	switch strings.ToLower(filepath.Ext(target)) {
	case ".txt", ".text", ".md", ".rtf", ".htm", ".html", ".pdf", ".chm", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return false
	}
	if allow(target) != nil {
		return true
	} // Unknown/unavailable; never follow denied paths.
	info, err := stat(target)
	if err != nil {
		return true
	}
	return !info.IsDir()
}
