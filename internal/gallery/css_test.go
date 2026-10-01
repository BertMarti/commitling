package gallery

import (
	"regexp"
	"strings"
	"testing"
)

// A tiny reader of the page's own CSS, so the tests can ask "what does this
// selector say about this property?" instead of looking for a literal piece
// of text (which breaks when a property is reordered and says nothing about
// the intent). It understands plain rules, selector lists, @media and
// @supports blocks (nested too) and ignores comments, strings, statement
// at-rules (@charset, @import) and blocks of declarations such as @keyframes
// and @font-face; that is all the page uses.

type cssRule struct {
	cond  string // the enclosing conditions: "(max-width:720px)" for @media, "supports (display:grid)" for @supports, joined by " && "; "" outside any
	sel   string // one selector of the list
	props map[string]string
}

type ruleSet []cssRule

var (
	spaces     = regexp.MustCompile(`\s+`)
	aroundPunc = regexp.MustCompile(`\s*([,():])\s*`)
)

// normalise collapses blanks so "minmax(0, 1fr)" and "minmax(0,1fr)" are the same.
func normalise(v string) string {
	return strings.TrimSpace(aroundPunc.ReplaceAllString(spaces.ReplaceAllString(v, " "), "$1"))
}

// skipQuoted returns the index after the string that starts at s[i] (a quote).
func skipQuoted(s string, i int) int {
	q := s[i]
	for i++; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case q:
			return i + 1
		}
	}
	return len(s)
}

// stripComments removes /* */ comments (an unterminated one runs to the end)
// and leaves strings alone.
func stripComments(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case s[i] == '"' || s[i] == '\'':
			j := skipQuoted(s, i)
			b.WriteString(s[i:j])
			i = j
		case strings.HasPrefix(s[i:], "/*"):
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			i += 2 + end + 2
			b.WriteByte(' ') // a comment separates tokens
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// indexTop is the index of the first of chars outside strings and
// parentheses, or -1.
func indexTop(s, chars string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"' || c == '\'':
			i = skipQuoted(s, i) - 1
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case depth <= 0 && strings.IndexByte(chars, c) >= 0:
			return i
		}
	}
	return -1
}

// splitTop splits s on sep where it is not inside strings or parentheses
// (a comma in :is(a, b) or a ; in url(data:...;base64,...) is not a separator).
func splitTop(s string, sep byte) []string {
	var parts []string
	for {
		i := indexTop(s, string(sep))
		if i < 0 {
			return append(parts, s)
		}
		parts = append(parts, s[:i])
		s = s[i+1:]
	}
}

// blockEnd returns the index after the } that closes the { at s[open].
func blockEnd(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '"', '\'':
			i = skipQuoted(s, i) - 1
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return i + 1
			}
		}
	}
	return len(s)
}

func joinCond(parent, label string) string {
	if parent == "" {
		return normalise(label)
	}
	return normalise(parent + " && " + label)
}

func cssRules(css string) ruleSet {
	var out ruleSet
	var walk func(src, cond string)
	walk = func(src, cond string) {
		for {
			src = strings.TrimSpace(src)
			if src == "" {
				return
			}
			end := indexTop(src, ";{")
			if end < 0 {
				return
			}
			if src[end] == ';' { // @charset, @import: a statement, not a block
				src = src[end+1:]
				continue
			}
			head := strings.TrimSpace(src[:end])
			stop := blockEnd(src, end)
			body := src[end+1 : max(end+1, stop-1)]
			src = src[stop:]
			switch {
			case strings.HasPrefix(head, "@media"):
				walk(body, joinCond(cond, strings.TrimPrefix(head, "@media")))
			case strings.HasPrefix(head, "@supports"):
				walk(body, joinCond(cond, "supports "+strings.TrimPrefix(head, "@supports")))
			case strings.HasPrefix(head, "@"):
				// @keyframes, @font-face...: not rules about the layout
			default:
				props := map[string]string{}
				for _, decl := range splitTop(body, ';') {
					if name, val, ok := strings.Cut(decl, ":"); ok {
						props[strings.TrimSpace(name)] = normalise(val)
					}
				}
				for _, sel := range splitTop(head, ',') {
					out = append(out, cssRule{cond: cond, sel: normalise(sel), props: props})
				}
			}
		}
	}
	walk(stripComments(css), "")
	return out
}

// prop is the value of property for sel outside any @media (the last rule wins).
func (rs ruleSet) prop(sel, property string) string { return rs.lookup("", sel, property) }

// hasMedia reports whether the CSS has any rule inside that @media condition.
func (rs ruleSet) hasMedia(media string) bool {
	for _, r := range rs {
		if r.cond == normalise(media) {
			return true
		}
	}
	return false
}

// propIn is prop inside the given @media condition. A condition the page does
// not have is a failure, not an empty value: a renamed breakpoint must not turn
// a test into one that checks nothing.
func (rs ruleSet) propIn(t testing.TB, media, sel, property string) string {
	t.Helper()
	if media != "" && !rs.hasMedia(media) {
		t.Fatalf("the page CSS has no @media %s", media)
	}
	return rs.lookup(media, sel, property)
}

