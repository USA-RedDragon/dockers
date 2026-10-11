package xtrace

import (
	"io"
	"strconv"
	"strings"
)

func XTrace(out io.Writer, args []string) {
	_, _ = io.WriteString(out, XTraceString(args))
}

func XTraceString(args []string) string {
	var n int
	for _, arg := range args {
		n += len(arg) + 1
		if strings.Contains(arg, " ") {
			n += 2
		}
	}

	var b strings.Builder
	b.Grow(n + 2)
	b.WriteByte('+')

	for _, arg := range args {
		b.WriteByte(' ')
		if strings.Contains(arg, " ") {
			arg = strconv.Quote(arg)
		}
		b.WriteString(arg)
	}

	b.WriteByte('\n')

	return b.String()
}
