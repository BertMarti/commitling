package gallery

import (
	"regexp"
	"strings"
	"testing"
)

// A tiny reader of the page's own CSS, so the tests can ask "what does this
// selector say about this property?" instead of looking for a literal piece
// of text (which breaks when a property is reordered and says nothing about
// the intent). It understands plain rules, selector lists and @media blocks,
// which is all the page uses; @keyframes and the like are skipped.

type cssRule struct {
	media string // the text of the @media condition, "" outside any
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

func cssRules(css string) ruleSet {
	var out ruleSet
	var walk func(src, media string)
	walk = func(src, media string) {
		for {
			open := strings.Index(src, "{")
			if open < 0 {
				return
			}
			head := strings.TrimSpace(src[:open])
			depth, end := 1, open+1
			for end < len(src) && depth > 0 {
				switch src[end] {
				case '{':
					depth++
				case '}':
					depth--
				}
				end++
			}
			body := src[open+1 : end-1]
			src = src[end:]
			switch {
			case strings.HasPrefix(head, "@media"):
				walk(body, normalise(strings.TrimPrefix(head, "@media")))
			case strings.HasPrefix(head, "@"):
				// @keyframes, @font-face...: not rules about the layout
			default:
				props := map[string]string{}
				for _, decl := range strings.Split(body, ";") {
					if name, val, ok := strings.Cut(decl, ":"); ok {
						props[strings.TrimSpace(name)] = normalise(val)
					}
				}
				for _, sel := range strings.Split(head, ",") {
					out = append(out, cssRule{media: media, sel: normalise(sel), props: props})
				}
			}
		}
	}
	walk(css, "")
	return out
}

// prop is the value of property for sel outside any @media (the last rule wins).
func (rs ruleSet) prop(sel, property string) string { return rs.lookup("", sel, property) }

// hasMedia reports whether the CSS has any rule inside that @media condition.
func (rs ruleSet) hasMedia(media string) bool {
	for _, r := range rs {
		if r.media == normalise(media) {
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
		if r.media == media && r.sel == normalise(sel) {
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
