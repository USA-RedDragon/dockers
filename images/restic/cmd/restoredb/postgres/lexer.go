package postgres

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ErrMetaCommand is returned for a psql meta-command in a plain dump other than
// \restrict and \unrestrict. psql runs it itself, and one such as \connect
// would carry on outside the restore's transaction.
var ErrMetaCommand = errors.New("the dump has a psql meta-command, which could run outside the restore's transaction")

// ErrTransactionControl is returned for a statement in a plain dump that would
// end or abandon the transaction the restore runs in.
var ErrTransactionControl = errors.New("the dump ends or rolls back a transaction itself, " +
	"so it cannot be restored in one transaction")

type lexState int

const (
	stNormal lexState = iota
	stLineComment
	stBlockComment
	stQuote
	stEscapeQuote
	stQuoteEnd
	stIdent
	stIdentEnd
	stDollar
)

type token int

const (
	tokNone token = iota
	tokIdent
	tokNumber
	tokJunk
)

const (
	maxWord = 32
	maxHead = 256
)

// lexer follows a plain SQL script the way psql's own lexer (psqlscan.l)
// splits it into statements: quoted strings and identifiers, dollar quotes,
// nested comments, parentheses, and the BEGIN ... END of a SQL-standard
// function body, inside which a semicolon does not end the statement. It
// remembers the start of each statement, so a statement can be checked once
// it ends.
type lexer struct {
	state      lexState
	quoted     lexState
	escaped    bool
	blockDepth int
	blockPrev  byte

	pending  byte
	tok      token
	word     []byte
	wordLen  int
	uAmp     bool
	dollar   []byte
	delim    []byte
	closed   []byte
	contKind lexState
	contLine bool

	parenDepth int
	beginDepth int
	idents     [8]byte
	identCount int

	empty      bool
	head       []byte
	stdStrings bool
	lit        []byte
	scsSeen    bool

	// copyPending is set by a statement that is COPY ... FROM STDIN, whose
	// data starts on the next line.
	copyPending bool
}

func newLexer() *lexer {
	return &lexer{empty: true, stdStrings: true}
}

// idle reports whether the lexer is between statements, outside any string,
// or block comment.
func (l *lexer) idle() bool {
	return (l.state == stNormal || l.state == stLineComment) && l.empty && l.pending == 0 && l.dollar == nil && !l.uAmp &&
		!l.copyPending && l.tok == tokNone
}

func isIdentStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= 0x80
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isNewline(c byte) bool { return c == '\n' || c == '\r' }

// feed lexes p, which may end anywhere, even inside a token.
func (l *lexer) feed(p []byte) error {
	for _, c := range p {
		if err := l.byte(c); err != nil {
			return err
		}
	}
	return nil
}

func (l *lexer) byte(c byte) error {
	switch l.state {
	case stLineComment:
		if isNewline(c) {
			l.state = stNormal
			l.contLine = true
		}
		return nil
	case stBlockComment:
		l.blockComment(c)
		return nil
	case stQuote, stEscapeQuote:
		l.addHead(c)
		switch {
		case l.escaped:
			l.escaped = false
		case c == '\\' && l.state == stEscapeQuote:
			l.escaped = true
		case c == '\'':
			l.quoted, l.state = l.state, stQuoteEnd
			return nil
		}
		l.literal(c)
		return nil
	case stQuoteEnd:
		if c == '\'' {
			l.addHead(c)
			l.literal(c)
			l.state = l.quoted
			return nil
		}
		l.named(l.lit)
		l.state = stNormal
		l.contKind, l.contLine = l.quoted, false
	case stIdent:
		l.addHead(c)
		if c == '"' {
			l.state = stIdentEnd
			return nil
		}
		l.literal(c)
		return nil
	case stIdentEnd:
		if c == '"' {
			l.addHead(c)
			l.literal(c)
			l.state = stIdent
			return nil
		}
		l.named(l.lit)
		l.state, l.contKind = stNormal, stNormal
	case stDollar:
		l.addHead(c)
		l.literal(c)
		l.dollarQuoted(c)
		if l.state == stNormal && len(l.lit) < maxLiteral {
			l.named(l.lit[:max(0, len(l.lit)-len(l.closed))])
		}
		return nil
	}
	return l.normal(c)
}

