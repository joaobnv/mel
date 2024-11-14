// This package transforms a sequence of tokens by adding tokens that represent RGB colors. The lexeme
// of that tokens is not suitable to be written to a file, see the subpackage terminal/rgb,
// these package transforms the colors of this package in formats suitables to be written to a file.
//
// The tokens of color added must affect only the next token that is not of the kinds defined in this package.
package color

import (
	"encoding/binary"
	"strconv"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

// RGB represents a RGB color.
type RGB uint32

// NewRGB creates a new RGB color.
func NewRGB(r, g, b byte) RGB {
	return RGB(binary.LittleEndian.Uint32([]byte{0, r, g, b}))
}

// MarshalLexeme marshalls c to a representation suitable to be used as the lexeme of a token.
func (c RGB) MarshalLexeme() [4]byte {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, uint32(c))
	return [4]byte(b)
}

// UnmarshalLexeme unmarshalls the lexeme into c.
func (c *RGB) UnmarshalLexeme(lexeme [4]byte) {
	*c = RGB(binary.LittleEndian.Uint32(lexeme[0:4]))
}

// rgb extracts the RGB components c.
func (c RGB) Components() (r, g, b byte) {
	var p []byte
	p = binary.LittleEndian.AppendUint32(p, uint32(c))
	r = p[1]
	g = p[2]
	b = p[3]
	return
}

// Nil is for permit the specification that a color must not be set.
var Nil = RGB(0x01000000)

// Transformer is a lexical.Transformer that adds tokens that represent RGB colors.
type Transformer struct {
	// kindPredicate determines for which kinds of tokens the transformation should take place.
	kindPredicate func(token.Kind) bool
	// foregroundColor is the foreground color that will be applied. If it is Nil then the foreground color will not be set.
	foregroundColor RGB
	// backgroundColor is the background color that will be applied. If it is Nil then the background color will not be Set.
	backgroundColor RGB
}

// NewTransformer creates a Transformer. foregroundColor is the foreground color that will be applied.
// If it is Nil then the foreground color will not be set. backgroundColor is the
// background color that will be applied. If it is Nil then the background color will
// not be set.
func NewTransformer(kindPredicate func(token.Kind) bool, foregroundColor, backgroundColor RGB) *Transformer {
	return &Transformer{kindPredicate: kindPredicate, foregroundColor: foregroundColor, backgroundColor: backgroundColor}
}

// Transform implements lexical.Transformer. It adds tokens with a RGB color if kindPredicate returns true for the kind of tok.
// The tokens added has kind equals TokenKindForegroundColor or TokenKindBackgroundColor.
func (t *Transformer) Transform(tok *token.Token) []*token.Token {
	if t.kindPredicate(tok.Kind) {
		return t.transform(tok)
	}
	return []*token.Token{tok}
}

// transform apply the transformation without considering kindPredicate.
func (t *Transformer) transform(tok *token.Token) []*token.Token {
	var result []*token.Token
	if t.foregroundColor != Nil {
		l := t.foregroundColor.MarshalLexeme()
		result = append(result, token.New(l[0:4], TokenKindForegroundColor))
	}
	if t.backgroundColor != Nil {
		l := t.backgroundColor.MarshalLexeme()
		result = append(result, token.New(l[0:4], TokenKindBackgroundColor))
	}
	result = append(result, tok)
	return result
}

// Transformers is a lexical.Transformer with appy a set of Transformer. It is more eficient than a chain
// of Transformer. It is not equivalent to a chain of Transformer because it choose the first Transformer
// for wich the kindPredicate returns true.
type Transformers []*Transformer

// NewTransformers creates a Transformers.
func NewTransformers(cts ...*Transformer) Transformers {
	return Transformers(cts)
}

// Transform implements lexical.Transformer.
func (cts Transformers) Transform(tok *token.Token) []*token.Token {
	for i := range cts {
		if cts[i].kindPredicate(tok.Kind) {
			return cts[i].transform(tok)
		}
	}
	return []*token.Token{tok}
}

// tokenKind is a type for token kinds speceific to this package.
type tokenKind int

var (
	tokenKindForegroundColor            = tokenKind(0)
	tokenKindBackgroundColor            = tokenKind(1)
	TokenKindForegroundColor token.Kind = &tokenKindForegroundColor
	TokenKindBackgroundColor token.Kind = &tokenKindBackgroundColor
)

// String returns a string representation of k.
func (k *tokenKind) String() string {
	if *k < 0 || int(*k) >= len(tokenKindStrings) {
		return strconv.Itoa(int(*k))
	}
	return tokenKindStrings[*k]
}

// tokenKindStrings contains the string representation of the token kinds specific to this package.
// Note that the value of a tokenKind is the index of your string representation.
var tokenKindStrings = []string{
	"ForegroundColor", "BackgroundColor",
}
