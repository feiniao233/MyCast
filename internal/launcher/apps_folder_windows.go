//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
	"time"
)

func (s *launcher) readShellApps(ctx context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
	if time.Since(s.shellImagesAt) >= 10*time.Minute {
		s.shellImages = nil
		s.shellImagesAt = time.Now()
	}
	apps, images, errs := platform.AppsFolderApplications(ctx, s.shellImages)
	if ctx.Err() == nil && len(errs) == 0 {
		s.shellImages = images
	}
	icons := map[string]*ui.Bitmap{}
	for id, im := range images {
		if im != nil {
			icons[id] = ui.NewBitmap(im)
		}
	}
	return core.PrepareSearchContext(ctx, apps), icons, errs
}
