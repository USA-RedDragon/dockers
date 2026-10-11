package mariadb

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	tailSize = 4 << 10
	// headSize is how much of a statement is read before it is classified.
	headSize = 4 << 10
	// holdSize is how much of a statement, comments included, is held back
	// while it is classified.
	holdSize = 64 << 10
)

type stmtMode int

const (
	modeHold stmtMode = iota
	modePass
	modeSkip
)

// dumpFilter copies a dump a statement at a time, split the way the mariadb
// client splits it. It leaves out the statements that name a database, so
// every statement runs in the client's own database, and stops at the first
// statement the swap cannot handle: one that writes to a table in another
// database, changes the server, or starts an object RENAME TABLE cannot move.
// Each statement is held back until it is classified, so a refused statement
// never reaches the client. It keeps the end of the dump for DumpComplete.
type dumpFilter struct {
	// database is the database the dump's USE and CREATE DATABASE lines name.
	database  string
	tail      []byte
	lex       clientLexer
	active    bool
	mode      stmtMode
	head      []byte
	held      []byte
	out       []byte
	sawObject bool
	names     qualifiedNames
	values    valueWatch
	modes     sqlModes
	isSet     bool
	// createTable is set once the statement is known to be a CREATE TABLE,
	// where a dump may name another database in a foreign key.
	createTable bool
	decided     bool
}

func (f *dumpFilter) keepTail(p []byte) {
	f.tail = append(f.tail, p...)
	if over := len(f.tail) - tailSize; over > 0 {
		f.tail = append(f.tail[:0], f.tail[over:]...)
	}
}

// complete reports whether the dump ended between statements with the dump
// tool's trailer.
func (f *dumpFilter) complete() bool {
	return f.lex.idle() && DumpComplete(f.tail)
}

func (f *dumpFilter) copy(dst io.Writer, src *bufio.Reader) error {
	var carry []byte
	for {
		chunk, err := src.ReadSlice('\n')
		full := errors.Is(err, bufio.ErrBufferFull)
		f.keepTail(chunk)
		if carry != nil {
			chunk = append(carry, chunk...)
			carry = nil
		}
		f.out = f.out[:0]
		n, ferr := f.lex.scan(chunk, !full, f.span)
		if ferr != nil {
			return ferr
		}
		if n < len(chunk) {
			carry = append([]byte(nil), chunk[n:]...)
		}
		if len(f.out) > 0 {
			if _, werr := dst.Write(f.out); werr != nil {
				return werr
			}
		}
		switch {
		case err == nil, full:
		case errors.Is(err, io.EOF):
			if f.active && f.mode == modeHold {
				f.out = f.out[:0]
				if err := f.decide(true); err != nil {
					return err
				}
				_, werr := dst.Write(f.out)
				return werr
			}
			return nil
		default:
			return err
		}
	}
}

// span routes bytes of one kind according to the statement they are part of.
func (f *dumpFilter) span(p []byte, kind byteKind) error {
	if kind == byteText && !f.active {
		f.active, f.mode = true, modeHold
		f.head, f.held = f.head[:0], f.held[:0]
		f.names.reset()
		f.values.reset(false)
		f.createTable, f.decided, f.isSet = false, false, false
	}
	if !f.active {
		f.out = append(f.out, p...)
		return nil
	}
	cut := len(p)
	switch kind {
	case byteText:
		cut = min(len(p), max(0, headSize-len(f.head)))
		f.head = append(f.head, p[:cut]...)
		f.names.text(p, f.lex.inString)
		f.values.text(p, f.lex.inString)
	case byteEnd:
		// The semicolon is not part of the statement, but it ends its last
		// word, which the watcher has to see.
		f.names.text(p, 0)
	case byteGap:
		if len(f.head) < headSize {
			f.head = append(f.head, ' ')
		}
		f.names.byte(' ', 0)
	}
	if f.decided && f.mode == modePass {
		if err := f.checkNames(); err != nil {
			return err
		}
		if err := f.checkValues(); err != nil {
			return err
		}
	}
	switch f.mode {
	case modeHold:
		f.held = append(f.held, p...)
	case modePass:
		f.out = append(f.out, p...)
	}
	switch {
	case kind == byteEnd:
		f.active = false
		var err error
		if f.mode == modeHold {
			err = f.decide(true)
		}
		if f.names.sawUserVar && !f.isSet {
			f.modes.forget()
		}
		return err
	case f.mode == modeHold && (len(f.head) >= headSize || len(f.held) >= holdSize):
		if err := f.decide(false); err != nil {
			return err
		}
		return f.checkRest(p, cut)
	}
	return nil
}

