package lexical

import (
	"fmt"
	"strings"
	"testing"

	"github.com/joaobnv/mel/sqlite/v3_46_1/lexer"
	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

func TestTransformer(t *testing.T) {
	cases := []struct {
		code         string
		transformers []Transformer
		expected     string
	}{
		{
			code:         "select * where ok IN table a",
			transformers: []Transformer{KeywordToUppercase()},
			expected:     "SELECT * WHERE ok IN TABLE a",
		}, {
			code:         "select *\n\tWHERE ok\n\tin table a",
			transformers: []Transformer{TabToSpace(2)},
			expected:     "select *\n  WHERE ok\n  in table a",
		}, {
			code:         "select *\n\tWHERE ok\n\tin table a",
			transformers: []Transformer{TabToSpace(2), KeywordToUppercase()},
			expected:     "SELECT *\n  WHERE ok\n  IN TABLE a",
		},
		{
			code:         "select * WHERE ok in table a",
			transformers: []Transformer{newRepeat(func(k token.Kind) bool { return k == token.KindSelect }, 2)},
			expected:     "selectselectselect * WHERE ok in table a",
		}, {
			code:         "select * WHERE ok in table a",
			transformers: []Transformer{newRepeat(IsOperator, 2)},
			expected:     "select *** WHERE ok in table a",
		},
	}

	for _, c := range cases {
		tp := NewTokenProvider(lexer.New([]byte(c.code)), Chain(c.transformers...))

		var b strings.Builder
		tok := tp.Next()
		for tok.Kind != token.KindEOF {
			b.Write(tok.Lexeme)
			tok = tp.Next()
		}

		if c.expected != b.String() {
			fmt.Printf("want %q, got %q\n", c.expected, b.String())
			t.Fail()
		}
	}
}

// repeat is a transformer that repeats n times a token.
type repeat struct {
	kindPredicate func(k token.Kind) bool
	n             int
}

func newRepeat(kindPredicate func(k token.Kind) bool, n int) Transformer {
	return &repeat{kindPredicate: kindPredicate, n: n}
}

func (r *repeat) Transform(tok *token.Token) []*token.Token {
	if !r.kindPredicate(tok.Kind) {
		return []*token.Token{tok}
	}
	result := []*token.Token{tok}
	for range r.n {
		result = append(result, token.New(tok.Lexeme, tok.Kind))
	}
	return result
}

func TestTokenKind(t *testing.T) {
	if TokenKindAnsiCode.String() != "AnsiCode" {
		fmt.Printf("TokenKindAnsiCode.String = %q, want \"AnsiCode\"\n", TokenKindAnsiCode.String())
		t.Fail()
	}
	k := tokenKind(2)
	if k.String() != "2" {
		fmt.Printf("k.String = %q, want \"2\"\n", k.String())
		t.Fail()
	}
	if TokenKindAnsiCode.IsKeyword() {
		t.Errorf("%s is a keyword", TokenKindAnsiCode.String())
	}
}
