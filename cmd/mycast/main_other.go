// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("MyCast currently supports Windows 10/11. See README.md for cross-compilation.")
}
