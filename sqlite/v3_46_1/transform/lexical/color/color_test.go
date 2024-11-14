package color

import (
	"fmt"
	"strings"
	"testing"

	"github.com/joaobnv/mel/sqlite/v3_46_1/lexer"
	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
	"github.com/joaobnv/mel/sqlite/v3_46_1/transform/lexical"
)

func TestColor(t *testing.T) {
	lex := lexer.New([]byte("select * where"))
	tr := NewTransformer(lexical.IsKeyword, NewRGB(0, 0, 0), NewRGB(0xAA, 0xAA, 0xAA))
	tp := lexical.NewTokenProvider(lex, tr)

	var b strings.Builder
	tok := tp.Next()
	for tok.Kind != token.KindEOF {
		b.Write(tok.Lexeme)
		tok = tp.Next()
	}

	expected := "\x00\x00\x00\x00\x00\xaa\xaa\xaaselect * \x00\x00\x00\x00\x00\xaa\xaa\xaawhere"
	if expected != b.String() {
		fmt.Printf("want %q, got %q\n", expected, b.String())
		t.Fail()
	}
}

func TestTransformers(t *testing.T) {
	lex := lexer.New([]byte("update table_a where"))
	tr := NewTransformers(NewTransformer(lexical.IsKeyword, Nil, NewRGB(0xAA, 0xAA, 0xAA)))
	tp := lexical.NewTokenProvider(lex, tr)

	var b strings.Builder
	tok := tp.Next()
	for tok.Kind != token.KindEOF {
		b.Write(tok.Lexeme)
		tok = tp.Next()
	}

	expected := "\x00\xaa\xaa\xaaupdate table_a \x00\xaa\xaa\xaawhere"
	if expected != b.String() {
		fmt.Printf("want %q, got %q\n", expected, b.String())
		t.Fail()
	}
}

func TestMarshal(t *testing.T) {
	c := NewRGB(0xAA, 0xBB, 0xCC)
	b := c.MarshalLexeme()
	var d RGB
	d.UnmarshalLexeme(b)

	rc, gc, bc := c.Components()
	rd, gd, bd := c.Components()

	if rc != rd {
		fmt.Printf("want red %x, got %x", rc, rd)
		t.Fail()
	}

	if gc != gd {
		fmt.Printf("want green %x, got %x", gc, gd)
		t.Fail()
	}

	if bc != bd {
		fmt.Printf("want blue %x, got %x", bc, bd)
		t.Fail()
	}
}

func TestTokenKindString(t *testing.T) {
	if TokenKindBackgroundColor.String() != "BackgroundColor" {
		fmt.Printf("want %s, got %s", "BackgroundColor", TokenKindBackgroundColor)
		t.Fail()
	}
	if TokenKindForegroundColor.String() != "ForegroundColor" {
		fmt.Printf("want %s, got %s", "ForegroundColor", TokenKindForegroundColor)
		t.Fail()
	}
	k := tokenKind(-1)
	if k.String() != "-1" {
		fmt.Printf("want %s, got %s", "-1", k.String())
		t.Fail()
	}
}
