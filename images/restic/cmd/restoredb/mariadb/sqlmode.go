package mariadb

import (
	"fmt"
	"slices"
	"strings"
)

// sqlModes follows the user variables that hold the sql_mode or the client
// charset a dump saved at its start, such as mariadb-dump's
//
//	SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO'
//	SET @saved_cs_client = @@character_set_client
//
// Both decide how the server reads the rest of the dump, so a SET may change
// either only to a literal the filter can check, to DEFAULT, or back to a
// variable saved from that same variable and not changed since. Any other
// statement that uses a user variable could have changed one, so it forgets
// them all.
type sqlModes struct {
	// saved maps a user variable to the system variable it holds.
	saved map[string]string
}

func (m *sqlModes) forget() {
	clear(m.saved)
}

// set follows the assignments of a SET statement, toks without the SET.
func (m *sqlModes) set(toks []token, stmt string) error {
	if m.saved == nil {
		m.saved = map[string]string{}
	}
	for _, item := range splitTopLevel(toks) {
		for len(item) > 0 && item[0].kind == tokWord &&
			(strings.EqualFold(item[0].text, "SESSION") || strings.EqualFold(item[0].text, "LOCAL")) {
			item = item[1:]
		}
		if done, err := m.names(item, stmt); done {
			if err != nil {
				return err
			}
			continue
		}
		eq := slices.IndexFunc(item, func(t token) bool { return t.kind == tokPunct && t.text == "=" })
		if eq < 0 {
			return fmt.Errorf("%w: a SET item restoredb does not know how to check: %.200s",
				ErrUnsupportedStatement, stmt)
		}
		lhs, rhs := item[:eq], item[eq+1:]
		if n := len(lhs); n > 0 && lhs[n-1].kind == tokPunct && lhs[n-1].text == ":" {
			lhs = lhs[:n-1]
		}
		switch {
		case isSystemVar(lhs, varCharsetClient):
			if !m.knownCharset(rhs) {
				return fmt.Errorf("%w: %s", ErrUnsupportedStatement, charsetRefusal(stmt))
			}
			continue
		case isSystemVar(lhs, engineVars...):
			if !safeEngine(rhs) {
				return fmt.Errorf("%w: a default engine the restore cannot check, so a CREATE TABLE without an "+
					"ENGINE of its own could reach tables outside the staging database: %.200s",
					ErrUnsupportedStatement, stmt)
			}
			continue
		case isSQLMode(lhs):
			if !m.knownMode(rhs) {
				return fmt.Errorf("%w: an sql_mode set to something other than a literal the restore can read "+
					"or the mode the dump saved at its start: %.200s", ErrUnsupportedStatement, stmt)
			}
			continue
		case userVar(lhs) != "":
			name := userVar(lhs)
			m.taint(rhs)
			delete(m.saved, name)
			if len(rhs) > 2 && rhs[0].text == "@" && rhs[1].text == "@" {
				for _, sys := range []string{varSQLMode, varCharsetClient} {
					if isSystemVar(rhs, sys) {
						m.saved[name] = sys
					}
				}
			}
			continue
		}
		m.taint(rhs)
	}
	return nil
}

// names checks an item of a SET that is NAMES, CHARACTER SET or CHARSET, which
// all set the charset the server reads the dump in. MariaDB takes them
// anywhere in the list. done reports whether item is one of them.
func (m *sqlModes) names(item []token, stmt string) (done bool, err error) {
	word := func(i int, text string) bool {
		return len(item) > i && item[i].kind == tokWord && strings.EqualFold(item[i].text, text)
	}
	at := 0
	switch {
	case word(0, "NAMES"), word(0, "CHARSET"):
		at = 1
	case word(0, "CHARACTER") && word(1, "SET"):
		at = 2
	default:
		return false, nil
	}
	if !m.knownCharset(item[at:min(at+1, len(item))]) {
		return true, fmt.Errorf("%w: %s", ErrUnsupportedStatement, charsetRefusal(stmt))
	}
	return true, nil
}

func charsetRefusal(stmt string) string {
	return fmt.Sprintf("a client charset other than %s or the one the dump saved at its start, which would make "+
		"the server read the dump differently from restoredb: %.200s",
		strings.Join(safeCharsets, ", "), stmt)
}

// safeCharsets are the charsets in which no character ends in 0x5c, the
// backslash, so the server ends a quoted string exactly where the restore's
// lexer does. gbk, big5, sjis, cp932, gb18030 and euckr do not qualify.
var safeCharsets = []string{"utf8mb4", "utf8mb3", "utf8", "latin1", "ascii", "binary"}

