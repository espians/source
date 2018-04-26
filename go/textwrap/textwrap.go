// Public Domain (-) 2018-present, The Espian Source Authors.
// See the Espian Source UNLICENSE file for details.

// Package textwrap provides utilities for manipulating plain text.
package textwrap

import (
	"strings"
)

// Dedent removes any common leading whitespace from every line in the given
// text. Both tabs and spaces are treated as whitespace, and blank lines are
// ignored for the purposes of dedenting.
func Dedent(text string) string {
	common := ""
	firstLine := true
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		current := ""
		for i := 0; i < len(line); i++ {
			char := line[i]
			if char == ' ' {
				current += " "
			} else if char == '\t' {
				current += "\t"
			} else {
				if firstLine {
					common = current
					firstLine = false
				} else if common != current {
					if strings.HasPrefix(current, common) {
						break
					} else {
						found := false
						for j := len(current); j > 0; j-- {
							if strings.HasPrefix(common, current[:j]) {
								common = current[:j]
								found = true
								break
							}
						}
						if !found {
							return text
						}
					}
				}
				break
			}
		}
	}
	if common == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	formatted := make([]string, len(lines))
	for idx, line := range lines {
		if line == "" {
			formatted[idx] = ""
			continue
		}
		formatted[idx] = line[len(common):]
	}
	return strings.Join(formatted, "\n")
}
