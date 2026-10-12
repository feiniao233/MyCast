// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type QuickLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (link QuickLink) Validated() (QuickLink, error) {
	link.Name = strings.TrimSpace(link.Name)
	if link.Name == "" || !utf8.ValidString(link.Name) || utf8.RuneCountInString(link.Name) > 80 || strings.ContainsAny(link.Name, "|\r\n\x00") || strings.IndexFunc(link.Name, unicode.IsControl) >= 0 {
		return QuickLink{}, fmt.Errorf("link names require 1–80 characters without separators/controls")
	}
	value, err := ValidateHTTPURL(link.URL)
	if err != nil {
		return QuickLink{}, err
	}
	link.URL = value
	return link, nil
}
func (link QuickLink) ItemID() string {
	return fmt.Sprintf("link-%x", sha256.Sum256([]byte(link.Name+"\x00"+link.URL)))
}

func ValidateHTTPURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 2048 || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsSpace) >= 0 || strings.ContainsAny(value, "\"\\<>`^") {
		return "", fmt.Errorf("link URL is invalid or exceeds 2048 bytes")
	}
	decoded, err := url.PathUnescape(value)
	if err != nil || strings.IndexFunc(decoded, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("link URL contains invalid control data")
	}
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("link URL is invalid")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if (u.Scheme != "http" && u.Scheme != "https") || u.Opaque != "" || u.Hostname() == "" || u.User != nil {
		return "", fmt.Errorf("use an absolute HTTP(S) URL without embedded credentials")
	}
	if port := u.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("link port is invalid")
		}
	}
	canonical := u.String()
	if len(canonical) > 2048 {
		return "", fmt.Errorf("encoded link exceeds 2048 bytes")
	}
	return canonical, nil
}
