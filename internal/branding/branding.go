// SPDX-License-Identifier: AGPL-3.0-or-later
// Package branding contains the original MyCast icon shared by window and tray.
package branding

import _ "embed"

//go:embed mycast-256.png
var windowPNG []byte

//go:embed mycast-32.png
var trayPNG []byte

// WindowPNG supplies a high-resolution source to MyGo's DPI-aware SetIcon.
// MyGo owns both resulting HICONs, including replacement and destruction.
func WindowPNG() []byte { return append([]byte(nil), windowPNG...) }

func TrayPNG() []byte { return append([]byte(nil), trayPNG...) }
