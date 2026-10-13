// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// AppsFolder entries are Shell identities, never filesystem paths or commands.
// The namespace CLSID is distinct from FOLDERID_AppsFolder used for enumeration.
const AppsFolderParsingPrefix = `::{4234d49b-0245-4df3-b780-3893943456e1}\`

// Normalize only after the platform has proved that the item belongs to
// AppsFolder. Windows returns bare AUMIDs even for DESKTOPABSOLUTEPARSING.
func AppsFolderChildIdentity(value string) (string, error) {
	if normalized, err := ValidateShellApplication(value); err == nil {
		return normalized, nil
	}
	if value == "" || strings.ContainsAny(value, `\/:")`) {
		return "", fmt.Errorf("invalid AppsFolder child identity")
	}
	return ValidateShellApplication(AppsFolderParsingPrefix + value)
}

func ShellApplicationActivationName(value string) (string, error) {
	canonical, err := ValidateShellApplication(value)
	if err != nil {
		return "", err
	}
	return `shell:AppsFolder\` + strings.TrimPrefix(canonical, AppsFolderParsingPrefix), nil
}

// Search supplements are tied to a public package identity, not the label.
func SystemApplicationAliases(identity string) []string {
	if identity == AppsFolderParsingPrefix+"Microsoft.WindowsStore_8wekyb3d8bbwe!App" {
		return []string{"微软商店", "微软应用商店", "Windows Store"}
	}
	return nil
}

func ValidateShellApplication(value string) (string, error) {
	if len(value) > 2048 || !utf8.ValidString(value) || len(value) <= len(AppsFolderParsingPrefix) || !strings.EqualFold(value[:len(AppsFolderParsingPrefix)], AppsFolderParsingPrefix) {
		return "", fmt.Errorf("invalid AppsFolder identity")
	}
	tail := value[len(AppsFolderParsingPrefix):]
	for _, r := range tail {
		if unicode.IsControl(r) || strings.ContainsRune(`\/"`, r) {
			return "", fmt.Errorf("invalid AppsFolder identity")
		}
	}
	return AppsFolderParsingPrefix + tail, nil
}
