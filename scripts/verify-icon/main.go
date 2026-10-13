// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/feiniao233/MyCast/internal/iconcheck"
)

func main() {
	exe := flag.String("exe", "dist/MyCast.exe", "PE image to verify")
	ico := flag.String("ico", "assets/brand/mycast.ico", "canonical ICO")
	flag.Parse()
	b, err := os.ReadFile(*ico)
	if err == nil {
		err = iconcheck.Executable(*exe, b)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		Status string `json:"status"`
		Sizes  []int  `json:"sizes"`
		Group  int    `json:"rtGroupIcon"`
		Icons  int    `json:"rtIcon"`
		SHA    string `json:"icoSha256"`
		Native string `json:"nativeVisualAcceptance"`
	}{"passed", iconcheck.Sizes, 1, 8, fmt.Sprintf("%x", sha256.Sum256(b)), "not-tested"})
}
