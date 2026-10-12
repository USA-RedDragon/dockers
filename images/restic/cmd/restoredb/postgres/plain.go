package postgres

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
)

// ErrUnsupportedDump is returned for a plain dump the restore cannot follow.
var ErrUnsupportedDump = errors.New("the plain dump has something restoredb cannot restore in one transaction")

// PlainFilter passes a plain pg_dump script on to psql while following it the
// way psql reads it: statements, quoted text, and the data lines of each COPY
// ... FROM stdin up to its \. line.
//
// pg_dump wraps large-object data in its own BEGIN; and COMMIT; lines. Under
// psql --single-transaction the BEGIN is only a warning, but the COMMIT would
// commit everything before it, and the rest of the dump would then run outside
// any transaction. PlainFilter turns those lines into comments when they stand
// alone between statements, outside COPY data, so the whole dump stays in the
// one transaction psql commits at the end. Any other statement that ends,
// rolls back or prepares a transaction is refused, and so is every psql
// meta-command but the \restrict and \unrestrict lines pg_dump writes, and the
// \. that ends COPY data.
//
// With Extensions set, a DROP, CREATE or COMMENT ON EXTENSION statement that
// starts a line between statements is dropped when the policy skips its
// extension, and stops the restore when the policy finds it missing. Skipped
// is told about each statement dropped.
type PlainFilter struct {
	Extensions *ExtensionPolicy
	Skipped    func(statement, reason string)

	lex       *lexer
	inCopy    bool
	lineStart bool
	skipping  bool
}

// NewPlainFilter returns a PlainFilter at the start of a dump.
func NewPlainFilter() *PlainFilter {
	return &PlainFilter{lex: newLexer(), lineStart: true}
}

// Copy copies src to dst as described on PlainFilter. It stops at the first
// statement it refuses, before passing on the line that would complete it.
func (f *PlainFilter) Copy(dst io.Writer, src io.Reader) error {
	in, ok := src.(*bufio.Reader)
	if !ok {
		in = bufio.NewReaderSize(src, 64<<10)
	}
	var held []byte
	for {
		chunk, err := in.ReadSlice('\n')
		full := errors.Is(err, bufio.ErrBufferFull)
		if held != nil || f.lineStart && full {
			held = append(held, chunk...)
			if full && len(held) < maxHeldLine {
				continue
			}
			chunk, held = held, nil
		}
		if len(chunk) > 0 {
			out, ferr := f.line(chunk, !full)
			if ferr != nil {
				return ferr
			}
			if _, werr := dst.Write(out); werr != nil {
				return werr
			}
		}
		switch {
		case err == nil, full:
		case errors.Is(err, io.EOF):
			return nil
		default:
			return err
		}
	}
}

// Ended reports whether the dump read so far ends between statements, outside
// COPY data, a string or a comment, as a complete dump does. A trailer that
// COPY rows or a string spell out does not end the dump.
func (f *PlainFilter) Ended() bool {
	return !f.inCopy && f.lex.idle()
}

// line handles chunk, a whole line when whole is set, or else part of one.
func (f *PlainFilter) line(chunk []byte, whole bool) ([]byte, error) {
	start := f.lineStart
	f.lineStart = chunk[len(chunk)-1] == '\n'
	if f.inCopy {
		if start && whole && isCopyEnd(chunk) {
			f.inCopy = false
		}
		return chunk, nil
	}
	if start && whole && f.lex.idle() {
		if bytes.Equal(trimEOL(chunk), []byte("BEGIN;")) || bytes.Equal(trimEOL(chunk), []byte("COMMIT;")) {
			return append([]byte("-- "), chunk...), nil
		}
		if restrictLine.Match(trimEOL(chunk)) {
			return chunk, nil
		}
	}
	if start && whole && f.lex.idle() && f.Extensions != nil {
		if name, ok := plainEntry(chunk); ok {
			switch f.Extensions.action(name) {
			case extSkip:
				f.skipping = true
				if f.Skipped != nil {
					f.Skipped(string(trimEOL(chunk)), f.Extensions.reason(name))
				}
			case extMissing:
				return nil, f.Extensions.missingError([]string{name})
			}
		}
	}
	if err := f.lex.feed(chunk); err != nil {
		return nil, fmt.Errorf("%w; at %.200q", err, chunk)
	}
	if f.lineStart && f.lex.copyPending {
		f.lex.copyPending = false
		f.inCopy = true
	}
	if f.skipping {
		if f.lineStart && f.lex.idle() {
			f.skipping = false
		}
		return nil, nil
	}
	return chunk, nil
}

// maxHeldLine is how much of a long line is gathered before it is passed on in
// parts. Every line PlainFilter has to see whole is shorter.
const maxHeldLine = 4 << 10

var restrictLine = regexp.MustCompile(`^\\(un)?restrict [^\s]+[ \t]*$`)

func trimEOL(line []byte) []byte {
	return bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
}

// isCopyEnd reports whether line ends COPY data the way psql sees it: \.
// alone, then a newline.
func isCopyEnd(line []byte) bool {
	return string(line) == "\\.\n" || string(line) == "\\.\r\n"
}