// checkRest hands the value watch the part of the run that decided the
// statement past the head: the watch only starts once the statement is
// decided, and allowStatement saw no more than the head.
func (f *dumpFilter) checkRest(p []byte, cut int) error {
	if !f.values.on || cut >= len(p) {
		return nil
	}
	start := cut
	for start > 0 && isWordByte(p[start-1]) && isWordByte(p[cut]) {
		start--
	}
	f.values.text(p[start:], f.lex.inString)
	return f.checkValues()
}

// decide classifies the statement held so far and passes it on, leaves it
// out, or refuses it.
func (f *dumpFilter) decide(complete bool) error {
	held := f.held
	f.held = f.held[:0]
	kind, arg, shape := allowStatement(string(f.head), complete)
	toks := tokenize(skipLeadingComments(strings.TrimLeft(string(f.head), " \t\r\n")))
	f.isSet = len(toks) > 0 && toks[0].kind == tokWord && strings.EqualFold(toks[0].text, "SET")
	if f.isSet && complete && kind == stmtPass {
		if err := f.modes.set(toks[1:], string(f.head)); err != nil {
			return err
		}
	}
	f.createTable, f.decided = shape == shapeTableDef, true
	f.values.reset(shape == shapeValues)
	if err := f.checkNames(); err != nil && kind != stmtRefused && kind != stmtServer {
		return err
	}
	if err := f.checkValues(); err != nil {
		return err
	}
	switch kind {
	case stmtRefused:
		return fmt.Errorf("%w: the dump has %s", ErrUnsupportedObjects, arg)
	case stmtUnsupported:
		return fmt.Errorf("%w: %.200s", ErrUnsupportedStatement, arg)
	case stmtServer:
		return fmt.Errorf("%w: %.200s", ErrServerStatement, arg)
	case stmtOtherDatabase:
		return fmt.Errorf("%w: %s", ErrOtherDatabase, arg)
	case stmtDatabase:
		if !complete {
			return fmt.Errorf("%w: %.200s", ErrUnsupportedStatement, f.head)
		}
		if err := f.sawDatabase(arg); err != nil {
			return err
		}
		f.mode = modeSkip
		return nil
	case stmtObject:
		f.sawObject = true
	}
	f.out = append(f.out, held...)
	f.mode = modePass
	return nil
}

// checkValues refuses an INSERT whose value list holds something a dump does
// not write, such as a function call or a name.
func (f *dumpFilter) checkValues() error {
	if f.values.bad == "" {
		return nil
	}
	return fmt.Errorf("%w: a value a dump does not write, %.40q, in %.200s", ErrUnsupportedStatement,
		f.values.bad, f.head)
}

// checkNames refuses a statement that names something in another database:
// it could write past the staging database or run code there. CREATE TABLE
// is the exception, since mariadb-dump writes names qualified with a database
// in a foreign key to another database and in a sequence default, and creating
// a table runs nothing, unless it fills the table with a SELECT.
func (f *dumpFilter) checkNames() error {
	switch {
	case f.names.reach != "":
		return fmt.Errorf("%w: %s reaches tables outside it: %.200s", ErrOtherDatabase, f.names.reach, f.head)
	case f.createTable && f.names.sawSelect:
		return fmt.Errorf("%w: a CREATE TABLE that fills the table with a SELECT: %.200s",
			ErrUnsupportedStatement, f.head)
	case !f.createTable && f.names.qualifier != "":
		return fmt.Errorf("%w: it names something in %s: %.200s", ErrOtherDatabase,
			QuoteIdent(f.names.qualifier), f.head)
	}
	return nil
}

func (f *dumpFilter) sawDatabase(name string) error {
	switch {
	case f.database == "" && f.sawObject:
		return fmt.Errorf("%w: it names %s only after its first table", ErrMultipleDatabases, QuoteIdent(name))
	case f.database == "":
		f.database = name
	case f.database != name:
		return fmt.Errorf("%w: it names %s and %s", ErrMultipleDatabases, QuoteIdent(f.database), QuoteIdent(name))
	}
	return nil
}
