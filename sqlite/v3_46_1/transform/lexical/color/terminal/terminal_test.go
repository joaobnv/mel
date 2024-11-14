package terminal

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
	tr := NewTransformer(lexical.IsKeyword, ForegroundBlue, BackgroundCyan)
	tp := lexical.NewTokenProvider(lex, tr)

	var b strings.Builder
	tok := tp.Next()
	for tok.Kind != token.KindEOF {
		b.Write(tok.Lexeme)
		tok = tp.Next()
	}

	expected := "\x1B[34;46mselect\x1B[0m * \x1B[34;46mwhere\x1B[0m"
	if expected != b.String() {
		fmt.Printf("want %q, got %q\n", expected, b.String())
		t.Fail()
	}
}

func TestTransformers(t *testing.T) {
	lex := lexer.New([]byte("update table_a where"))
	tr := NewTransformers(NewTransformer(lexical.IsKeyword, ForegroundNil, BackgroundMagenta))
	tp := lexical.NewTokenProvider(lex, tr)

	var b strings.Builder
	tok := tp.Next()
	for tok.Kind != token.KindEOF {
		b.Write(tok.Lexeme)
		tok = tp.Next()
	}

	expected := "\x1B[45mupdate\x1B[0m table_a \x1B[45mwhere\x1B[0m"
	if expected != b.String() {
		fmt.Printf("want %q, got %q\n", expected, b.String())
		t.Fail()
	}
}

func TestTokenKindString(t *testing.T) {
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
}
