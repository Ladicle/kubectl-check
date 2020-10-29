package pritty

import (
	"fmt"
	"strconv"
	"strings"
)

type Attribute int

const (
	Bold = Attribute(iota + 1)
	Bright
	Italic
	Underscore
	Blink
	FastBlink
	Reverse
	Hidden
	Conceal
)

type Color int

const (
	Black = Color(-1)
	Red   = Color(iota)
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

type TextStyle struct {
	bg, fg    Color
	attribute Attribute
	Text      string
}

func Style(text string) *TextStyle {
	return &TextStyle{Text: text}
}

func (s *TextStyle) Decorate(at Attribute) *TextStyle {
	s.attribute = at
	return s
}

func (s *TextStyle) Fg(color Color) *TextStyle {
	s.fg = color
	return s
}

func (s *TextStyle) Bg(color Color) *TextStyle {
	s.bg = color
	return s
}

func (s TextStyle) String() string {
	var seq []string
	if s.fg != 0 {
		seq = append(seq, strconv.Itoa(colorCode(s.fg)+30))
	}
	if s.bg != 0 {
		seq = append(seq, strconv.Itoa(colorCode(s.bg)+40))
	}
	if s.attribute != 0 {
		seq = append(seq, strconv.Itoa(int(s.attribute)))
	}
	if len(seq) < 1 {
		return s.Text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", strings.Join(seq, ";"), s.Text)
}

func colorCode(color Color) int {
	if color == Black {
		return 0
	}
	return int(color)
}