func (l *lexer) blockComment(c byte) {
	switch {
	case l.blockPrev == '/' && c == '*':
		l.blockDepth++
		c = 0
	case l.blockPrev == '*' && c == '/':
		if l.blockDepth == 0 {
			l.state = stNormal
		} else {
			l.blockDepth--
		}
		c = 0
	}
	l.blockPrev = c
}

func isDollarTagByte(c byte, first bool) bool {
	return isIdentStart(c) || !first && isDigit(c)
}

// dollarQuoted looks for the closing delimiter of a dollar-quoted string.
func (l *lexer) dollarQuoted(c byte) {
	switch {
	case c == '$' && l.dollar == nil:
		l.dollar = []byte{'$'}
	case c == '$':
		if bytes.Equal(append(l.dollar, '$'), l.delim) {
			l.state, l.dollar, l.closed, l.delim = stNormal, nil, l.delim, nil
			return
		}
		l.dollar = []byte{'$'}
	case l.dollar != nil && isDollarTagByte(c, len(l.dollar) == 1):
		l.dollar = append(l.dollar, c)
	default:
		l.dollar = nil
	}
}

func (l *lexer) normal(c byte) error {
	if l.copyPending {
		if c != '\n' && c != ' ' && c != '\t' && c != '\r' {
			return fmt.Errorf("%w: text follows COPY FROM stdin on the same line", ErrUnsupportedDump)
		}
		return nil
	}
	if l.dollar != nil {
		if done, err := l.dollarOpen(c); done || err != nil {
			return err
		}
	}
	if l.pending != 0 {
		p := l.pending
		l.pending = 0
		switch {
		case p == '-' && c == '-':
			l.state = stLineComment
			return nil
		case p == '/' && c == '*':
			l.state, l.blockDepth, l.blockPrev = stBlockComment, 0, 0
			l.contKind = stNormal
			return nil
		}
		l.contKind = stNormal
		l.token(p)
	}
	if l.uAmp {
		l.uAmp = false
		switch c {
		case '\'':
			l.token('U')
			l.token('&')
			l.openQuote(stQuote, c)
			return nil
		case '"':
			l.token('U')
			l.token('&')
			l.addHead(c)
			l.state, l.lit = stIdent, l.lit[:0]
			return nil
		}
		l.token('U')
		l.token('&')
	}
	if l.tok != tokNone {
		if l.continueWord(c) {
			return nil
		}
		if l.endWord(c) {
			return nil
		}
	}
	return l.start(c)
}

// dollarOpen continues a possible opening $tag$. It reports whether c was used.
func (l *lexer) dollarOpen(c byte) (bool, error) {
	switch {
	case c == '$':
		l.addHead('$')
		l.delim = append(l.dollar, '$')
		l.dollar = nil
		l.state, l.lit = stDollar, l.lit[:0]
		return true, nil
	case isDollarTagByte(c, len(l.dollar) == 1):
		l.dollar = append(l.dollar, c)
		return true, nil
	}
	tag := l.dollar[1:]
	l.dollar = nil
	l.token('$')
	if len(tag) == 0 {
		return false, nil
	}
	for _, b := range tag {
		if err := l.normal(b); err != nil {
			return true, err
		}
	}
	return true, l.normal(c)
}

// continueWord extends the identifier or number being read with c.
func (l *lexer) continueWord(c byte) bool {
	switch l.tok {
	case tokIdent, tokJunk:
		if isIdentStart(c) || isDigit(c) || c == '$' {
			l.appendWord(c)
			return true
		}
	case tokNumber:
		switch {
		case isDigit(c) || c == '_':
			l.appendWord(c)
			return true
		case isIdentStart(c):
			l.tok = tokJunk
			l.appendWord(c)
			return true
		}
	}
	return false
}

