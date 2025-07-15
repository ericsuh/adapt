package aptfile

import (
	"unicode"
)

type Token struct {
	Type  uint8
	Coord FileCoord
}

func (t Token) Text() string {
	return t.Coord.Text()
}

const (
	StringToken uint8 = iota
	CommaToken
	ColonToken
)

type ParseError struct {
	Message string
	Coord   FileCoord
}

func (e ParseError) Error() string {
	return e.Coord.LineAnnotated(e.Message)
}

func lexLine(coord FileCoord) ([]Token, error) {
	toks := make([]Token, 0, 1)
	var bStart = -1
	var eolCol = -1
	quoted := false
	pushAccumulatedToken := func(end int, force bool) {
		if force || (bStart >= 0 && end-bStart > 0) {
			coord := FileCoord{
				Line:     coord.Line,
				LineNum:  coord.LineNum,
				ColStart: bStart,
				ColEnd:   end,
			}
			t := StringToken
			switch coord.Text() {
			case ",":
				t = CommaToken
			case ":":
				t = ColonToken
			}

			toks = append(toks, Token{
				Type:  t,
				Coord: coord,
			})
			bStart = -1
		}
	}
looping:
	for i, r := range coord.Line {
		if quoted {
			if r == '"' {
				// Push even if it's empty (e.g. "")
				pushAccumulatedToken(i, true)
				quoted = false
			}
			continue
		}
		switch {
		case r == '"':
			if i == 0 {
				return []Token{}, ParseError{
					Coord: FileCoord{
						Line:     coord.Line,
						LineNum:  coord.LineNum,
						ColStart: 0,
						ColEnd:   1,
					},
					Message: "cannot quote directive command",
				}
			}
			pushAccumulatedToken(i, false)
			quoted = true
			bStart = i + 1 // Start at next character
		case r == '#':
			pushAccumulatedToken(i, false)
			eolCol = i
			break looping
		case r == ':':
			pushAccumulatedToken(i, false)
			bStart = i
			pushAccumulatedToken(i+1, false)
		case r == ',':
			pushAccumulatedToken(i, false)
			bStart = i
			pushAccumulatedToken(i+1, false)
		case unicode.IsSpace(r):
			pushAccumulatedToken(i, false)
		default:
			if bStart < 0 {
				bStart = i
			}
			// Continue to next character
		}
	}
	if eolCol < 0 {
		eolCol = len(coord.Line)
	}
	if quoted {
		return []Token{}, ParseError{
			Coord: FileCoord{
				Line:     coord.Line,
				LineNum:  coord.LineNum,
				ColStart: bStart,
				ColEnd:   eolCol,
			},
			Message: "unclosed quotes",
		}
	}
	// Flush any trailing token that hasn't been emitted yet.
	pushAccumulatedToken(eolCol, false)
	return toks, nil
}
