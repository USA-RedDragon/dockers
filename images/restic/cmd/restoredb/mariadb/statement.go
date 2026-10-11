package mariadb

import (
	"strconv"
	"strings"
)

type stmtKind int

const (
	stmtPass stmtKind = iota
	// stmtDatabase is a USE, CREATE DATABASE or DROP DATABASE, which a
	// --databases or --all-databases dump writes and which must not reach the
	// server. Its argument is the database named.
	stmtDatabase
	// stmtObject creates, drops or fills a table or sequence.
	stmtObject
	// stmtRefused starts an object the swap cannot move.
	stmtRefused
	// stmtUnsupported is a statement the restore cannot keep out of the live
	// database, or cannot check.
	stmtUnsupported
	// stmtServer changes the server, replication or accounts.
	stmtServer
	// stmtOtherDatabase writes to a table it names with a database.
	stmtOtherDatabase
)

const (
	verbCreate      = "CREATE"
	verbDrop        = "DROP"
	verbAlter       = "ALTER"
	kindSequence    = "SEQUENCE"
	kindTable       = "TABLE"
	objectsTriggers = "triggers, stored routines or events"
	objectsViews    = "views"
	tooLong         = "a statement too long to check: "
	// neverVersion is the version from which an executable comment is taken
	// as one no server runs, such as mariadb-dump's /*M!999999\- ... */.
	neverVersion = 900000
)

var (
	serverVerbs = []string{
		"BACKUP", "BINLOG", "CALL", "CHANGE", "DEALLOCATE", "EXECUTE", "FLUSH", "GRANT", "HANDLER", "INSTALL",
		"KILL", "LOAD", "PREPARE", "PURGE", "RESET", "REVOKE", "SHUTDOWN", "STOP", "UNINSTALL", "XA",
	}
	serverObjects  = []string{"USER", "ROLE", "SERVER", "TABLESPACE", "LOGFILE", "INSTANCE"}
	routineObjects = []string{"TRIGGER", "PROCEDURE", "FUNCTION", "EVENT", "PACKAGE"}
	tableObjects   = []string{kindTable, "TABLES", kindSequence}
	// lexingModes change how the client and server read quotes and
	// backslashes, which the restore's lexer does not follow.
	lexingModes = []string{"ANSI", "ANSI_QUOTES", "NO_BACKSLASH_ESCAPES", "ORACLE", "POSTGRESQL", "DB2",
		"MAXDB", "MSSQL"}
)

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

type tokenKind int

const (
	tokWord tokenKind = iota
	tokIdent
	tokString
	tokPunct
)

type token struct {
	kind tokenKind
	text string
}

// tokenize splits the start of a statement into words, backquoted
// identifiers, quoted strings and punctuation. The opening and closing of an
// executable comment are left out, since the server reads what is inside it.
func tokenize(s string) []token {
	var toks []token
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case isClientSpace(c):
			i++
		case strings.HasPrefix(s[i:], "/*!") || strings.HasPrefix(s[i:], "/*M!"):
			rest := skipVersionComment(s[i:])
			i = len(s) - len(rest)
		case strings.HasPrefix(s[i:], "*/"):
			i += 2
		case c == '`':
			name, n := readQuoted(s[i:], '`')
			toks = append(toks, token{tokIdent, name})
			i += n
		case c == '\'' || c == '"':
			text, n := readQuoted(s[i:], c)
			toks = append(toks, token{tokString, text})
			i += n
		case isWordByte(c):
			j := i
			for j < len(s) && isWordByte(s[j]) {
				j++
			}
			toks = append(toks, token{tokWord, s[i:j]})
			i = j
		default:
			toks = append(toks, token{tokPunct, s[i : i+1]})
			i++
		}
	}
	return toks
}

func isWordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || isDigit(c) || c == '_' || c == '$' || c >= 0x80
}

// readQuoted reads the quoted text at the start of s, whose first byte is
// quote, and returns it unescaped and how many bytes it took.
func readQuoted(s string, quote byte) (string, int) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch {
		case s[i] == '\\' && quote != '`' && i+1 < len(s):
			i++
			b.WriteByte(s[i])
		case s[i] == quote && i+1 < len(s) && s[i+1] == quote:
			b.WriteByte(quote)
			i++
		case s[i] == quote:
			return b.String(), i + 1
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String(), len(s)
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// skipLeadingComments strips the executable comment openings, such as
// "/*!40000 ", and closings at the start of s, and skips whole any comment
// for a version no server has.
func skipLeadingComments(s string) string {
	for {
		s = strings.TrimLeft(s, " \t\r\n")
		switch {
		case strings.HasPrefix(s, "*/"):
			s = s[2:]
		case strings.HasPrefix(s, "/*!") || strings.HasPrefix(s, "/*M!"):
			rest := skipVersionComment(s)
			if commentVersion(s) >= neverVersion {
				if _, after, ok := strings.Cut(rest, "*/"); ok {
					rest = after
				}
			}
			s = rest
		default:
			return s
		}
	}
}

func commentVersion(s string) int {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "/*!"), "/*M!")
	end := 0
	for end < len(s) && isDigit(s[end]) {
		end++
	}
	v, _ := strconv.Atoi(s[:end])
	return v
}

// skipVersionComment strips the opening of a versioned comment, such as
// "/*!40000 " or "/*M!100616 ", from the start of s.
func skipVersionComment(s string) string {
	for _, open := range []string{"/*!", "/*M!"} {
		if rest, ok := strings.CutPrefix(s, open); ok {
			return strings.TrimLeft(strings.TrimLeft(rest, "0123456789"), " ")
		}
	}
	return s
}