func (l *lexer) appendWord(c byte) {
	l.addHead(c)
	l.wordLen++
	if len(l.word) < maxWord {
		l.word = append(l.word, c)
	}
}

// endWord finishes the word before c. It reports whether c was used, as the
// quote that a one-letter prefix such as E or U& opens.
func (l *lexer) endWord(c byte) bool {
	tok, word := l.tok, l.word
	l.tok, l.word = tokNone, l.word[:0]
	if tok != tokIdent {
		return false
	}
	if l.wordLen == 1 {
		switch w := word[0] | 0x20; {
		case c == '\'' && w == 'e':
			l.openQuote(stEscapeQuote, c)
			return true
		case c == '\'' && (w == 'b' || w == 'x'):
			l.openQuote(stQuote, c)
			return true
		case c == '\'' && w == 'n':
			return false
		case c == '&' && w == 'u':
			l.uAmp = true
			l.headTrim(1)
			return true
		}
	}
	l.named(word)
	l.trackIdent(strings.ToLower(string(word)))
	return false
}

// start begins the token that c starts outside any word.
func (l *lexer) start(c byte) error {
	cont, contLine := l.contKind, l.contLine
	if !isSpace(c) && c != '-' {
		l.contKind = stNormal
	}
	switch {
	case isSpace(c):
		if isNewline(c) {
			l.contLine = true
		}
		if !l.empty {
			l.addHead(' ')
		}
	case isIdentStart(c):
		l.tok, l.wordLen = tokIdent, 0
		l.empty = false
		l.appendWord(c)
	case isDigit(c):
		l.tok, l.wordLen = tokNumber, 0
		l.empty = false
		l.appendWord(c)
	case c == '$':
		l.empty = false
		l.dollar = []byte{'$'}
	case c == '\'':
		kind := stQuote
		switch {
		case cont != stNormal && contLine:
			kind = cont
		case !l.stdStrings:
			kind = stEscapeQuote
		}
		l.openQuote(kind, c)
	case c == '"':
		l.empty = false
		l.addHead(c)
		l.state, l.lit = stIdent, l.lit[:0]
	case c == '-' || c == '/':
		l.pending = c
	case c == '\\':
		return ErrMetaCommand
	case c == ';':
		l.token(c)
		if l.parenDepth == 0 && l.beginDepth == 0 {
			return l.endStatement()
		}
	case c == '(':
		l.parenDepth++
		l.token(c)
	case c == ')':
		if l.parenDepth > 0 {
			l.parenDepth--
		}
		l.token(c)
	default:
		l.token(c)
	}
	return nil
}

func (l *lexer) openQuote(kind lexState, c byte) {
	l.lit = l.lit[:0]
	l.empty = false
	l.addHead(c)
	l.state, l.escaped = kind, false
}

func (l *lexer) token(c byte) {
	l.empty = false
	l.addHead(c)
}

func (l *lexer) addHead(c byte) {
	if len(l.head) < maxHead {
		l.head = append(l.head, c)
	}
}

// scsName is the setting that decides how psql reads a backslash in a quoted
// string. psql follows the server's value, so a statement that names it, as a
// word or as a whole string such as set_config's first argument, is refused
// unless endStatement can read what it sets.
const scsName = "standard_conforming_strings"

const maxLiteral = 64

// literal collects the text of the quoted string or identifier being read.
func (l *lexer) literal(c byte) {
	if len(l.lit) < maxLiteral {
		l.lit = append(l.lit, c)
	}
}

func (l *lexer) named(name []byte) {
	if strings.EqualFold(string(name), scsName) {
		l.scsSeen = true
	}
}

func (l *lexer) headTrim(n int) {
	if len(l.head) >= n {
		l.head = l.head[:len(l.head)-n]
	}
}

