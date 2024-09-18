package lexer

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
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
		{code: "\"TAB\"\"LE\"", tokens: parseTokens(`<"\"TAB\"\"LE\"", Identifier> <EOF>`)},
		{code: "\"TABLE", tokens: parseTokens(`<"\"TABLE: unexpected EOF", Error> <EOF>`)},
		{code: "[TABLE]", tokens: parseTokens(`<"[TABLE]", Identifier> <EOF>`)},
		{code: "[TABLE", tokens: parseTokens(`<"[TABLE: unexpected EOF", Error> <EOF>`)},
		{code: "`TABLE`", tokens: parseTokens("<\"`TABLE`\", Identifier> <EOF>")},
		{code: "`TAB``LE`", tokens: parseTokens("<\"`TAB``LE`\", Identifier> <EOF>")},
		{code: "`TABLE", tokens: parseTokens("<\"`TABLE: unexpected EOF\", Error> <EOF>")},
		{code: "'TABLE'", tokens: parseTokens("<\"'TABLE'\", String> <EOF>")},
		{code: "'TAB''LE'", tokens: parseTokens("<\"'TAB''LE'\", String> <EOF>")},
		{code: "X'CAFE'", tokens: parseTokens("<\"X'CAFE'\", Blob> <EOF>")},
		{code: "x'cafe'", tokens: parseTokens("<\"x'cafe'\", Blob> <EOF>")},
		// the lexer doesn't care if the number of hexadecimal runes is even
		{code: "x'C0FEE'", tokens: parseTokens("<\"x'C0FEE'\", Blob> <EOF>")},
		{code: "x'CAR'", tokens: parseTokens("<\"x'CAR: a blob must contain only hexadecimal characters\", Error> <\"': unexpected EOF\", Error> <EOF>")},
		{code: "1", tokens: parseTokens(`<"1", Numeric> <EOF>`)},
		{code: "1234", tokens: parseTokens(`<"1234", Numeric> <EOF>`)},
		{code: "1_2", tokens: parseTokens(`<"1_2", Numeric> <EOF>`)},
		{code: "_1_2", tokens: parseTokens(`<"_1_2", Identifier> <EOF>`)},
		{code: "1_2_", tokens: parseTokens(`<"1_2", Numeric> <"_", Identifier> <EOF>`)},
		{code: "1.", tokens: parseTokens(`<"1.", Numeric> <EOF>`)},
		{code: "1234.", tokens: parseTokens(`<"1234.", Numeric> <EOF>`)},
		{code: "1_23_4.", tokens: parseTokens(`<"1_23_4.", Numeric> <EOF>`)},
		{code: "1.0", tokens: parseTokens(`<"1.0", Numeric> <EOF>`)},
		{code: "1.9_1", tokens: parseTokens(`<"1.9_1", Numeric> <EOF>`)},
		{code: "1.9E", tokens: parseTokens(`<"1.9", Numeric> <"E", Identifier> <EOF>`)},
		{code: "1.9e", tokens: parseTokens(`<"1.9", Numeric> <"e", Identifier> <EOF>`)},
		{code: "1.9EA", tokens: parseTokens(`<"1.9", Numeric> <"EA", Identifier> <EOF>`)},
		{code: "1.9E8", tokens: parseTokens(`<"1.9E8", Numeric> <EOF>`)},
		{code: "1.9e8_0", tokens: parseTokens(`<"1.9e8_0", Numeric> <EOF>`)},
		{code: "1.9E+8_0", tokens: parseTokens(`<"1.9E+8_0", Numeric> <EOF>`)},
		{code: "1.9e-00", tokens: parseTokens(`<"1.9e-00", Numeric> <EOF>`)},
		{code: "1.9E-", tokens: parseTokens(`<"1.9", Numeric> <"E", Identifier> <"-", Minus> <EOF>`)},
		{code: "1.9e+", tokens: parseTokens(`<"1.9", Numeric> <"e", Identifier> <"+", Plus> <EOF>`)},
		{code: "1.9e+A", tokens: parseTokens(`<"1.9", Numeric> <"e", Identifier> <"+", Plus> <"A", Identifier> <EOF>`)},
		{code: ".99", tokens: parseTokens(`<".99", Numeric> <EOF>`)},
		{code: ".9e+3", tokens: parseTokens(`<".9e+3", Numeric> <EOF>`)},
		{code: ".9_1e+3_2", tokens: parseTokens(`<".9_1e+3_2", Numeric> <EOF>`)},
		{code: "0X92A", tokens: parseTokens(`<"0X92A", Numeric> <EOF>`)},
		{code: "0x0_F", tokens: parseTokens(`<"0x0_F", Numeric> <EOF>`)},
		{code: "0x0_F_", tokens: parseTokens(`<"0x0_F", Numeric> <"_", Identifier> <EOF>`)},
		{code: "0x0_FQ", tokens: parseTokens(`<"0x0_F", Numeric> <"Q", Identifier> <EOF>`)},
		{code: "-- comment", tokens: parseTokens(`<"-- comment", SQLComment> <EOF>`)},
		{code: "-- comment\ntable", tokens: parseTokens(`<"-- comment", SQLComment> <"table", Table> <EOF>`)},
		{code: "/**/", tokens: parseTokens(`<"/**/", CComment> <EOF>`)},
		{code: "/*****/ table", tokens: parseTokens(`<"/*****/", CComment> <"table", Table> <EOF>`)},
		{code: "/**", tokens: parseTokens(`<"/**", CComment> <EOF>`)},
		{code: "/* comment\n table", tokens: parseTokens(`<"/* comment\n table", CComment> <EOF>`)},
		{code: "/* comment*/ table", tokens: parseTokens(`<"/* comment*/", CComment> <"table", Table> <EOF>`)},
		{code: "/* comment* **/ table", tokens: parseTokens(`<"/* comment* **/", CComment> <"table", Table> <EOF>`)},
		{code: "?", tokens: parseTokens(`<"?", QuestionVariable> <EOF>`)},
		{code: "?0", tokens: parseTokens(`<"?0", QuestionVariable> <EOF>`)},
		{code: "?90", tokens: parseTokens(`<"?90", QuestionVariable> <EOF>`)},
		{code: "?75a", tokens: parseTokens(`<"?75", QuestionVariable> <"a", Identifier> <EOF>`)},
		{code: ":table", tokens: parseTokens(`<":table", ColonVariable> <EOF>`)},
		{code: ":tab$le", tokens: parseTokens(`<":tab$le", ColonVariable> <EOF>`)},
		{code: ":_tab$20", tokens: parseTokens(`<":_tab$20", ColonVariable> <EOF>`)},
		{code: ":select *", tokens: parseTokens(`<":select", ColonVariable> <"*", Asterisk> <EOF>`)},
		{code: ":", tokens: parseTokens(`<":: unexpected EOF", Error> <EOF>`)},
		{code: ":2", tokens: parseTokens(`<":: invalid character after colon", Error> <"2", Numeric> <EOF>`)},
		{code: "@table", tokens: parseTokens(`<"@table", AtVariable> <EOF>`)},
		{code: "@tab$le", tokens: parseTokens(`<"@tab$le", AtVariable> <EOF>`)},
		{code: "@_tab$20", tokens: parseTokens(`<"@_tab$20", AtVariable> <EOF>`)},
		{code: "@select *", tokens: parseTokens(`<"@select", AtVariable> <"*", Asterisk> <EOF>`)},
		{code: "@", tokens: parseTokens(`<"@: unexpected EOF", Error> <EOF>`)},
		{code: "@2", tokens: parseTokens(`<"@: invalid character after at", Error> <"2", Numeric> <EOF>`)},
		{code: "-", tokens: parseTokens(`<"-", Minus> <EOF>`)},
		{code: "(", tokens: parseTokens(`<"(", LeftParen> <EOF>`)},
		{code: ")", tokens: parseTokens(`<")", RightParen> <EOF>`)},
		{code: ";", tokens: parseTokens(`<";", Semicolon> <EOF>`)},
		{code: "+", tokens: parseTokens(`<"+", Plus> <EOF>`)},
		{code: "*", tokens: parseTokens(`<"*", Asterisk> <EOF>`)},
		{code: "/", tokens: parseTokens(`<"/", Slash> <EOF>`)},
		{code: "/ * */", tokens: parseTokens(`<"/", Slash> <"*", Asterisk> <"*", Asterisk> <"/", Slash> <EOF>`)},
		{code: "%", tokens: parseTokens(`<"%", Percent> <EOF>`)},
		{code: "=", tokens: parseTokens(`<"=", Equal> <EOF>`)},
		{code: "==", tokens: parseTokens(`<"==", EqualEqual> <EOF>`)},
		{code: "<=", tokens: parseTokens(`<"<=", LessThanOrEqual> <EOF>`)},
		{code: "<>", tokens: parseTokens(`<"<>", LessThanGreaterThan> <EOF>`)},
		{code: "<<", tokens: parseTokens(`<"<<", LessThanLessThan> <EOF>`)},
		{code: "<", tokens: parseTokens(`<"<", LessThan> <EOF>`)},
		{code: ">=", tokens: parseTokens(`<">=", GreaterThanEqual> <EOF>`)},
		{code: ">>", tokens: parseTokens(`<">>", GreaterThanGreaterThan> <EOF>`)},
		{code: ">", tokens: parseTokens(`<">", GreaterThan> <EOF>`)},
		{code: "!=", tokens: parseTokens(`<"!=", ExclamationEqual> <EOF>`)},
		{code: "!", tokens: parseTokens(`<"!: unexpected EOF", Error> <EOF>`)},
		{code: "!a", tokens: parseTokens(`<"!: unexpected character", Error> <"a", Identifier> <EOF>`)},
		{code: ",", tokens: parseTokens(`<",", Comma> <EOF>`)},
		{code: "&", tokens: parseTokens(`<"&", Ampersand> <EOF>`)},
		{code: "~", tokens: parseTokens(`<"~", Tilde> <EOF>`)},
		{code: "|", tokens: parseTokens(`<"|", Pipe> <EOF>`)},
		{code: "||", tokens: parseTokens(`<"||", PipePipe> <EOF>`)},
		{code: ".", tokens: parseTokens(`<".", Dot> <EOF>`)},
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
	re := regexp.MustCompile(`<(?:("(?:[^\\]|\\"|\\[^"])*?"),\s?)?([a-zA-Z]+)>`)
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
		fmt.Fprintf(tw, "\t%s\n", scanned[i])
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

// TestReaderUnreadOnStart tests the case where the unreadRune is called more than one times.
func TestReaderMoreThanOneUnread(t *testing.T) {
	r := newReader([]byte("abcde"))

	r.readRune()
	r.readRune()
	r.readRune()
	r.readRune()
	r.readRune()

	r.unreadRune()
	r.unreadRune()
	r.unreadRune()
	r.unreadRune()
	r.unreadRune()

	rn, eof := r.readRune()
	if eof {
		t.Errorf("at EOF")
		return
	}

	if rn != 'a' {
		t.Errorf("exected 'a', got %q", rn)
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
