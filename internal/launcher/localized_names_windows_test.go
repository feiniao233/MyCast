//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func TestLocalizedDisplayPreviewAndEnglishQuery(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.apps = s.apps[:8]
			for i, name := range []string{"合成计算器", "合成记事本", "合成画图工具", "合成系统工具", "合成截图工具", "合成设置", "合成文件管理器"} {
				s.apps[i] = core.PreferChineseDisplayName(s.apps[i], name)
			}
			s.apps = core.PrepareSearch(s.apps)
			s.status = "合成名称预览 · 有中文元数据时优先显示，英文仍可搜"
			tester := newPaletteTester(s, scale)
			if !tester.HasText("合成计算器") || !tester.HasText("Synthetic App 07") {
				t.Fatal("Chinese or untranslated brand display missing")
			}
			savePreview(t, tester, fmt.Sprintf("localized-applications-%03d", int(scale*100)))
			s.query = "Synthetic App 00"
			s.resetResults()
			tester.Frame()
			if !tester.HasText("合成计算器") {
				t.Fatal("original English query lost localized result")
			}
		})
	}
}
