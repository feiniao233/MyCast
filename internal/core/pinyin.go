// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"strings"
	"unicode"

	pinyin "github.com/mozillazg/go-pinyin"
)

// PinyinTerms uses only the bundled local dictionary. At most eight readings
// survive each step, bounding polyphonic combinations. Latin text is preserved;
// dictionary readings do not infer word context, so user aliases handle misses.
func PinyinTerms(name string) []string {
	if len([]rune(name)) > 256 {
		return nil
	}
	type variant struct{ full, initial string }
	variants := []variant{{}}
	hasHan := false
	args := pinyin.Args{Style: pinyin.Normal, Heteronym: true}
	for _, r := range strings.ToLower(name) {
		if !unicode.Is(unicode.Han, r) {
			for i := range variants {
				variants[i].full += string(r)
				variants[i].initial += string(r)
			}
			continue
		}
		readings := pinyin.SinglePinyin(r, args)
		if len(readings) == 0 {
			for i := range variants {
				variants[i].full += string(r)
				variants[i].initial += string(r)
			}
			continue
		}
		hasHan = true
		var next []variant
		seen := map[variant]bool{}
		for _, base := range variants {
			for _, reading := range readings {
				if reading == "" {
					continue
				}
				candidate := variant{base.full + reading, base.initial + reading[:1]}
				if !seen[candidate] {
					seen[candidate] = true
					next = append(next, candidate)
				}
				if len(next) >= 8 {
					break
				}
			}
			if len(next) >= 8 {
				break
			}
		}
		variants = next
	}
	if !hasHan {
		return nil
	}
	var result []string
	seen := map[string]bool{}
	for _, value := range variants {
		for _, term := range []string{value.full, value.initial} {
			term = strings.TrimSpace(term)
			if term != "" && !seen[term] {
				seen[term] = true
				result = append(result, term)
			}
		}
	}
	return result
}

// PrepareSearch runs once per index refresh, never changing launch identity.
func PrepareSearch(apps []Application) []Application {
	return PrepareSearchContext(context.Background(), apps)
}
func PrepareSearchContext(ctx context.Context, apps []Application) []Application {
	result := append([]Application(nil), apps...)
	for i := range result {
		if ctx.Err() != nil {
			break
		}
		var terms []string
		seen := map[string]bool{}
		for _, name := range append([]string{result[i].Name}, result[i].Aliases...) {
			if ctx.Err() != nil {
				break
			}
			for _, term := range PinyinTerms(name) {
				if !seen[term] {
					seen[term] = true
					terms = append(terms, term)
				}
			}
		}
		result[i].SearchTerms = terms
	}
	return result
}

func WithUserAliases(apps []Application, aliases map[string][]string) []Application {
	result := append([]Application(nil), apps...)
	for i := range result {
		result[i].UserAliases = nil
		result[i].UserAliasKinds = nil
		for _, id := range result[i].PreferenceIDs() {
			for _, alias := range aliases[id] {
				result[i].UserAliases = append(result[i].UserAliases, alias)
				result[i].UserAliasKinds = append(result[i].UserAliasKinds, "user-alias")
				for _, term := range PinyinTerms(alias) {
					result[i].UserAliases = append(result[i].UserAliases, term)
					result[i].UserAliasKinds = append(result[i].UserAliasKinds, "user-alias-pinyin")
				}
			}
		}
	}
	return result
}
