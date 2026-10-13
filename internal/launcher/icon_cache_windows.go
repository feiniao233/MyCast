//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/feiniao233/MyCast/internal/platform"
	"image"
	"os"
)

type fileIconEntry struct {
	info  os.FileInfo
	image image.Image
}

// Owned by the single scan worker. No user paths or image data are serialized.
func (s *launcher) readFileIcon(path string) image.Image {
	if platform.LocalAppPath(path) != nil {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	if cached, ok := s.fileImages[path]; ok && os.SameFile(info, cached.info) && info.Size() == cached.info.Size() && info.ModTime().Equal(cached.info.ModTime()) {
		return cached.image
	}
	im := platform.ApplicationIcon(path)
	if s.fileImages == nil || len(s.fileImages) >= 256 {
		s.fileImages = map[string]fileIconEntry{}
	}
	s.fileImages[path] = fileIconEntry{info: info, image: im}
	return im
}
