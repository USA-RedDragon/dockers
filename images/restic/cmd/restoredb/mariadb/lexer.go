package mariadb

import (
	"bytes"
	"errors"
	"fmt"
)

// ErrClientCommand is returned for a mariadb client command in a dump, other
// than the \- that turns on the client's sandbox mode.
var ErrClientCommand = errors.New("the dump has a mariadb client command")

type byteKind int

const (
	// byteSkip is a byte the client does not send: a comment, or whitespace
	// before a statement.
	byteSkip byteKind = iota
	// byteText is part of the statement the client sends.
	byteText
	// byteEnd is the semicolon that ends a statement.
	byteEnd
	// byteGap is a byte the client does not send, but where it leaves a gap
	// between the words around it: the end of a comment.
	byteGap
)

// clientLexer splits a dump into statements the way the mariadb client does
// in binary mode (add_line in client/mysql.cc): quoted strings and
// identifiers, # and -- comments, /* */ comments that it drops, and the
// executable /*! */ comments it sends. The delimiter is always a semicolon,
// since a DELIMITER command is refused.
type clientLexer struct {
	inString  byte
	escaped   bool
	mlComment bool
	// ssComment is set inside an executable /*! comment. The client tracks it
	// per line, so it ends with the line even where the comment does not.
	ssComment   bool
	lineComment bool
	inStatement bool
	// ssState follows the version of an executable comment, to tell one the
	// server runs from one it may skip.
	ssState     int
	ssVersion   int
	ssSkippable bool
}

const (
	ssIdle = iota
	ssOpen
	ssBody
)

// maxCheckedVersion is the highest version an executable comment may name and
// still be one every server restoredb supports runs: 10.5 is the oldest, since
// the staging databases carry a comment. A comment above it may be skipped by
// the server while the restore's lexer reads its content as code, so its
// content may not hold a quote or a semicolon, which could move where the
// server thinks the statement ends.
const maxCheckedVersion = 100500

// idle reports whether the lexer is between statements, outside any string or
// comment.
func (l *clientLexer) idle() bool {
	return l.inString == 0 && !l.escaped && !l.mlComment && !l.inStatement
}

func isClientSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

// scan classifies the bytes of buf, a whole line when whole is set or else
// part of one, and calls emit for each run of bytes of one kind. It returns how many bytes it used:
// fewer than len(buf) only when buf is part of a line and a byte at its end
// cannot be told apart without the bytes that follow.
func (l *clientLexer) scan(buf []byte, whole bool, emit func([]byte, byteKind) error) (int, error) {
	at := func(i int) (byte, bool) {
		if i < len(buf) {
			return buf[i], true
		}
		return 0, whole
	}
	i := 0
	for i < len(buf) {
		n, err := l.step(buf, i, at, emit)
		if err != nil {
			return i, err
		}
		if n == 0 {
			return i, nil
		}
		i += n
		if buf[i-1] == '\n' {
			l.ssComment = false
		}
	}
	return i, nil
}

// step handles the byte at i and returns how many bytes it used, or 0 when it
// needs bytes past the end of a partial line.
func (l *clientLexer) step(buf []byte, i int, at func(int) (byte, bool),
	emit func([]byte, byteKind) error) (int, error) {
	c := buf[i]
	if n := l.run(buf[i:]); n > 0 {
		kind := byteText
		if l.lineComment || l.mlComment {
			kind = byteSkip
		}
		return n, emit(buf[i:i+n], kind)
	}
	if err := l.skippable(c); err != nil {
		return 0, err
	}
	switch {
	case l.lineComment && c == '\n':
		l.lineComment = false
		return 1, emit(buf[i:i+1], byteGap)
	case l.lineComment:
		return 1, emit(buf[i:i+1], byteSkip)
	case l.escaped:
		l.escaped = false
		return 1, emit(buf[i:i+1], byteText)
	case !l.mlComment && c == '\\' && l.inString != '`':
		return l.backslash(buf, i, at, emit)
	case !l.mlComment && l.inString == 0 && c == ';':
		l.inStatement = false
		return 1, emit(buf[i:i+1], byteEnd)
	}
	if n, done, err := l.comment(buf, i, at, emit); done {
		return n, err
	}
	return l.plain(buf, i, at, emit)
}

// backslash handles a backslash outside a backquoted name: an escape in a
// string, or else a client command.
func (l *clientLexer) backslash(buf []byte, i int, at func(int) (byte, bool),
	emit func([]byte, byteKind) error) (int, error) {
	if l.inString != 0 {
		l.escaped = true
		return 1, emit(buf[i:i+1], byteText)
	}
	next, ok := at(i + 1)
	if !ok {
		return 0, nil
	}
	switch next {
	case 'N':
		l.inStatement = true
		return 2, emit(buf[i:i+2], byteText)
	case '-':
		return 2, emit(buf[i:i+2], byteSkip)
	}
	return 0, fmt.Errorf("%w: \\%c", ErrClientCommand, next)
}

