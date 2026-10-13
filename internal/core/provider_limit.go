// SPDX-License-Identifier: AGPL-3.0-or-later
package core

// Protect both providers from an append-order truncation. Keep known identities
// before new entries in each partition, and lend unused slots to the other pool.
func FairProviderLimit(apps, previous []Application, limit int) []Application {
	if len(apps) <= limit {
		return apps
	}
	known := map[string]bool{}
	for _, app := range previous {
		known[app.ItemID()] = true
	}
	var files, shell []Application
	for _, keepKnown := range []bool{true, false} {
		for _, app := range apps {
			if known[app.ItemID()] != keepKnown {
				continue
			}
			if app.ShellID != "" {
				shell = append(shell, app)
			} else {
				files = append(files, app)
			}
		}
	}
	shellCount := min(len(shell), limit/2)
	fileCount := min(len(files), limit-shellCount)
	shellCount = min(len(shell), limit-fileCount)
	fileCount = min(len(files), limit-shellCount)
	result := append([]Application(nil), files[:fileCount]...)
	return append(result, shell[:shellCount]...)
}
