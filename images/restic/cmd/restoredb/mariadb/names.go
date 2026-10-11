package mariadb

import (
	"slices"
	"strings"
)

type nameToken int

const (
	// nameOther is any token but a name: a number, a string, a variable, or
	// punctuation.
	nameOther nameToken = iota
	// nameWord is an unquoted word that can name something.
	nameWord
	// nameQuoted is a backquoted name.
	nameQuoted
	// nameDot is a dot after a name.
	nameDot
	// nameNumber is a word of digits so far. MariaDB reads one that goes on
	// with a letter, such as 1db, as a name.
	nameNumber
)

// qualifiedNames watches the whole text of a statement, as the client sends
// it, for a name qualified with a database, such as `other`.`t` or other.f(),
// and for the word SELECT. The statement is seen a byte at a time with the
// lexer's quote state, so a name in the part of a long statement passed on
// before the end is seen too.
type qualifiedNames struct {
	last nameToken
	// word holds the start of the current unquoted word, or of the last
	// name.
	word      []byte
	inWord    bool
	at        bool
	prevName  string
	qualifier string
	sawSelect bool
	// ats counts the @ just before the current byte; one alone starts a user
	// variable.
	ats        int
	sawUserVar bool
	// afterEngine is set after the word ENGINE, until its value.
	afterEngine bool
	// reach names an engine or table option that reaches tables outside the
	// staging database, such as MERGE's UNION or FEDERATED's CONNECTION, or
	// places a table's files outside the data directory, DATA or INDEX
	// DIRECTORY.
	reach string
}

// reachingEngines read and write tables other than their own, in another
// database or on another server.
var reachingEngines = []string{"MERGE", "MRG_MYISAM", "MRG_ISAM", "FEDERATED", "FEDERATEDX", "CONNECT", "SPIDER",
	"OQGRAPH"}

const maxNameWord = 64

func (q *qualifiedNames) reset() {
	*q = qualifiedNames{word: q.word[:0]}
}

func isNameStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '$' || c >= 0x80
}

// text sees the statement bytes p, whose quote state at emission is inString.
func (q *qualifiedNames) text(p []byte, inString byte) {
	for _, c := range p {
		q.byte(c, inString)
	}
}

func (q *qualifiedNames) byte(c, inString byte) {
	if c == '@' && inString == 0 {
		q.ats++
	} else {
		if q.ats == 1 {
			q.sawUserVar = true
		}
		q.ats = 0
	}
	if q.afterEngine && inString != 0 {
		q.afterEngine = false
		q.see("a quoted ENGINE")
	}
	if inString != 0 {
		q.quoted(c, inString)
		return
	}
	if q.inWord {
		if isNameStart(c) || isDigit(c) {
			if q.last == nameNumber && isNameStart(c) {
				q.last = nameWord
			}
			if len(q.word) < maxNameWord {
				q.word = append(q.word, c)
			}
			return
		}
		q.endWord()
	}
	switch {
	case c == '`':
	case isNameStart(c):
		if q.at {
			q.at = false
			q.inWord, q.last = true, nameOther
			q.word = q.word[:0]
			return
		}
		q.startName(nameWord)
		q.word = append(q.word, c)
		q.inWord = true
	case isDigit(c) && (q.at || q.last != nameDot):
		q.last = nameNumber
		if q.at {
			q.last = nameOther
		}
		q.at = false
		q.word = append(q.word[:0], c)
		q.inWord = true
	case isDigit(c):
		q.startName(nameWord)
		q.word = append(q.word, c)
		q.inWord = true
	case isClientSpace(c), c == '=' && q.afterEngine:
	case c == '.' && (q.last == nameWord || q.last == nameQuoted):
		q.prevName = string(q.word)
		q.last = nameDot
	case c == '@':
		q.at = true
		q.last = nameOther
	default:
		q.at = false
		q.last = nameOther
	}
	if !isClientSpace(c) && c != '=' && !q.inWord {
		q.afterEngine = false
	}
}

// quoted sees a byte of a quoted string or a backquoted name.
func (q *qualifiedNames) quoted(c, inString byte) {
	if inString != '`' {
		q.endWord()
		q.last = nameOther
		return
	}
	if q.last != nameQuoted {
		q.startName(nameQuoted)
		if c == '`' {
			return
		}
	}
	if len(q.word) < maxNameWord {
		q.word = append(q.word, c)
	}
}

func (q *qualifiedNames) see(reach string) {
	if q.reach == "" {
		q.reach = reach
	}
}

// startName begins a name token; one right after a dot after a name makes the
// name before the dot a qualifier.
func (q *qualifiedNames) startName(kind nameToken) {
	if q.last == nameDot && q.qualifier == "" {
		q.qualifier = q.prevName
	}
	q.last = kind
	q.word = q.word[:0]
}

func (q *qualifiedNames) endWord() {
	if !q.inWord {
		return
	}
	q.inWord = false
	if q.last != nameWord {
		return
	}
	word := strings.ToUpper(string(q.word))
	if q.afterEngine {
		q.afterEngine = false
		if slices.Contains(reachingEngines, word) {
			q.see("ENGINE=" + word)
		}
	}
	switch word {
	case verbSelect:
		q.sawSelect = true
	case "ENGINE":
		q.afterEngine = true
	case "UNION", "CONNECTION", "DIRECTORY":
		q.see(word)
	}
}
