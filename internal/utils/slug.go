package utils

import (
	"regexp"
	"strings"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9-]+`)
var multiDash = regexp.MustCompile(`-+`)

func Slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "&", " and ")
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, " ", "-")
	s = nonSlugChars.ReplaceAllString(s, "-")
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