func (rs ruleSet) lookup(media, sel, property string) string {
	media = normalise(media)
	val := ""
	for _, r := range rs {
		if r.cond == media && r.sel == normalise(sel) {
			if v, ok := r.props[property]; ok {
				val = v
			}
		}
	}
	return val
}

func TestCSSRulesReader(t *testing.T) {
	rs := cssRules(`
a,b{color:red;margin: 0 }
@media (max-width: 720px){
  a{color:blue;grid-template-columns: minmax(0, 1fr)}
}
@keyframes k{0%{opacity:0}100%{opacity:1}}
.x{display:grid}`)
	if got := rs.prop("b", "color"); got != "red" {
		t.Errorf("selector list: %q", got)
	}
	if got := rs.prop("a", "margin"); got != "0" {
		t.Errorf("trimmed value: %q", got)
	}
	if got := rs.propIn(t, "(max-width:720px)", "a", "grid-template-columns"); got != "minmax(0,1fr)" {
		t.Errorf("media rule, normalised: %q", got)
	}
	if rs.hasMedia("(max-width:480px)") || !rs.hasMedia("(max-width: 720px)") {
		t.Error("hasMedia must know the media of the page, ignoring blanks, and only those")
	}
	if got := rs.prop("a", "grid-template-columns"); got != "" {
		t.Errorf("a media rule leaked outside its media: %q", got)
	}
	if got := rs.prop(".x", "display"); got != "grid" {
		t.Errorf("rule after a @keyframes: %q", got)
	}
	if got := rs.prop("0%", "opacity"); got != "" {
		t.Errorf("@keyframes steps are not rules: %q", got)
	}
}

// Comments are not rules: a commented-out rule must not count, braces or an
// @media inside a comment must not confuse the reader, and what is written in
// a string is not CSS.
func TestCSSRulesReaderIgnoresComments(t *testing.T) {
	rs := cssRules(`
/* header { with: braces } */
a{color:red /* inline; comment */;margin:0}
/* .gone{display:none} */
@media (max-width:720px){ /* @media (min-width:1px){ x{y:z} } */
  a{color:blue}
  /* b{color:green} */
}
.q{content:"/* not a comment */";quotes:'}' '{'}
.after{display:grid}
/* unterminated comment at the end: a{color:black} `)
	if got := rs.prop("a", "color"); got != "red" {
		t.Errorf("a comment inside a declaration: color = %q", got)
	}
	if got := rs.prop("a", "margin"); got != "0" {
		t.Errorf("margin = %q", got)
	}
	if got := rs.propIn(t, "(max-width:720px)", "a", "color"); got != "blue" {
		t.Errorf("rule inside @media next to a comment: %q", got)
	}
	for _, gone := range []string{".gone", "b", "x", "header"} {
		for _, r := range rs {
			if r.sel == gone {
				t.Errorf("%q is in a comment and must not be a rule", gone)
			}
		}
	}
	if rs.hasMedia("(min-width:1px)") {
		t.Error("an @media inside a comment is not a media")
	}
	if got := rs.prop(".q", "content"); got != `"/* not a comment */"` {
		t.Errorf("a string that looks like a comment: %q", got)
	}
	if got := rs.prop(".after", "display"); got != "grid" {
		t.Errorf("a rule after braces in strings: %q", got)
	}
}

// @supports is a conditional group like @media: its rules are read, keyed by
// the condition, and the rules next to it are not lost.
func TestCSSRulesReaderUnderstandsSupports(t *testing.T) {
	rs := cssRules(`
@charset "utf-8";
@import url("x.css");
.a{display:block}
@supports (display: grid){
  .a{display:grid;gap:1px}
  @media (max-width:720px){ .a{gap:0} }
}
@supports not (aspect-ratio:1){ .b{height:10px} }
.c:is(.x, .y){color:red}
.d{background:url(data:image/svg+xml;base64,AAA=);color:blue}`)
	if got := rs.prop(".a", "display"); got != "block" {
		t.Errorf("a rule after @charset and @import: %q", got)
	}
	if got := rs.propIn(t, "supports (display:grid)", ".a", "display"); got != "grid" {
		t.Errorf("rule inside @supports: %q", got)
	}
	if got := rs.prop(".a", "gap"); got != "" {
		t.Errorf("an @supports rule leaked outside it: %q", got)
	}
	if got := rs.propIn(t, "supports (display:grid) && (max-width:720px)", ".a", "gap"); got != "0" {
		t.Errorf("@media inside @supports: %q", got)
	}
	if got := rs.propIn(t, "supports not (aspect-ratio:1)", ".b", "height"); got != "10px" {
		t.Errorf("@supports not: %q", got)
	}
	if got := rs.prop(".c:is(.x,.y)", "color"); got != "red" {
		t.Errorf("a comma inside :is() is not a selector list: %q", got)
	}
	if got := rs.prop(".d", "color"); got != "blue" {
		t.Errorf("a ; inside url(...) is not the end of a declaration: %q", got)
	}
	if !strings.Contains(rs.prop(".d", "background"), "base64,AAA=") {
		t.Errorf("background = %q", rs.prop(".d", "background"))
	}
}
