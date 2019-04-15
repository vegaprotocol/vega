package commentator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/internal/commentator"
)

const sampleToml = `[section]
  a = "b"`

func TestCommentateNilPointer(t *testing.T) {
	assert.Equal(t, sampleToml, commentator.Commentate(sampleToml, nil))
}

func TestCommentateNil(t *testing.T) {
	c := &commentator.Comments{
		Header: nil,
		Items:  nil,
		Footer: nil,
	}
	assert.Equal(t, sampleToml+"\n", commentator.Commentate(sampleToml, c))
}

func TestCommentateEmpty(t *testing.T) {
	c := &commentator.Comments{
		Header: []string{},
		Items:  []*commentator.CommentItem{},
		Footer: []string{},
	}
	assert.Equal(t, sampleToml+"\n", commentator.Commentate(sampleToml, c))
}

func TestCommentateHeader(t *testing.T) {
	c := &commentator.Comments{
		Header: []string{
			"This is a TOML config file.",
			"For more information, see https://github.com/toml-lang/toml",
		},
	}
	expected := "# This is a TOML config file.\n# For more information, see https://github.com/toml-lang/toml\n\n[section]\n  a = \"b\"\n"
	assert.Equal(t, expected, commentator.Commentate(sampleToml, c))
}

func TestCommentateFooter(t *testing.T) {
	c := &commentator.Comments{
		Footer: []string{
			"This was a TOML config file.",
			"And here is the end of the file.",
		},
	}
	expected := "[section]\n  a = \"b\"\n\n# This was a TOML config file.\n# And here is the end of the file.\n"
	assert.Equal(t, expected, commentator.Commentate(sampleToml, c))
}

func TestCommentateCommentEOL(t *testing.T) {
	c := &commentator.Comments{
		Items: []*commentator.CommentItem{
			&commentator.CommentItem{
				Regex:      `^\[section\]$`,
				CommentEOL: "This is a section",
			},
		},
	}
	expected := "[section]  # This is a section\n  a = \"b\"\n"
	assert.Equal(t, expected, commentator.Commentate(sampleToml, c))
}

func TestCommentateCommentPara(t *testing.T) {
	c := &commentator.Comments{
		Items: []*commentator.CommentItem{
			&commentator.CommentItem{
				Regex:       `^\[section\]$`,
				CommentPara: []string{"This is a section"},
			},
		},
	}
	expected := "# This is a section\n[section]\n  a = \"b\"\n"
	assert.Equal(t, expected, commentator.Commentate(sampleToml, c))
}

func TestCommentateCommentParaIndented(t *testing.T) {
	c := &commentator.Comments{
		Items: []*commentator.CommentItem{
			&commentator.CommentItem{
				Regex:       `a = ".*"$`,
				CommentPara: []string{`This is variable "a"`},
			},
		},
	}
	expected := "[section]\n  # This is variable \"a\"\n  a = \"b\"\n"
	assert.Equal(t, expected, commentator.Commentate(sampleToml, c))
}

func TestCommentateCommentParaIndentedWhitespace(t *testing.T) {
	c := &commentator.Comments{
		Items: []*commentator.CommentItem{
			&commentator.CommentItem{
				Regex:       `    `, // just four spaces
				CommentPara: []string{`Why is the next line just whitespace`},
			},
		},
	}
	data := "[section]\n    \n  a = \"b\"\n"
	expected := "[section]\n    # Why is the next line just whitespace\n    \n  a = \"b\"\n"
	assert.Equal(t, expected, commentator.Commentate(data, c))
}
