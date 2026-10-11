package mariadb

import (
	"slices"
	"strings"
)

// shape says how much of a statement the head holds, and so what the rest of
// it may be.
type shape int

const (
	verbSelect   = "SELECT"
	verbCommit   = "COMMIT"
	kindView     = "VIEW"
	kindDatabase = "DATABASE"
	kindSchema   = "SCHEMA"
	verbStart    = "START"
)

const (
	// shapeWhole is a statement the head holds entirely. One longer than the
	// head is refused.
	shapeWhole shape = iota
	// shapeValues is an INSERT whose head ends in VALUES. The rest is a list
	// of values, checked token by token as it streams.
	shapeValues
	// shapeTableDef is a CREATE TABLE. The rest is a column and option list,
	// which runs nothing; the name and engine watchers check it as it streams.
	shapeTableDef
)

// allowedVerbs are the statements mariadb-dump writes. Anything else is
// refused: this list, not a list of known-bad statements, is what keeps a dump
// from reaching past the staging database.
var allowedVerbs = []string{
	"USE", verbCreate, verbDrop, verbAlter, "INSERT", "REPLACE", "LOCK", "UNLOCK", "SET", "DO", verbSelect,
	verbCommit, verbStart, "DELIMITER",
}

// sessionVars are the session variables a dump may set. A dump sets them to
// make the server read it as the dump tool meant, and sets them back at the
// end from the @OLD_ or @saved_ variables it saved them in.
var sessionVars = []string{
	"CHARACTER_SET_CLIENT", "CHARACTER_SET_RESULTS", "CHARACTER_SET_CONNECTION", "CHARACTER_SET_DATABASE",
	"CHARACTER_SET_SERVER", "COLLATION_CONNECTION", "COLLATION_DATABASE", "COLLATION_SERVER",
	"TIME_ZONE", "SQL_MODE", "UNIQUE_CHECKS", "FOREIGN_KEY_CHECKS", "CHECK_CONSTRAINT_CHECKS",
	"AUTOCOMMIT", "SQL_NOTES", "SQL_QUOTE_SHOW_CREATE", "NOTE_VERBOSITY", "SQL_LOG_BIN", "LOCK_WAIT_TIMEOUT",
	"INNODB_STRICT_MODE", "SYSTEM_VERSIONING_INSERT_HISTORY", "OLD_ALTER_TABLE", "ALTER_ALGORITHM",
	"DEFAULT_STORAGE_ENGINE", "DEFAULT_TMP_STORAGE_ENGINE", "STORAGE_ENGINE",
}

