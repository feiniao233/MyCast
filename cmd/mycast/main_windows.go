// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build windows

package main

import (
	"github.com/feiniao233/MyCast/internal/launcher"
	"log"
)

func main() {
	if err := launcher.Run(); err != nil {
		log.Fatal(err)
	}
}
