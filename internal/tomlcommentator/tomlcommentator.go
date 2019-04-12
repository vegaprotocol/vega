package tomlcommentator

import (
	"fmt"
	"regexp"
	"strings"
)

// Comments encapsulates all the comments that can be added to a toml file.
type Comments struct {
	Header []string
	Items  []*CommentItem
	Footer []string
}

// CommentItem stores a single comment addition.
type CommentItem struct {
	Regex       string
	regex       *regexp.Regexp
	CommentEOL  string
	CommentPara []string
}

// Commentate adds all the given comments to a toml string.
func Commentate(toml string, c *Comments) string {
	if c == nil {
		return toml
	}

	results := make([]string, 0)
	if c.Header != nil && len(c.Header) > 0 {
		if c.Header[len(c.Header)-1] != "" {
			c.Header = append(c.Header, "")
		}
		results = append(results, addHashes(c.Header, 0)...)
	}

	tomlLines := strings.Split(toml, "\n")
	if c.Items != nil && len(c.Items) > 0 {
		// Compile all regexes once.
		for _, item := range c.Items {
			item.regex = regexp.MustCompile(item.Regex)
		}

		// Process lines
		for _, tomlLine := range tomlLines {
			indent := countIndent(tomlLine)
			eolAdditions := make([]string, 0)
			for _, item := range c.Items {
				if item.regex.MatchString(tomlLine) {
					if item.CommentPara != nil && len(item.CommentPara) > 0 {
						results = append(results, addHashes(item.CommentPara, indent)...)
					}
					if item.CommentEOL != "" {
						eolAdditions = append(eolAdditions, item.CommentEOL)
					}
				}
			}
			if len(eolAdditions) > 0 {
				tomlLine = fmt.Sprintf("%s  # %s", tomlLine, strings.Join(eolAdditions, ""))
			}
			results = append(results, tomlLine)
		}
	} else {
		results = append(results, tomlLines...)
	}

	if c.Footer != nil && len(c.Footer) > 0 {
		if len(results) > 0 && results[len(results)-1] != "" {
			results = append(results, "")
		}
		results = append(results, addHashes(c.Footer, 0)...)
	}
	if len(results) > 0 && results[len(results)-1] != "" {
		results = append(results, "")
	}
	return strings.Join(results, "\n")
}

func addHashes(l []string, indent int) []string {
	results := make([]string, len(l))
	for i, item := range l {
		if item != "" {
			results[i] = fmt.Sprintf("%s# %s", strings.Repeat(" ", indent), item)
		}
	}
	return results
}

func countIndent(s string) int {
	l := len(s)
	for i := 1; i <= l; i++ {
		if !strings.HasPrefix(s, strings.Repeat(" ", i)) {
			return i - 1
		}
	}
	return l
}
