// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"strings"
	"testing"
)

func TestBoundedArithmeticGrammarAndResults(t *testing.T) {
	for expression, want := range map[string]string{"1+2*3": "7", "(1+2)*3": "9", "−2 × (3 + 4)": "-14", "= .5 + 1e-2": "0.51", "0.1+0.2": "0.3", "1/3": "0.333333333333", "-(-2)": "2", "1e3 / 4": "250"} {
		got, err := Calculate(expression)
		if err != nil || got != want {
			t.Fatalf("arithmetic %q: %q %v", expression, got, err)
		}
	}
	for _, expression := range []string{"", "1/0", "1+", "1;exit", "system(1)", "1**2", "NaN", "Inf", "1e999", "1e300*1e300", "(1+2", "1 2", strings.Repeat("(", 33) + "1" + strings.Repeat(")", 33), strings.Repeat("1+", 100) + "1", strings.Repeat("1", 65), strings.Repeat("1", 513)} {
		if _, err := Calculate(expression); err == nil {
			t.Fatalf("unsafe/unbounded expression accepted: %q", expression)
		}
	}
}

func TestQuickLinkValidationAllowsOnlyExplicitHTTPURLs(t *testing.T) {
	for _, value := range []string{`https://example.com/?q="literal"`, `https://example.com/?q=\path`, "https://example.com/" + strings.Repeat("字", 400)} {
		if _, err := ValidateHTTPURL(value); err == nil {
			t.Fatal("raw unsafe characters or oversized canonical URL accepted")
		}
	}
	for _, value := range []string{"https://example.com/path?q=synthetic#anchor", "HTTP://example.com:8080/", "https://example.com/a%20b", "http://[::1]:8080/"} {
		if _, err := ValidateHTTPURL(value); err != nil {
			t.Fatalf("valid synthetic URL rejected: %q %v", value, err)
		}
	}
	for _, value := range []string{"javascript:alert(1)", "file:///C:/Synthetic.exe", "https:", "//example.com", "https://name:synthetic@example.com", "https://example.com/%0a", "https://example.com/%00", "https://example.com:99999", "https://exa mple.com", "https://example.com/\x00", strings.Repeat("x", 2049)} {
		if _, err := ValidateHTTPURL(value); err == nil {
			t.Fatal("unsupported URL accepted")
		}
	}
	for _, name := range []string{"", "a|b", "a\nb", strings.Repeat("字", 81)} {
		if _, err := (QuickLink{Name: name, URL: "https://example.com"}).Validated(); err == nil {
			t.Fatal("invalid link name accepted")
		}
	}
}
