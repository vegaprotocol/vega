package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

const indentation = `  `

type exampleData struct {
	Software string
}

// LongDesc normalizes a command's long description to follow the conventions.
func LongDesc(s string) string {
	if len(s) == 0 {
		return s
	}
	return normalizer{s}.trim().noIndent().string
}

// Examples normalizes a command's examples to follow the conventions.
func Examples(s string) string {
	if len(s) == 0 {
		return s
	}

	software := os.Args[0]

	// If the wallet cmd is embedded inside vega, we display the software as
	// a sub-command in the examples.
	if software == "vega" || strings.HasSuffix(software, "/vega") {
		software = fmt.Sprintf("%s wallet", software)
	}

	sweaters := exampleData{
		Software: software,
	}
	tmpl, err := template.New("example").Parse(s)
	if err != nil {
		panic(err)
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, sweaters)
	if err != nil {
		panic(err)
	}

	return normalizer{tpl.String()}.trim().indent().string
}

type normalizer struct {
	string
}

func (s normalizer) trim() normalizer {
	s.string = strings.TrimSpace(s.string)
	return s
}

func (s normalizer) indent() normalizer {
	indentedLines := []string{}
	for _, line := range strings.Split(s.string, "\n") {
		trimmed := strings.TrimSpace(line)
		indented := indentation + trimmed
		indentedLines = append(indentedLines, indented)
	}
	s.string = strings.Join(indentedLines, "\n")
	return s
}

func (s normalizer) noIndent() normalizer {
	indentedLines := []string{}
	for _, line := range strings.Split(s.string, "\n") {
		trimmed := strings.TrimSpace(line)
		indentedLines = append(indentedLines, trimmed)
	}
	s.string = strings.Join(indentedLines, "\n")
	return s
}
