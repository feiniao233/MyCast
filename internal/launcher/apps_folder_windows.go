//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
)

func readShellApps(ctx context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
	apps, images, errs := platform.AppsFolderApplications(ctx)
	icons := map[string]*ui.Bitmap{}
	for id, im := range images {
		icons[id] = ui.NewBitmap(im)
	}
	return core.PrepareSearchContext(ctx, apps), icons, errs
}