// comment handles the start of a comment, or the end of a /* */ comment, at i.
// done is false when there is none there.
func (l *clientLexer) comment(buf []byte, i int, at func(int) (byte, bool),
	emit func([]byte, byteKind) error) (n int, done bool, err error) {
	c := buf[i]
	if !l.mlComment && l.inString == 0 && (c == '#' || c == '-') {
		comment, ok, cerr := l.lineCommentAt(c, i, at)
		switch {
		case cerr != nil:
			return 0, true, cerr
		case !ok:
			return 0, true, nil
		case comment:
			l.lineComment = true
			return 1, true, emit(buf[i:i+1], byteSkip)
		}
	}
	if l.inString == 0 && c == '/' {
		next, ok1 := at(i + 1)
		next2, ok2 := at(i + 2)
		next3, ok3 := at(i + 3)
		if !ok1 || next == '*' && (!ok2 || next2 == 'M' && !ok3) {
			return 0, true, nil
		}
		if next == '*' && next2 != '!' && (next2 != 'M' || next3 != '!') {
			l.mlComment = true
			return 2, true, emit(buf[i:i+2], byteSkip)
		}
	}
	if l.mlComment && !l.ssComment && c == '*' {
		next, ok := at(i + 1)
		if !ok {
			return 0, true, nil
		}
		if next == '/' {
			l.mlComment = false
			return 2, true, errors.Join(emit(buf[i:i+1], byteSkip), emit(buf[i+1:i+2], byteGap))
		}
	}
	return 0, false, nil
}

// run returns the length of the run of bytes at the start of buf that cannot
// change the lexer's state: the inside of a comment or a quoted string, or
// statement text up to the next byte that could start or end one.
func (l *clientLexer) run(buf []byte) int {
	var stops string
	switch {
	case l.escaped:
		return 0
	case l.ssState == ssOpen:
		// The version of an executable comment is read a byte at a time.
		return 0
	case l.lineComment:
		stops = "\n"
	case l.mlComment && !l.ssComment:
		stops = "*/"
	case l.mlComment:
		return 0
	case l.inString == '`':
		stops = "`"
	case l.inString != 0:
		stops = string(l.inString) + "\\"
	case l.inStatement:
		stops = "\\;#-/*'\"`"
	default:
		return 0
	}
	if n := bytes.IndexAny(buf, stops); n >= 0 {
		return n
	}
	return len(buf)
}

// version follows the digits of an executable comment's version and refuses a
// quote or a semicolon inside one the server may skip.
func (l *clientLexer) version(c byte) error {
	switch {
	case l.ssState == ssOpen && isDigit(c):
		l.ssVersion = l.ssVersion*10 + int(c-'0')
	case l.ssState == ssOpen && (c == '*' || c == '!' || c == 'M' || c == '/'):
	case l.ssState == ssOpen:
		l.ssState = ssBody
		l.ssSkippable = l.ssVersion > maxCheckedVersion
		fallthrough
	case l.ssState == ssBody:
		return l.skippable(c)
	}
	return nil
}

// skippable refuses a quote or a semicolon inside an executable comment the
// server may skip: the restore could not tell where such a statement ends.
func (l *clientLexer) skippable(c byte) error {
	if l.ssState != ssBody || !l.ssSkippable {
		return nil
	}
	if c == '\'' || c == '"' || c == '`' || c == ';' {
		return fmt.Errorf("%w: a quote or a semicolon inside an executable comment the server may skip, "+
			"so restoredb cannot tell where the statement ends", ErrUnsupportedStatement)
	}
	return nil
}

// lineCommentAt reports whether the # or - at i starts a comment to the end of
// the line: # always, -- when whitespace or the end of the line follows, or
// at the start of a statement. Inside a statement, -- followed by a control
// byte is refused: the server takes it as a comment where the client does not,
// so where the statement ends would differ.
func (l *clientLexer) lineCommentAt(c byte, i int, at func(int) (byte, bool)) (comment, ok bool, err error) {
	if c == '#' {
		return true, true, nil
	}
	next, ok := at(i + 1)
	if !ok {
		return false, false, nil
	}
	if next != '-' {
		return false, true, nil
	}
	if !l.inStatement {
		return true, true, nil
	}
	next2, ok := at(i + 2)
	if !ok {
		return false, false, nil
	}
	if isControlByte(next2) {
		return false, true, fmt.Errorf("%w: -- followed by a control byte, which the server takes as a comment "+
			"and the client does not, so restoredb cannot tell where the statement ends", ErrUnsupportedStatement)
	}
	return next2 == 0 || isClientSpace(next2), true, nil
}

// isControlByte reports whether c is one of the control bytes the server takes
// as whitespace after --, which the client does not.
func isControlByte(c byte) bool {
	return c != 0 && (c < 0x20 && !isClientSpace(c) || c == 0x7f)
}

func (l *clientLexer) plain(buf []byte, i int, at func(int) (byte, bool),
	emit func([]byte, byteKind) error) (int, error) {
	c := buf[i]
	if l.inString == 0 && (c == '/' || c == '*') {
		next, ok := at(i + 1)
		if !ok {
			return 0, nil
		}
		next2, ok2 := at(i + 2)
		next3, ok3 := at(i + 3)
		switch {
		case c == '/' && next == '*' && !ok2, c == '/' && next == '*' && next2 == 'M' && !ok3:
			return 0, nil
		case c == '/' && next == '*' && (next2 == '!' || next2 == 'M' && next3 == '!'):
			l.ssComment = true
			l.ssState, l.ssVersion = ssOpen, 0
		case c == '*' && next == '/' && l.ssComment:
			l.ssComment = false
			l.ssState, l.ssSkippable = ssIdle, false
		}
	}
	if err := l.version(c); err != nil {
		return 0, err
	}
	switch {
	case c == l.inString:
		l.inString = 0
	case !l.mlComment && l.inString == 0 && (c == '\'' || c == '"' || c == '`'):
		l.inString = c
	}
	if l.mlComment || !l.inStatement && isClientSpace(c) {
		return 1, emit(buf[i:i+1], byteSkip)
	}
	l.inStatement = true
	return 1, emit(buf[i:i+1], byteText)
}
