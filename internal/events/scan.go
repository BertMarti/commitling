package events

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// A small JSON reader for the few fields commitling needs. encoding/json
// (with reflect behind it) weighed about a megabyte in the WebAssembly build,
// and the shape of a public event is fixed and tiny, so the fields are read
// by hand. It follows encoding/json where it matters here: strict syntax,
// null leaves a field untouched, keys match without regard to case (Unicode simple folding, so "ſize" is "size", like encoding/json), the
// last duplicate key wins and unknown fields are skipped. The differential
// test in scan_test.go keeps it honest against encoding/json.

// maxDepth limits the nesting (encoding/json allows 10000, but a deep
// recursion overflows the stack of a browser, and public events nest 3 levels
// at most apart from the commit payloads).
const maxDepth = 500

type scanner struct {
	b     []byte
	i     int
	depth int
}

func (s *scanner) fail(msg string) error {
	return errors.New("JSON no válido en el byte " + strconv.Itoa(s.i) + ": " + msg)
}

func (s *scanner) ws() {
	for s.i < len(s.b) {
		switch s.b[s.i] {
		case ' ', '\t', '\n', '\r':
			s.i++
		default:
			return
		}
	}
}

// peek returns the next significant byte, or 0 at the end of the input.
func (s *scanner) peek() byte {
	s.ws()
	if s.i < len(s.b) {
		return s.b[s.i]
	}
	return 0
}

func (s *scanner) expect(c byte) error {
	if s.peek() != c {
		return s.fail("se esperaba '" + string(c) + "'")
	}
	s.i++
	return nil
}

// null consumes a null literal if it is next.
func (s *scanner) null() bool {
	if s.peek() == 'n' && s.literal("null") == nil {
		return true
	}
	return false
}

func (s *scanner) literal(word string) error {
	for j := 0; j < len(word); j++ {
		if s.i+j >= len(s.b) || s.b[s.i+j] != word[j] {
			return s.fail("valor inesperado")
		}
	}
	s.i += len(word)
	return nil
}

// str reads a string. With keep false it only validates.
func (s *scanner) str(keep bool) (string, error) {
	if err := s.expect('"'); err != nil {
		return "", err
	}
	start, escaped := s.i, false
	for ; s.i < len(s.b); s.i++ {
		c := s.b[s.i]
		switch {
		case c == '"':
			raw := s.b[start:s.i]
			s.i++
			if !keep {
				return "", nil
			}
			if escaped {
				return unescape(raw), nil
			}
			return clean(string(raw)), nil
		case c < 0x20:
			return "", s.fail("carácter de control dentro de una cadena")
		case c == '\\':
			escaped = true
			s.i++
			if s.i >= len(s.b) {
				return "", s.fail("cadena sin cerrar")
			}
			switch s.b[s.i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				if s.i+4 >= len(s.b) || !isHex(s.b[s.i+1:s.i+5]) {
					return "", s.fail("escape \\u no válido")
				}
				s.i += 4
			default:
				return "", s.fail("escape no válido")
			}
		}
	}
	return "", s.fail("cadena sin cerrar")
}

func isHex(b []byte) bool {
	for _, c := range b {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
			return false
		}
	}
	return true
}

// clean replaces every invalid UTF-8 byte by U+FFFD, as encoding/json does.
func clean(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r) // RuneError for an invalid byte
		i += n
	}
	return b.String()
}

// unescape decodes the body of a string already validated by str.
func unescape(raw []byte) string {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		switch raw[i] {
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			r := hex4(raw[i+1 : i+5])
			i += 4
			if utf16.IsSurrogate(r) {
				if i+6 < len(raw) && raw[i+1] == '\\' && raw[i+2] == 'u' && isHex(raw[i+3:i+7]) {
					if d := utf16.DecodeRune(r, hex4(raw[i+3:i+7])); d != utf8.RuneError {
						r = d
						i += 6
					} else {
						r = utf8.RuneError
					}
				} else {
					r = utf8.RuneError
				}
			}
			b.WriteRune(r)
		default: // " \ /
			b.WriteByte(raw[i])
		}
	}
	return clean(b.String())
}

func hex4(b []byte) rune {
	var r rune
	for _, c := range b {
		switch {
		case c <= '9':
			c -= '0'
		case c <= 'F':
			c = c - 'A' + 10
		default:
			c = c - 'a' + 10
		}
		r = r<<4 | rune(c)
	}
	return r
}

// number returns the text of a number, validated against the JSON grammar.
func (s *scanner) number() (string, error) {
	start := s.i
	if s.i < len(s.b) && s.b[s.i] == '-' {
		s.i++
	}
	digits := func() int {
		n := 0
		for s.i < len(s.b) && '0' <= s.b[s.i] && s.b[s.i] <= '9' {
			s.i++
			n++
		}
		return n
	}
	switch {
	case s.i < len(s.b) && s.b[s.i] == '0':
		s.i++
	case digits() == 0:
		return "", s.fail("valor inesperado")
	}
	if s.i < len(s.b) && s.b[s.i] == '.' {
		s.i++
		if digits() == 0 {
			return "", s.fail("número no válido")
		}
	}
	if s.i < len(s.b) && (s.b[s.i] == 'e' || s.b[s.i] == 'E') {
		s.i++
		if s.i < len(s.b) && (s.b[s.i] == '+' || s.b[s.i] == '-') {
			s.i++
		}
		if digits() == 0 {
			return "", s.fail("número no válido")
		}
	}
	return string(s.b[start:s.i]), nil
}

