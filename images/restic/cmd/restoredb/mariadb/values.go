package mariadb

// valueWatch checks the tokens of an INSERT's value list as they stream past,
// so a value list longer than the head is checked to its end. It sees the
// bytes outside strings; what is inside a string is the value itself.
type valueWatch struct {
	on   bool
	word []byte
	bad  string
}

func (v *valueWatch) reset(on bool) {
	*v = valueWatch{on: on, word: v.word[:0]}
}

// text sees the statement bytes p, whose quote state at emission is inString.
func (v *valueWatch) text(p []byte, inString byte) {
	if !v.on || v.bad != "" {
		return
	}
	for _, c := range p {
		v.byte(c, inString)
	}
}

func (v *valueWatch) byte(c, inString byte) {
	if inString != 0 {
		v.end()
		return
	}
	if isWordByte(c) {
		if len(v.word) < maxNameWord {
			v.word = append(v.word, c)
		}
		return
	}
	v.end()
	if isClientSpace(c) {
		return
	}
	if !isValueToken(token{kind: tokPunct, text: string(c)}) {
		v.refuse(string(c))
	}
}

// end finishes the word being read and checks it.
func (v *valueWatch) end() {
	if len(v.word) == 0 {
		return
	}
	word := string(v.word)
	v.word = v.word[:0]
	if !isValueToken(token{kind: tokWord, text: word}) {
		v.refuse(word)
	}
}

func (v *valueWatch) refuse(what string) {
	if v.bad == "" {
		v.bad = what
	}
}
