// SPDX-License-Identifier: AGPL-3.0-or-later
package branding

import (
	"bytes"
	"os"
	"testing"

	"github.com/feiniao233/MyCast/internal/iconcheck"
)

func TestWindowAndTrayUseCanonicalArtwork(t *testing.T) {
	data, err := os.ReadFile("../../assets/brand/mycast.ico")
	if err != nil {
		t.Fatal(err)
	}
	images, err := iconcheck.ICO(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(WindowPNG(), images[7]) || !bytes.Equal(TrayPNG(), images[3]) {
		t.Fatal("window/tray differs from canonical icon")
	}
	window, tray := WindowPNG(), TrayPNG()
	window[0] = 0
	tray[0] = 0
	if !bytes.Equal(WindowPNG(), images[7]) || !bytes.Equal(TrayPNG(), images[3]) {
		t.Fatal("caller can alter shared icon")
	}
}
