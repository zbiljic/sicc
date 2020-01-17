package cmd

import (
	"sort"
	"strings"
)

const doubleQuoteSpecialChars = "\\\n\r\"!$`"

func stripPrefix(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix)+1:]
	}

	return s
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	i := 0

	for k := range m {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	return keys
}

func doubleQuoteEscape(line string) string {
	for _, c := range doubleQuoteSpecialChars {
		toReplace := "\\" + string(c)

		if c == '\n' {
			toReplace = `\n`
		}

		if c == '\r' {
			toReplace = `\r`
		}

		line = strings.ReplaceAll(line, string(c), toReplace)
	}

	return line
}
