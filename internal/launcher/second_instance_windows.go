//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

// MyGo v0.4.6 already forwards os.Args[1:] without an executable slot.
func handleSecondInvocation(args []string, show func()) {
	if !backgroundInvocation(args) {
		show()
	}
}
