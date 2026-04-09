package normalize

import (
	"regexp"
	"strings"
	"unicode"
)

var titleSearchParentheticalPattern = regexp.MustCompile(`\s*[\(\[][^(\)\[\]]+[\)\]]`)
var titleSearchParentheticalContentPattern = regexp.MustCompile(`[\(\[]([^()\[\]]+)[\)\]]`)

// SearchTitleVariants returns ordered title strings suitable for search
// fallbacks. It keeps the original title first, then adds romanized or
// alternate titles found in parentheses, and finally a version with the
// parenthetical content stripped.
func SearchTitleVariants(title string) []string {
	variants := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	appendUnique := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := Text(value)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		variants = append(variants, value)
	}

	appendUnique(title)
	if containsNonLatinLetter(title) {
		for _, match := range titleSearchParentheticalContentPattern.FindAllStringSubmatch(title, -1) {
			if len(match) < 2 {
				continue
			}
			candidate := strings.TrimSpace(match[1])
			if containsLatinLetter(candidate) {
				appendUnique(candidate)
			}
		}
	}

	stripped := strings.Join(strings.Fields(titleSearchParentheticalPattern.ReplaceAllString(title, " ")), " ")
	appendUnique(stripped)
	return variants
}

func containsLatinLetter(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) && unicode.In(r, unicode.Latin) {
			return true
		}
	}
	return false
}

func containsNonLatinLetter(value string) bool {
	for _, r := range value {
		if !unicode.IsLetter(r) {
			continue
		}
		if !unicode.In(r, unicode.Latin) {
			return true
		}
	}
	return false
}