// trackIdent mirrors psqlscan_track_identifier: it records the first words of
// a statement, to tell COPY ... FROM STDIN and CREATE FUNCTION or PROCEDURE,
// and counts BEGIN and END within the latter.
func (l *lexer) trackIdent(word string) {
	if l.parenDepth != 0 {
		return
	}
	if l.identCount == 0 {
		l.idents = [8]byte{}
	}
	if l.identCount < len(l.idents) {
		switch word {
		case "create", "function", "procedure", "or", "replace":
			l.idents[l.identCount] = word[0]
		case "copy", "from", "stdin", "stdout":
			l.idents[l.identCount] = word[0] - 'a' + 'A'
		}
		l.identCount++
	}
	id := l.idents
	if id[0] == 'c' && (id[1] == 'f' || id[1] == 'p' || id[1] == 'o' && id[2] == 'r' && (id[3] == 'f' || id[3] == 'p')) {
		switch word {
		case "begin":
			l.beginDepth++
		case "case":
			if l.beginDepth >= 1 {
				l.beginDepth++
			}
		case "end":
			if l.beginDepth > 0 {
				l.beginDepth--
			}
		}
	}
}

func (l *lexer) copyFromStdin() bool {
	if l.idents[0] != 'C' {
		return false
	}
	for i := 1; i < len(l.idents)-1; i++ {
		if l.idents[i] == 'F' {
			return l.idents[i+1] == 'S'
		}
	}
	return false
}

// endStatement checks the statement that a semicolon just ended and resets
// the per-statement state, as psql does before it reads the next one.
func (l *lexer) endStatement() error {
	head := strings.ToLower(string(bytes.TrimSpace(l.head)))
	copyIn := l.identCount > 0 && l.copyFromStdin()
	l.head, l.empty, l.parenDepth, l.beginDepth, l.identCount = l.head[:0], true, 0, 0, 0
	if word, ok := transactionControl(head); ok {
		return fmt.Errorf("%w: %s", ErrTransactionControl, word)
	}
	seen := l.scsSeen
	l.scsSeen = false
	on, ok := standardConformingStrings(head)
	switch {
	case ok:
		l.stdStrings = on
	case seen:
		return fmt.Errorf("%w: it changes %s in a way restoredb does not follow", ErrUnsupportedDump, scsName)
	}
	l.copyPending = copyIn
	return nil
}

// transactionControl reports whether a statement, by its start, ends,
// abandons or prepares the current transaction or starts another.
func transactionControl(head string) (string, bool) {
	word := firstWord(head)
	switch word {
	case "begin", "start", "commit", "end", "rollback", "abort":
		return strings.ToUpper(word), true
	case "prepare":
		if next := firstWord(strings.TrimSpace(head[len(word):])); next == "transaction" {
			return "PREPARE TRANSACTION", true
		}
	}
	return "", false
}

func firstWord(s string) string {
	end := 0
	for end < len(s) && (isIdentStart(s[end]) || end > 0 && (isDigit(s[end]) || s[end] == '$')) {
		end++
	}
	return s[:end]
}

// standardConformingStrings reads a SET or RESET of standard_conforming_strings,
// which changes how psql reads a backslash in a quoted string.
func standardConformingStrings(head string) (on, ok bool) {
	fields := strings.FieldsFunc(strings.TrimSuffix(head, ";"), func(r rune) bool {
		return r == ' ' || r == '=' || r == '\'' || r == '"'
	})
	if len(fields) >= 2 && fields[0] == "reset" {
		return true, fields[1] == "standard_conforming_strings" || fields[1] == "all"
	}
	if len(fields) < 3 || fields[0] != "set" {
		return false, false
	}
	fields = fields[1:]
	if fields[0] == "session" || fields[0] == "local" {
		fields = fields[1:]
	}
	if len(fields) < 2 || fields[0] != "standard_conforming_strings" {
		return false, false
	}
	value := fields[1]
	if value == "to" && len(fields) > 2 {
		value = fields[2]
	}
	switch value {
	case "on", "true", "yes", "1", "default":
		return true, true
	case "off", "false", "no", "0":
		return false, true
	}
	return false, false
}
