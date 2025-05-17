package rgb

import (
	"fmt"
	"strings"
	"testing"

	"github.com/joaobnv/mel/dit/lexer"
	"github.com/joaobnv/mel/dit/token"
	"github.com/joaobnv/mel/dit/transform/lexical"
	"github.com/joaobnv/mel/dit/transform/lexical/color"
)

func TestColor(t *testing.T) {
	lex := lexer.New([]byte("select * where"))
	tr := lexical.Chain(
		color.NewTransformer(lexical.IsKeyword, color.NewRGB(0xAA, 0xAA, 0xAA), color.NewRGB(0xBB, 0xBB, 0xBB)),
		NewTransformer(),
	)
	tp := lexical.NewTokenProvider(lex, tr)

	var b strings.Builder
	tok := tp.Next()
	for tok.Kind != token.KindEOF {
		b.Write(tok.Lexeme)
		tok = tp.Next()
	}

	expected := "\x1B[38;2;170;170;170m\x1B[48;2;187;187;187mselect\x1B[0m * \x1B[38;2;170;170;170m\x1B[48;2;187;187;187mwhere\x1B[0m"
	if expected != b.String() {
		fmt.Printf("want %q, got %q\n", expected, b.String())
		t.Fail()
	}
}

func TestTransformers(t *testing.T) {
	lex := lexer.New([]byte("update table_a where"))
	tr := lexical.Chain(
		color.NewTransformer(lexical.IsKeyword, color.Nil, color.NewRGB(0x00, 0xBB, 0xBB)),
		NewTransformer(),
	)
	tp := lexical.NewTokenProvider(lex, tr)

	var b strings.Builder
	tok := tp.Next()
	for tok.Kind != token.KindEOF {
		b.Write(tok.Lexeme)
		tok = tp.Next()
	}

	expected := "\x1B[48;2;0;187;187mupdate\x1B[0m table_a \x1B[48;2;0;187;187mwhere\x1B[0m"
	if expected != b.String() {
		fmt.Printf("want %q, got %q\n", expected, b.String())
		t.Fail()
	}
}

func TestTokenKind(t *testing.T) {
	if TokenKindColor.String() != "Color" {
		fmt.Printf("want %s, got %s", "Color", TokenKindColor.String())
		t.Fail()
	}
	if TokenKindReset.String() != "Reset" {
		fmt.Printf("want %s, got %s", "Reset", TokenKindReset.String())
		t.Fail()
	}

	k := tokenKind(-1)
	if k.String() != "-1" {
		fmt.Printf("want %s, got %s", "-1", k.String())
		t.Fail()
	}
	if TokenKindColor.IsKeyword() {
		t.Errorf("%s is a keyword", TokenKindColor.String())
	}
}
