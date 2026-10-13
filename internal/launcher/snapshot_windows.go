//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

type snapshotJob struct {
	clear bool
	epoch uint64
	value settings.Snapshot
}

func (s *launcher) drawSnapshotSettings(c *ui.Context) {
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	ui.ButtonBase(c.Key("snapshot-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(p.ModalScrim).Label("启动快照背景")
	panel := ui.Column(c.Key("snapshot-panel")).Absolute().Left(p.ModalSide).Right(p.ModalSide).Top(p.ModalTop).Bottom(p.ModalTop).Padding(p.ModalPadding).Gap(p.ModalGap).Radius(p.ModalRadius).Background(p.MenuSurface).Border(1, p.Border).Role(ui.RoleDialog).Label("启动快照面板").Focusable()
	if s.settingsNeedsFocus {
		panel.Focus()
		s.settingsNeedsFocus = false
	}
	panel.Children(func() {
		ui.Text(c, "本地启动显示快照").FontSize(p.SearchSize).Bold()
		ui.Checkbox(c.Key("settings-snapshot"), &s.settingsDraft.StartupSnapshot, "启用显示快照（默认关闭）").Label("启动显示快照")
		ui.Text(c, "只保存显示名和身份哈希；不保存路径、启动参数、查询或剪贴板。\n重启后可先显示待验证入口，来源验证前不能启动。\n关闭后不再读取或更新；已有快照可在这里清除。\n返回设置并保存后，开关才会生效。").FillWidth().FontSize(p.ModalBody).TextColor(p.Secondary)
		clear := ui.ButtonBase(c.Key("settings-clear-snapshot")).Height(p.ModalClear).Shrink(0).Padding(p.ControlInset, p.Gap).Radius(p.Inset).Background(p.Control).Label("清除本地启动快照").Disabled(s.snapshotClearing)
		clear.Children(func() { ui.Text(c, "立即清除启动快照").FontSize(12) })
		if clear.Clicked() {
			s.clearStartupSnapshot()
		}
		ui.Text(c, s.snapshotStatus).FillWidth().FontSize(12).TextColor(p.Secondary)
		ui.Spacer(c)
		back := ui.ButtonBase(c.Key("snapshot-back")).Height(p.ModalBack).Shrink(0).Padding(p.MenuInset, p.BodySize).Radius(p.Inset).Background(p.Control).Label("返回设置")
		back.Children(func() { ui.Text(c, "返回设置").FontSize(14) })
		if back.Clicked() || c.Shortcut(0, ui.KeyEscape) {
			s.snapshotOpen = false
			s.settingsNeedsFocus = true
		}
	})
}

func (s *launcher) queueSnapshot(job snapshotJob) {
	if s.snapshotStore == nil || s.uiUpdate == nil {
		return
	}
	if s.snapshotWorker == nil {
		s.snapshotWorker = &snapshotWorker{store: s.snapshotStore, notify: func(job snapshotJob, err error) {
			s.uiUpdate(func() {
				if s.quitting {
					return
				}
				if job.epoch != s.snapshotStore.Epoch() {
					return
				}
				if job.clear {
					s.snapshotClearing = false
				}
				if err != nil {
					if job.clear {
						s.snapshotStatus = "清除未完成；原缓存可能保留，可重试。"
					} else {
						s.snapshotStatus = "快照操作未完成；应用发现继续运行。"
					}
				} else if job.clear {
					s.snapshotStatus = "本地启动快照已清除。"
				} else {
					s.snapshotStatus = "已保存完整来源的显示快照。"
				}
			})
		}}
	}
	if !s.snapshotWorker.submit(job) && job.clear {
		s.snapshotClearing = false
		s.snapshotStatus = "退出期间未接受新的清除；原缓存可能保留。"
	}
}
func (s *launcher) clearStartupSnapshot() {
	if s.snapshotStore == nil {
		s.settingsNote = "快照存储不可用。"
		return
	}
	s.snapshotStore.Invalidate()
	s.snapshotClearing = true
	s.pendingApps = nil
	s.invalidateApplicationSearch()
	s.queueSnapshot(snapshotJob{clear: true, epoch: s.snapshotStore.Epoch()})
	s.settingsNote = "正在清除本地显示快照；原应用和设置不受影响。"
}
func snapshotDisplayApplications(value settings.Snapshot, aliases map[string][]string, configs ...settings.Settings) []core.Application {
	apps := make([]core.Application, 0, len(value.Entries))
	for _, e := range value.Entries {
		apps = append(apps, core.Application{Name: e.Name, DisplayOnlyID: e.ID})
	}
	if len(configs) > 0 {
		apps = configs[0].ApplyIdentityLinks(apps)
	} // Existing saved preferences only; no cache proof.
	return core.WithUserAliases(core.PrepareSearch(apps), aliases)
}

// Keep unobserved cached rows pending during partial discovery. They are never
// supplied as previous provider entries and cannot become equivalence proofs.
func (s *launcher) adoptApplications(apps []core.Application, icons map[string]*ui.Bitmap) {
	seen := map[string]bool{}
	for _, app := range apps {
		seen[app.ItemID()] = true
	}
	if s.pendingComplete {
		s.pendingApps = nil
	} else {
		pending := make([]core.Application, 0, len(s.pendingApps))
		for _, app := range s.pendingApps {
			if !seen[app.ItemID()] && len(pending)+len(apps) < 5000 {
				pending = append(pending, app)
			}
		}
		s.pendingApps = pending
	}
	s.pendingComplete = false
	if len(s.pendingApps) == 0 {
		s.pendingReviewQueued = false
	}
	s.apps, s.icons = apps, icons
	s.invalidateApplicationSearch()
}

// Cache entries deliberately have no source-path proof. Independently healthy
// configured sources therefore trigger a fresh full review rather than deleting
// an unobserved row by attribution guessed from its cached name/hash.
func (s *launcher) needsPendingSnapshotReview(roots []string, shell bool) bool {
	if len(s.pendingApps) == 0 {
		return false
	}
	complete := func(source string) bool {
		for _, state := range s.sourceDiagnostics {
			if state.source == source {
				return !state.partial
			}
		}
		return false
	}
	for _, root := range roots {
		if !complete(root) {
			s.pendingReviewQueued = false
			return false
		}
	}
	if shell && !complete("AppsFolder") {
		s.pendingReviewQueued = false
		return false
	}
	return !s.pendingReviewQueued
}
