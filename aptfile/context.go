package aptfile

import (
	"fmt"
	"strings"
)

type FileCoord struct {
	Line     string
	LineNum  int
	ColStart int
	ColEnd   int
}

func (c FileCoord) Text() string {
	return c.Line[c.ColStart:c.ColEnd]
}

// Format a message as an annotation on a line.
func (c FileCoord) LineAnnotated(message string) string {
	lineNum := fmt.Sprintf("%d", c.LineNum)
	return fmt.Sprintf(
		"%s | %s\n%s   %s%s %s",
		lineNum,
		c.Line,
		strings.Repeat(" ", len(lineNum)),
		strings.Repeat(" ", c.ColStart),
		strings.Repeat("^", max(c.ColEnd-c.ColStart, 1)),
		message,
	)
}
