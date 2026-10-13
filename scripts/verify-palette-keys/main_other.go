//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import "fmt"

func main() { fmt.Println("Windows-only owned palette key fixture; not tested on this platform.") }
