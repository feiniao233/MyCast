//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/feiniao233/MyCast/internal/core"
)

type sourceDiagnostic struct {
	source     string
	at         time.Time
	generation uint64
	partial    bool
}

// Only accepted configured-source batches are retained, one record per source.
// Partial batch status does not claim that an individual root is unavailable or
// that an unobserved application was uninstalled. No log/export/persistence.
func (s *launcher) recordSourceDiagnostics(roots []string, sources uint8, filePartial, shellPartial bool, generation uint64) {
	put := func(source string, partial bool) {
		value := sourceDiagnostic{source, time.Now(), generation, partial}
		for i, old := range s.sourceDiagnostics {
			if old.source == source {
				s.sourceDiagnostics[i] = value
				return
			}
		}
		if len(s.sourceDiagnostics) < 33 {
			s.sourceDiagnostics = append(s.sourceDiagnostics, value)
		}
	}
	if sources&scanFiles != 0 {
		for _, root := range roots {
			put(root, filePartial)
		}
	}
	if sources&scanShell != 0 && s.config.AppsFolder {
		put("AppsFolder", shellPartial)
	}
}

func applicationIdentityEvidence(app core.Application) string {
	switch {
	case app.DisplayOnlyID != "":
		return "未验证显示身份：不能关联、启动或推断卸载。"
	case app.ShellID != "":
		return "AppsFolder 命名空间身份；按原始 Shell 默认动作激活。"
	case strings.EqualFold(filepath.Ext(app.Path), ".lnk") && !app.Launch.AppUserModelIDKnown:
		return "AppID 属性读取未知；保留原始快捷方式身份，不按目标推断等价。"
	case app.Launch.Independent:
		return "独立启动语义标记；保留原始来源身份。"
	case app.Launch.ShellTarget != "" && app.Launch.DesktopBridge:
		return "已验证本地 EXE、AppID、空参数和默认工作目录的 Shell 关联；保留原始文件启动适配器。"
	case app.Launch.ShellTarget != "":
		return "已验证直接 AppsFolder 身份及默认快捷方式语义；保留原始文件适配器。"
	case app.Launch.Target == "":
		return "目标元数据不足；以原始来源路径区分入口。"
	default:
		value := "身份包含目标、原始参数、工作目录"
		if app.Launch.AppUserModelID != "" {
			value += "和显式 AppID"
		}
		return value + "；仅完全相同的身份可合并。"
	}
}

func (s *launcher) applicationDiagnostics(app core.Application) string {
	s.applicationResults() // Refresh the one slot before reading its evidence.
	evidence, found := s.appSearch.evidence[app.ItemID()]
	text := "\n\n搜索与身份诊断（只读；不自动保存或导出）\n"
	if !found {
		text += "当前输入中无匹配证据。\n"
	} else if evidence.Field == "" {
		text += "输入为空：按收藏、最近及名称/来源顺序显示。\n"
	} else {
		field := map[string]string{"name": "显示名", "alias": "索引别名", "pinyin": "名称/索引别名的拼音", "user-alias": "用户别名", "user-alias-pinyin": "用户别名的拼音", "user-alias-or-pinyin": "用户别名或其搜索展开"}[evidence.Field]
		class := []string{"精确", "前缀", "连续子串", "子序列"}[evidence.Class]
		text += fmt.Sprintf("匹配字段：%s\n匹配文本：%s\n匹配方式：%s；间隔 %d，起始 %d（字符位置）\n收藏加权：%t；最近序位权重：%d\n", field, evidence.Text, class, evidence.Gaps, evidence.Start, evidence.Favorite, evidence.Recent)
	}
	adapter := "原始文件 ShellExecute"
	if app.DisplayOnlyID != "" {
		adapter = "无（显示快照待验证）"
	} else if app.ShellID != "" {
		adapter = "AppsFolder 默认动作"
	}
	text += "当前适配器：" + adapter + "\n身份依据：" + applicationIdentityEvidence(app) + "\n"
	if len(app.NamePaths) > 1 {
		text += fmt.Sprintf("扫描保留来源数：%d（身份等价的文件来源）\n", len(app.NamePaths))
	}
	observed := false
	if app.DisplayOnlyID != "" {
		text += "快照不保存来源信息；以下为当前配置来源的批次记录。\n"
	}
	for _, state := range s.sourceDiagnostics {
		if app.DisplayOnlyID != "" || app.ShellID != "" && state.source == "AppsFolder" || app.Path != "" && state.source != "AppsFolder" && core.PathInApplicationRoots(app.Path, []string{state.source}) {
			status := "完整完成"
			if state.partial {
				status = "批次部分完成；未观察项的可用性未知"
			}
			text += fmt.Sprintf("来源 %s\n最近扫描批次：%s；世代 %d；%s\n", state.source, status, state.generation, state.at.UTC().Format(time.RFC3339))
			observed = true
		}
	}
	if !observed {
		text += "此来源本会话尚无已发布扫描批次记录。\n"
	}
	if len(text) > 8192 {
		text = text[:8192]
		for !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
		text += "\n（详情达到显示上限）"
	}
	return text
}
