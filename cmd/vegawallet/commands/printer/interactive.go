package printer

import (
	"fmt"
	"io"

	"github.com/muesli/termenv"
)

type InteractivePrinter struct {
	writer  io.Writer
	profile termenv.Profile

	checkMark    string
	crossMark    string
	questionMark string
	arrow        termenv.Style
	bangMark     termenv.Style
}

func (p *InteractivePrinter) String() *FormattedString {
	return &FormattedString{
		profile:      p.profile,
		checkMark:    p.checkMark,
		crossMark:    p.crossMark,
		questionMark: p.questionMark,
		arrow:        p.arrow,
		bangMark:     p.bangMark,
	}
}

func (p *InteractivePrinter) Print(s *FormattedString) {
	if _, err := fmt.Fprint(p.writer, s.str); err != nil {
		panic(fmt.Sprintf("couldn't write to %v: %v", p.writer, err))
	}
}

type FormattedString struct {
	profile termenv.Profile

	checkMark    string
	crossMark    string
	questionMark string
	arrow        termenv.Style
	bangMark     termenv.Style

	str string
}

func (s *FormattedString) GreenArrow() *FormattedString {
	s.str += s.arrow.Foreground(s.profile.Color("2")).String()
	return s
}

func (s *FormattedString) RedArrow() *FormattedString {
	s.str += s.arrow.Foreground(s.profile.Color("1")).String()
	return s
}

func (s *FormattedString) BlueArrow() *FormattedString {
	s.str += s.arrow.Foreground(s.profile.Color("6")).String()
	return s
}

func (s *FormattedString) CheckMark() *FormattedString {
	s.str += s.checkMark
	return s
}

func (s *FormattedString) ListItem() *FormattedString {
	s.str += "    "
	return s
}

// Pad adds a padding that compensate the status characters.
// It's useful to display information on multiple lines.
func (s *FormattedString) Pad() *FormattedString {
	s.str += "  "
	return s
}

func (s *FormattedString) WarningBangMark() *FormattedString {
	s.str += s.bangMark.Foreground(s.profile.Color("3")).String()
	return s
}

func (s *FormattedString) DangerBangMark() *FormattedString {
	s.str += s.bangMark.Foreground(s.profile.Color("1")).String()
	return s
}

func (s *FormattedString) QuestionMark() *FormattedString {
	s.str += s.questionMark
	return s
}

func (s *FormattedString) CrossMark() *FormattedString {
	s.str += s.crossMark
	return s
}

func (s *FormattedString) SuccessText(t string) *FormattedString {
	s.str += termenv.String(t).Foreground(s.profile.Color("2")).String()
	return s
}

func (s *FormattedString) InfoText(t string) *FormattedString {
	s.str += termenv.String(t).Foreground(s.profile.Color("6")).String()
	return s
}

func (s *FormattedString) WarningText(t string) *FormattedString {
	s.str += termenv.String(t).Foreground(s.profile.Color("3")).String()
	return s
}

func (s *FormattedString) DangerText(t string) *FormattedString {
	s.str += termenv.String(t).Foreground(s.profile.Color("1")).String()
	return s
}

func (s *FormattedString) NextLine() *FormattedString {
	s.str += "\n"
	return s
}

func (s *FormattedString) NextSection() *FormattedString {
	s.str += "\n\n"
	return s
}

func (s *FormattedString) Text(str string) *FormattedString {
	s.str += str
	return s
}

func (s *FormattedString) Code(str string) *FormattedString {
	s.str += fmt.Sprintf("    $ %s", str)
	return s
}

func (s *FormattedString) Bold(str string) *FormattedString {
	s.str += termenv.String(str).Bold().String()
	return s
}

func (s *FormattedString) DangerBold(str string) *FormattedString {
	s.str += termenv.String(str).Bold().Foreground(s.profile.Color("1")).String()
	return s
}

func (s *FormattedString) SuccessBold(str string) *FormattedString {
	s.str += termenv.String(str).Bold().Foreground(s.profile.Color("2")).String()
	return s
}

func (s *FormattedString) Underline(str string) *FormattedString {
	s.str += termenv.String(str).Underline().String()
	return s
}