// knownCharset reports whether rhs sets a charset the filter can follow.
// DEFAULT is not one: it is the server's configured charset, which the filter
// cannot see.
func (m *sqlModes) knownCharset(rhs []token) bool {
	if len(rhs) == 1 {
		switch rhs[0].kind {
		case tokWord, tokString, tokIdent:
			return slices.ContainsFunc(safeCharsets, func(cs string) bool { return strings.EqualFold(rhs[0].text, cs) })
		}
	}
	return m.savedFrom(rhs, varCharsetClient)
}

// savedFrom reports whether rhs is a user variable saved from the system
// variable sys and not changed since.
func (m *sqlModes) savedFrom(rhs []token, sys string) bool {
	name := userVar(rhs)
	return name != "" && m.saved[name] == sys
}

// knownMode reports whether rhs sets a mode the filter can follow.
func (m *sqlModes) knownMode(rhs []token) bool {
	switch {
	case len(rhs) == 1 && rhs[0].kind == tokString:
		for mode := range strings.SplitSeq(strings.ToUpper(rhs[0].text), ",") {
			if slices.Contains(lexingModes, strings.TrimSpace(mode)) {
				return false
			}
		}
		return true
	case len(rhs) == 1 && rhs[0].kind == tokWord:
		return strings.EqualFold(rhs[0].text, "DEFAULT")
	}
	return m.savedFrom(rhs, varSQLMode)
}

// taint forgets every user variable rhs names, since := can assign it.
func (m *sqlModes) taint(rhs []token) {
	for i := 0; i+1 < len(rhs); i++ {
		if rhs[i].text != "@" || rhs[i].kind != tokPunct {
			continue
		}
		if rhs[i+1].kind == tokPunct && rhs[i+1].text == "@" {
			i++
			continue
		}
		delete(m.saved, strings.ToLower(rhs[i+1].text))
	}
}

// userVar returns the name of the user variable toks is, such as @x, or "".
func userVar(toks []token) string {
	if len(toks) != 2 || toks[0].kind != tokPunct || toks[0].text != "@" || toks[1].kind == tokPunct {
		return ""
	}
	return strings.ToLower(toks[1].text)
}

const (
	varSQLMode       = "SQL_MODE"
	varCharsetClient = "CHARACTER_SET_CLIENT"
)

// engineVars are the variables that decide the engine of a table created
// without an ENGINE of its own.
var engineVars = []string{"DEFAULT_STORAGE_ENGINE", "DEFAULT_TMP_STORAGE_ENGINE", "STORAGE_ENGINE",
	"ENFORCE_STORAGE_ENGINE"}

// safeEngines are the engines a dump of tables needs. The others are refused
// as a default, since reachingEngines reads and writes other tables and an
// engine the restore does not know could too.
var safeEngines = []string{"INNODB", "ARIA", "MYISAM", "MEMORY", "HEAP", "CSV", "ARCHIVE", "SEQUENCE", "BLACKHOLE"}

// safeEngine reports whether rhs is one plain name of a safe engine.
func safeEngine(rhs []token) bool {
	return len(rhs) == 1 && (rhs[0].kind == tokString || rhs[0].kind == tokWord || rhs[0].kind == tokIdent) &&
		slices.Contains(safeEngines, strings.ToUpper(rhs[0].text))
}

// isSQLMode reports whether toks name the session's sql_mode, as sql_mode,
// @@sql_mode or @@SESSION.sql_mode.
func isSQLMode(toks []token) bool {
	return isSystemVar(toks, varSQLMode)
}

// isSystemVar reports whether toks name one of the given system variables, in
// any of the forms x, @@x and @@SESSION.x.
func isSystemVar(toks []token, names ...string) bool {
	var words []string
	for _, t := range toks {
		switch {
		case t.kind == tokPunct && (t.text == "@" || t.text == "."):
		case t.kind == tokWord && (strings.EqualFold(t.text, "SESSION") || strings.EqualFold(t.text, "LOCAL")):
		default:
			words = append(words, strings.ToUpper(t.text))
		}
	}
	return len(words) == 1 && slices.Contains(names, words[0])
}

// splitTopLevel splits toks at the commas outside parentheses.
func splitTopLevel(toks []token) [][]token {
	var items [][]token
	depth, start := 0, 0
	for i, t := range toks {
		if t.kind != tokPunct {
			continue
		}
		switch t.text {
		case "(":
			depth++
		case ")":
			depth--
		case ",":
			if depth == 0 {
				items = append(items, toks[start:i])
				start = i + 1
			}
		}
	}
	return append(items, toks[start:])
}