// object walks the members of an object; member must consume the value.
func (s *scanner) object(member func(key string) error) error {
	if err := s.expect('{'); err != nil {
		return err
	}
	if s.depth++; s.depth > maxDepth {
		return s.fail("demasiado anidado")
	}
	defer func() { s.depth-- }()
	if s.peek() == '}' {
		s.i++
		return nil
	}
	for {
		if s.peek() != '"' {
			return s.fail("se esperaba el nombre de un campo")
		}
		key, err := s.str(true)
		if err != nil {
			return err
		}
		if err := s.expect(':'); err != nil {
			return err
		}
		if err := member(key); err != nil {
			return err
		}
		switch s.peek() {
		case ',':
			s.i++
		case '}':
			s.i++
			return nil
		default:
			return s.fail("se esperaba ',' o '}'")
		}
	}
}

// array walks the elements of an array; element must consume the value.
func (s *scanner) array(element func() error) error {
	if err := s.expect('['); err != nil {
		return err
	}
	if s.depth++; s.depth > maxDepth {
		return s.fail("demasiado anidado")
	}
	defer func() { s.depth-- }()
	if s.peek() == ']' {
		s.i++
		return nil
	}
	for {
		if err := element(); err != nil {
			return err
		}
		switch s.peek() {
		case ',':
			s.i++
		case ']':
			s.i++
			return nil
		default:
			return s.fail("se esperaba ',' o ']'")
		}
	}
}

// skip consumes any value, validating it.
func (s *scanner) skip() error {
	switch c := s.peek(); {
	case c == '{':
		return s.object(func(string) error { return s.skip() })
	case c == '[':
		return s.array(s.skip)
	case c == '"':
		_, err := s.str(false)
		return err
	case c == 't':
		return s.literal("true")
	case c == 'f':
		return s.literal("false")
	case c == 'n':
		return s.literal("null")
	default:
		_, err := s.number()
		return err
	}
}

// raw consumes any value and returns its bytes.
func (s *scanner) raw() ([]byte, error) {
	s.ws()
	start := s.i
	if err := s.skip(); err != nil {
		return nil, err
	}
	return s.b[start:s.i], nil
}

// text reads a string field; null leaves it as it is.
func (s *scanner) text(dst *string) error {
	if s.null() {
		return nil
	}
	if s.peek() != '"' {
		return s.fail("se esperaba una cadena")
	}
	v, err := s.str(true)
	if err == nil {
		*dst = v
	}
	return err
}

// nested reads an object whose only interesting member is one string.
func (s *scanner) nested(name string, dst *string) error {
	if s.null() {
		return nil
	}
	if s.peek() != '{' {
		return s.fail("se esperaba un objeto")
	}
	return s.object(func(key string) error {
		if strings.EqualFold(key, name) {
			return s.text(dst)
		}
		return s.skip()
	})
}

func (s *scanner) event(e *Event) error {
	if s.null() {
		return nil
	}
	if s.peek() != '{' {
		return s.fail("se esperaba un evento")
	}
	return s.object(func(key string) error {
		switch {
		case strings.EqualFold(key, "id"):
			return s.text(&e.ID)
		case strings.EqualFold(key, "type"):
			return s.text(&e.Type)
		case strings.EqualFold(key, "created_at"):
			if s.null() {
				return nil
			}
			raw, err := s.raw()
			if err != nil {
				return err
			}
			return e.CreatedAt.UnmarshalJSON(raw)
		case strings.EqualFold(key, "actor"):
			return s.nested("login", &e.Actor.Login)
		case strings.EqualFold(key, "repo"):
			return s.nested("name", &e.Repo.Name)
		case strings.EqualFold(key, "payload"):
			raw, err := s.raw()
			e.Payload = raw
			return err
		}
		return s.skip()
	})
}

// parseEvents reads a JSON array of events. Like encoding/json it does not
// look at what follows the array.
func parseEvents(data []byte) ([]Event, error) {
	s := &scanner{b: data}
	if s.peek() == 0 {
		return nil, s.fail("la entrada está vacía")
	}
	if s.null() {
		return nil, nil
	}
	var events []Event
	err := s.array(func() error {
		events = append(events, Event{})
		return s.event(&events[len(events)-1])
	})
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []Event{}
	}
	return events, nil
}

// payloadObject returns a scanner on the payload when it is a valid JSON object.
func payloadObject(payload []byte) (*scanner, bool) {
	s := &scanner{b: payload}
	if s.skip() != nil || s.peek() != 0 { // not JSON, or something after the value
		return nil, false
	}
	s = &scanner{b: payload}
	return s, s.peek() == '{'
}

// pushInfo reads size and the number of commits of a push payload. Anything
// that does not fit (not JSON, not an object, wrong types) counts as absent,
// like the lenient decoding it replaces.
func pushInfo(payload []byte) (size int, hasSize bool, commits int) {
	s, ok := payloadObject(payload)
	if !ok {
		return
	}
	_ = s.object(func(key string) error {
		switch {
		case strings.EqualFold(key, "size"):
			if s.null() { // null resets a pointer, as in encoding/json
				size, hasSize = 0, false
				return nil
			}
			if c := s.peek(); c == '-' || '0' <= c && c <= '9' {
				num, _ := s.number()
				if n, err := strconv.Atoi(num); err == nil {
					size, hasSize = n, true
				}
				return nil
			}
		case strings.EqualFold(key, "commits"):
			if s.null() {
				commits = 0
				return nil
			}
			if s.peek() == '[' {
				commits = 0
				return s.array(func() error { commits++; return s.skip() })
			}
		}
		return s.skip()
	})
	return
}

// actionOf reads the "action" string of a payload ("" when absent).
func actionOf(payload []byte) (action string) {
	s, ok := payloadObject(payload)
	if !ok {
		return
	}
	_ = s.object(func(key string) error {
		if strings.EqualFold(key, "action") && s.peek() == '"' {
			v, err := s.str(true)
			action = v
			return err
		}
		return s.skip()
	})
	return
}
