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
