package lexer

import (
	"bytes"
	"fmt"
	"mel/sqlite/v3_46_1/token"
	"regexp"
	"slices"
	"strings"
	"testing"
	"text/tabwriter"
)

// TestLexer tests the lexer.
func TestLexer(t *testing.T) {
	cases := []struct {
		code   string
		tokens []*token.Token
	}{
		{code: "", tokens: parseTokens(`<EOF>`)},
		{code: "\t\n\x0C\x0D\x20", tokens: parseTokens(`<EOF>`)},
		{code: "abc ", tokens: parseTokens(`<"abc", Identifier> <EOF>`)},
		{code: "TABLE", tokens: parseTokens(`<"TABLE", Table> <EOF>`)},
		{code: "\"TABLE\"", tokens: parseTokens(`<"\"TABLE\"", Identifier> <EOF>`)},
		{code: "\"TABLE", tokens: parseTokens(`<"\"TABLE: unexpected EOF", Error> <EOF>`)},
		{code: "[TABLE]", tokens: parseTokens(`<"[TABLE]", Identifier> <EOF>`)},
		{code: "[TABLE", tokens: parseTokens(`<"[TABLE: unexpected EOF", Error> <EOF>`)},
		{code: "`TABLE`", tokens: parseTokens("<\"`TABLE`\", Identifier> <EOF>")},
		{code: "`TABLE", tokens: parseTokens("<\"`TABLE: unexpected EOF\", Error> <EOF>")},
	}

	for _, c := range cases {
		l := New([]byte(c.code))

		var scanned []*token.Token
		for {
			tok := l.Next()
			scanned = append(scanned, tok)
			if tok.Kind == token.KindEOF {
				break
			}
		}

		equals := slices.EqualFunc(c.tokens, scanned, func(a, b *token.Token) bool {
			if a.Kind != b.Kind {
				return false
			}
			return bytes.Equal(a.Lexeme, b.Lexeme)
		})

		if !equals {
			t.Errorf("code=\"%s\": tokens differ", c.code)
			var b strings.Builder
			printTokens(&b, c.tokens, scanned)
			t.Log(b.String())
		}
	}
}

// parseTokens unmarshalls tkens from code.
func parseTokens(code string) (result []*token.Token) {
	re := regexp.MustCompile(`<(?:("(?:[^\\]|\\"|\\[^"])*"),\s?)?([a-zA-Z]+)>`)
	matches := re.FindAllString(code, -1)
	for i := range matches {
		var tok token.Token
		if err := tok.UnmarshalText([]byte(matches[i])); err != nil {
			panic(fmt.Errorf("\"%s\": %w", matches[i], err))
		}
		result = append(result, &tok)
	}
	return
}

// printTokens writes the tokens to b in tabular form.
func printTokens(b *strings.Builder, expected, scanned []*token.Token) {
	// we use a strings.Builder because it dont returns errors like the more general io.Writer.
	tw := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintln(tw, "expected\tscanned")
	for i := range expected {
		fmt.Fprintf(tw, "%s\t", expected[i])
		if i < len(scanned) {
			fmt.Fprintf(tw, "%s\n", scanned[i])
		} else {
			fmt.Fprintln(tw)
		}
	}
	for i := len(expected); i < len(scanned); i++ {
		fmt.Fprintf(tw, "\t%s", scanned[i])
	}
}

// TestReaderReadPanic tests the case where the reader reads from a invalid UTF-8 encoded byte slice.
func TestReaderReadPanic(t *testing.T) {
	defer func() {
		resultPanic := recover()
		if resultPanic == nil {
			t.Errorf("not panic")
			return
		}
		err := resultPanic.(error)
		if err.Error() != "utf-8 encoding invalid" {
			t.Errorf("invalid error message: %s", err)
		}
	}()
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := newReader(data)
	r.readRune()
}

// TestReaderUnreadOnStart tests the case where the read is at the start and we try a unreadRune.
func TestReaderUnreadOnStart(t *testing.T) {
	data := []byte{0x00}
	r := newReader(data)
	onStart := r.unreadRune()
	if !onStart {
		t.Error("not on start")
	}
}

// TestReaderUnreadPanic tests the case where a unreadRune cannot find a valid start of rune.
func TestReaderUnreadPanic(t *testing.T) {
	defer func() {
		resultPanic := recover()
		if resultPanic == nil {
			t.Errorf("not panic")
			return
		}
		err := resultPanic.(error)
		if err.Error() != "utf-8 encoding invalid" {
			t.Errorf("invalid error message: %s", err)
		}
	}()
	data := []byte{0x00, 0xFF, 0xFF, 0xFF}
	r := newReader(data)
	r.readRune()
	data[0] = 0b10111111
	r.unreadRune()
}