// allowStatement reads a statement as the client sends it and reports whether
// it is one a dump may hold. complete is false when stmt is only the start of
// a longer statement.
func allowStatement(stmt string, complete bool) (stmtKind, string, shape) {
	s := strings.TrimLeft(stmt, " \t\r\n")
	switch {
	case hasPrefixFold(s, "DELIMITER"):
		return stmtRefused, objectsTriggers, shapeWhole
	case strings.HasPrefix(s, "/*!50001 "):
		return stmtRefused, objectsViews, shapeWhole
	}

	toks := tokenize(skipLeadingComments(s))
	if len(toks) == 0 {
		// Only comments, which run nothing.
		return stmtPass, "", shapeWhole
	}
	if toks[0].kind != tokWord {
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	verb := strings.ToUpper(toks[0].text)
	if !slices.Contains(allowedVerbs, verb) {
		if slices.Contains(serverVerbs, verb) {
			return stmtServer, s, shapeWhole
		}
		return stmtUnsupported, notAllowed(s), shapeWhole
	}

	kind, arg, sh := allowVerb(verb, toks, s)
	if kind == stmtUnsupported && !complete {
		// The statement may only look wrong because the head cut it short.
		return stmtUnsupported, tooLong + s, sh
	}
	if sh == shapeWhole && !complete {
		return stmtUnsupported, tooLong + s, sh
	}
	return kind, arg, sh
}

func notAllowed(s string) string {
	return "not a statement a mariadb-dump holds: " + s
}

func allowVerb(verb string, toks []token, s string) (stmtKind, string, shape) {
	switch verb {
	case "USE":
		if name, ok := identAt(toks, 1); ok && len(toks) == 2 {
			return stmtDatabase, name, shapeWhole
		}
		return stmtUnsupported, s, shapeWhole
	case verbCreate:
		return allowCreate(toks, s)
	case verbDrop:
		return allowDrop(toks, s)
	case verbAlter:
		return allowAlter(toks, s)
	case "INSERT", "REPLACE":
		return allowInsert(toks, s)
	case "LOCK":
		return allowLock(toks, s)
	case "UNLOCK":
		if len(toks) == 2 && wordAt(toks, 1, "TABLES") {
			return stmtPass, "", shapeWhole
		}
		return stmtUnsupported, notAllowed(s), shapeWhole
	case "SET":
		return allowSet(toks, s)
	case "DO", verbSelect:
		return allowSequenceCall(toks, s)
	case verbCommit, verbStart:
		return allowTransaction(verb, toks, s)
	}
	return stmtUnsupported, notAllowed(s), shapeWhole
}

// allowCreate takes CREATE TABLE, CREATE SEQUENCE and the CREATE DATABASE of a
// --databases dump, and refuses what the swap cannot move.
func allowCreate(toks []token, s string) (stmtKind, string, shape) {
	at := 1
	if wordAt(toks, at, "OR") && wordAt(toks, at+1, "REPLACE") {
		at += 2
	}
	object, objectAt := objectWord(toks)
	switch {
	case object == kindView:
		return stmtRefused, objectsViews, shapeWhole
	case slices.Contains(routineObjects, object):
		return stmtRefused, objectsTriggers, shapeWhole
	case slices.Contains(serverObjects, object):
		return stmtServer, s, shapeWhole
	case object == kindDatabase || object == kindSchema:
		at = objectAt
		if at != 1 {
			// CREATE OR REPLACE DATABASE drops the database it names.
			return stmtUnsupported, s, shapeWhole
		}
		at++
		at += skipWords(toks, at, "IF", "NOT", "EXISTS")
		if name, ok := identAt(toks, at); ok {
			return stmtDatabase, name, shapeWhole
		}
		return stmtUnsupported, s, shapeWhole
	case object == kindSequence:
		at = objectAt + 1
		if !nameAt(toks, at) {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		return stmtObject, "", shapeWhole
	case object == kindTable:
		if objectAt != at {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		at++
		at += skipWords(toks, at, "IF", "NOT", "EXISTS")
		if !nameAt(toks, at) {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		if !punctAt(toks, at+1, "(") {
			return stmtUnsupported, notAllowed(s), shapeTableDef
		}
		return stmtObject, "", shapeTableDef
	}
	return stmtUnsupported, notAllowed(s), shapeWhole
}

func allowDrop(toks []token, s string) (stmtKind, string, shape) {
	object, objectAt := objectWord(toks)
	at := objectAt
	switch {
	case object == kindView:
		return stmtRefused, objectsViews, shapeWhole
	case slices.Contains(routineObjects, object):
		return stmtRefused, objectsTriggers, shapeWhole
	case slices.Contains(serverObjects, object):
		return stmtServer, s, shapeWhole
	case object == kindDatabase || object == kindSchema:
		if at != 1 {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		at++
		at += skipWords(toks, at, "IF", "EXISTS")
		if name, ok := identAt(toks, at); ok {
			return stmtDatabase, name, shapeWhole
		}
		return stmtUnsupported, s, shapeWhole
	case (object == kindTable || object == kindSequence) && at == 1:
		table := object == kindTable
		at++
		at += skipWords(toks, at, "IF", "EXISTS")
		if !nameList(toks, at) {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		if table {
			return stmtObject, "", shapeWhole
		}
		return stmtPass, "", shapeWhole
	}
	return stmtUnsupported, notAllowed(s), shapeWhole
}

// allowAlter takes only the ALTER TABLE ... DISABLE KEYS and ENABLE KEYS that
// a dump wraps around its inserts.
func allowAlter(toks []token, s string) (stmtKind, string, shape) {
	object, _ := objectWord(toks)
	if !wordAt(toks, 1, kindTable) {
		if slices.Contains(serverObjects, object) {
			return stmtServer, s, shapeWhole
		}
		if slices.Contains(routineObjects, object) || object == kindView {
			return stmtRefused, objectsTriggers, shapeWhole
		}
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	at := 2
	if !nameAt(toks, at) {
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	at++
	if len(toks) == at+2 && wordAt(toks, at+1, "KEYS") &&
		(wordAt(toks, at, "DISABLE") || wordAt(toks, at, "ENABLE")) {
		return stmtPass, "", shapeWhole
	}
	return stmtUnsupported, notAllowed(s), shapeWhole
}

func allowInsert(toks []token, s string) (stmtKind, string, shape) {
	at := 1
	at += skipWords(toks, at, "LOW_PRIORITY", "DELAYED", "HIGH_PRIORITY", "IGNORE")
	at += skipWords(toks, at, "INTO")
	if !nameAt(toks, at) {
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	at++
	if punctAt(toks, at, "(") {
		end, ok := columnList(toks, at)
		if !ok {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		at = end
	}
	if !wordAt(toks, at, "VALUES") && !wordAt(toks, at, "VALUE") {
		return stmtUnsupported, notAllowed(s), shapeValues
	}
	for _, t := range toks[at+1:] {
		if !isValueToken(t) {
			return stmtUnsupported, "a value a dump does not write: " + s, shapeValues
		}
	}
	return stmtObject, "", shapeValues
}

func allowLock(toks []token, s string) (stmtKind, string, shape) {
	if !wordAt(toks, 1, "TABLES") {
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	for at := 2; at < len(toks); {
		if !nameAt(toks, at) {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		at++
		switch {
		case wordAt(toks, at, "WRITE"):
			at++
		case wordAt(toks, at, "READ"):
			at++
			at += skipWords(toks, at, "LOCAL")
		default:
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		if at == len(toks) {
			return stmtPass, "", shapeWhole
		}
		if !punctAt(toks, at, ",") {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
		at++
	}
	return stmtUnsupported, notAllowed(s), shapeWhole
}

// allowSequenceCall takes the DO SETVAL and SELECT SETVAL a dump writes to set
// a sequence's next value.
func allowSequenceCall(toks []token, s string) (stmtKind, string, shape) {
	if !wordAt(toks, 1, "SETVAL") || !punctAt(toks, 2, "(") {
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	at := 3
	if !nameAt(toks, at) {
		return stmtUnsupported, notAllowed(s), shapeWhole
	}
	for at++; at < len(toks); at++ {
		t := toks[at]
		switch {
		case t.kind == tokPunct && (t.text == "," || t.text == ")" || t.text == "-"):
		case t.kind == tokWord && isNumberWord(t.text):
		case t.kind == tokWord && (strings.EqualFold(t.text, "TRUE") || strings.EqualFold(t.text, "FALSE")):
		default:
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
	}
	return stmtPass, "", shapeWhole
}

func allowTransaction(verb string, toks []token, s string) (stmtKind, string, shape) {
	switch {
	case verb == verbCommit && len(toks) == 1:
		return stmtPass, "", shapeWhole
	case verb == verbStart && len(toks) == 2 && wordAt(toks, 1, "TRANSACTION"):
		return stmtPass, "", shapeWhole
	case verb == verbStart:
		return stmtServer, s, shapeWhole
	}
	return stmtUnsupported, notAllowed(s), shapeWhole
}

// allowSet takes a SET of the session variables in sessionVars, the saving of
// one in a user variable, and SET NAMES. sqlModes checks the values that
// decide how the server reads the rest of the dump.
func allowSet(toks []token, s string) (stmtKind, string, shape) {
	for _, t := range toks {
		if t.kind == tokWord {
			switch strings.ToUpper(t.text) {
			case "GLOBAL", "PERSIST", "PERSIST_ONLY", "PASSWORD", "ROLE", "STATEMENT":
				return stmtServer, s, shapeWhole
			}
		}
	}
	for _, item := range splitTopLevel(toks[1:]) {
		if !allowSetItem(item) {
			return stmtUnsupported, notAllowed(s), shapeWhole
		}
	}
	return stmtPass, "", shapeWhole
}

func allowSetItem(item []token) bool {
	for len(item) > 0 && item[0].kind == tokWord &&
		(strings.EqualFold(item[0].text, "SESSION") || strings.EqualFold(item[0].text, "LOCAL")) {
		item = item[1:]
	}
	if len(item) == 0 {
		return false
	}
	switch {
	case wordAt(item, 0, "NAMES"), wordAt(item, 0, "CHARSET"):
		return allowCharsetValue(item[1:])
	case wordAt(item, 0, "CHARACTER") && wordAt(item, 1, "SET"):
		return allowCharsetValue(item[2:])
	}
	eq := slices.IndexFunc(item, func(t token) bool { return t.kind == tokPunct && t.text == "=" })
	if eq < 0 {
		return false
	}
	lhs, rhs := item[:eq], item[eq+1:]
	if n := len(lhs); n > 0 && lhs[n-1].kind == tokPunct && lhs[n-1].text == ":" {
		lhs = lhs[:n-1]
	}
	if userVar(lhs) == "" && !isSystemVar(lhs, sessionVars...) {
		return false
	}
	return isSetValue(rhs)
}

// allowCharsetValue takes the shape of a SET NAMES value: a charset, and the
// collation a dump may name after it. Which charsets are safe is sqlModes's
// to say, since it follows what the dump saved.
func allowCharsetValue(rest []token) bool {
	if len(rest) == 0 {
		return false
	}
	switch {
	case rest[0].kind == tokWord, rest[0].kind == tokString, rest[0].kind == tokIdent:
	case userVar(rest) != "":
		return true
	default:
		return false
	}
	rest = rest[1:]
	if n := skipWords(rest, 0, "COLLATE"); n == 1 {
		rest = rest[1:]
		if len(rest) == 1 && (rest[0].kind == tokWord || rest[0].kind == tokString || rest[0].kind == tokIdent) {
			return true
		}
		return false
	}
	return len(rest) == 0
}

// isSetValue reports whether rhs is a value a dump assigns: a literal, a
// system variable, or a user variable it saved one in.
func isSetValue(rhs []token) bool {
	switch {
	case len(rhs) == 0:
		return false
	case userVar(rhs) != "":
		return true
	case len(rhs) > 2 && rhs[0].kind == tokPunct && rhs[0].text == "@" &&
		rhs[1].kind == tokPunct && rhs[1].text == "@":
		return isSystemVar(rhs, sessionVars...)
	}
	for _, t := range rhs {
		switch {
		case t.kind == tokString, t.kind == tokIdent:
		case t.kind == tokWord:
		case t.kind == tokPunct && (t.text == "+" || t.text == "-" || t.text == "." || t.text == ":"):
		default:
			return false
		}
	}
	return len(rhs) <= 3
}

// isValueToken reports whether t can appear in the VALUES list of a dump's
// INSERT: a literal, NULL, DEFAULT, a charset introducer, or the punctuation
// that separates them. An identifier or a function call cannot.
func isValueToken(t token) bool {
	switch t.kind {
	case tokString:
		return true
	case tokPunct:
		return strings.Contains("(),-+.'\"", t.text)
	case tokWord:
		switch strings.ToUpper(t.text) {
		case "NULL", "DEFAULT", "TRUE", "FALSE", "X", "B", "N", "E":
			return true
		}
		return isNumberWord(t.text) || strings.HasPrefix(t.text, "_")
	}
	return false
}

// isNumberWord reports whether word is a number as the tokenizer leaves it: a
// run of digits, or a hex literal such as 0x1F, or a digit run with an
// exponent such as 1e3.
func isNumberWord(word string) bool {
	if word == "" {
		return false
	}
	if !isDigit(word[0]) && word[0] != 'e' && word[0] != 'E' {
		return false
	}
	if len(word) > 2 && word[0] == '0' && (word[1] == 'x' || word[1] == 'X') {
		return strings.IndexFunc(word[2:], func(r rune) bool {
			return !strings.ContainsRune("0123456789abcdefABCDEF", r)
		}) < 0
	}
	if word[0] == 'e' || word[0] == 'E' {
		// The tail of an exponent, such as the e30 of 1.2345e30, which the
		// tokenizer leaves as its own word.
		return len(word) > 1 && strings.IndexFunc(word[1:], func(r rune) bool { return !isDigit(byte(r)) }) < 0
	}
	seenE := false
	for i := 1; i < len(word); i++ {
		switch {
		case isDigit(word[i]):
		case !seenE && (word[i] == 'e' || word[i] == 'E'):
			seenE = true
		default:
			return false
		}
	}
	return true
}

// objectWord returns the kind of object a statement names, such as TABLE or
// VIEW, and where it is. It looks past what may come before it, such as the
// DEFINER of a routine.
func objectWord(toks []token) (string, int) {
	for i, t := range toks {
		if t.kind == tokPunct && t.text == "(" {
			break
		}
		if t.kind != tokWord {
			continue
		}
		word := strings.ToUpper(t.text)
		if word == kindView || word == kindDatabase || word == kindSchema || word == "INDEX" ||
			slices.Contains(routineObjects, word) || slices.Contains(serverObjects, word) ||
			slices.Contains(tableObjects, word) {
			return word, i
		}
	}
	return "", len(toks)
}

func wordAt(toks []token, i int, text string) bool {
	return i >= 0 && i < len(toks) && toks[i].kind == tokWord && strings.EqualFold(toks[i].text, text)
}

func punctAt(toks []token, i int, text string) bool {
	return i >= 0 && i < len(toks) && toks[i].kind == tokPunct && toks[i].text == text
}

// identAt returns the backquoted name at i.
func identAt(toks []token, i int) (string, bool) {
	if i < len(toks) && toks[i].kind == tokIdent {
		return toks[i].text, true
	}
	return "", false
}

// nameAt reports whether toks holds at i the name of a table or sequence,
// backquoted or a plain word, and not qualified with a database.
func nameAt(toks []token, i int) bool {
	if punctAt(toks, i+1, ".") {
		return false
	}
	if _, ok := identAt(toks, i); ok {
		return true
	}
	return i < len(toks) && toks[i].kind == tokWord && !isDigit(toks[i].text[0])
}

// nameList reports whether toks from at is a comma-separated list of names and
// nothing else.
func nameList(toks []token, at int) bool {
	for {
		if !nameAt(toks, at) {
			return false
		}
		at++
		switch {
		case at == len(toks):
			return true
		case punctAt(toks, at, ","):
			at++
		default:
			return false
		}
	}
}

// columnList reads the parenthesized column list of an INSERT and returns the
// index after it.
func columnList(toks []token, at int) (int, bool) {
	at++
	for {
		if !nameAt(toks, at) {
			return at, false
		}
		at++
		switch {
		case punctAt(toks, at, ","):
			at++
		case punctAt(toks, at, ")"):
			return at + 1, true
		default:
			return at, false
		}
	}
}

func skipWords(toks []token, at int, words ...string) int {
	n := 0
	for _, w := range words {
		if wordAt(toks, at+n, w) {
			n++
		}
	}
	return n
}
