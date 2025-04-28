// This package transforms a sequence of tokens by replacing tokens with kind equals color.TokenKindForegroundColor or
// color.TokenKindForegroundColor by RGB colors in the format of ANSI escape codes. The codes of the color follow the
// pattern: CSI 38;2;r;g;b m (foreground) or CSI 2;r;g;b m (background). This package uses CSI 0 m after an token that
// received a color.
// See https://en.wikipedia.org/wiki/ANSI_escape_code.
package rgb

import (
	"fmt"
	"strconv"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
	"github.com/joaobnv/mel/sqlite/v3_46_1/transform/lexical/color"
)

// resetToken is the token used for removing the color that was applied to a token.
var resetToken = token.New([]byte("\x1B[0m"), TokenKindReset)

// Transformer is a lexical.Transformer that operates as specified in the documentation for this package.
type Transformer struct {
	// previousToken stores the previousToken passed to Transform. It is used to determine when is necessary to
	// put resetToken.
	previousToken *token.Token
}

// NewTransformer creates a Transformer.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform implements lexical.Transformer.
func (t *Transformer) Transform(tok *token.Token) []*token.Token {
	if tok.Kind == color.TokenKindForegroundColor {
		r, g, b := t.rgb(tok)
		newTok := token.New([]byte(fmt.Sprintf("\x1B[38;2;%d;%d;%dm", r, g, b)), TokenKindColor)
		t.previousToken = newTok
		return []*token.Token{newTok}
	} else if tok.Kind == color.TokenKindBackgroundColor {
		r, g, b := t.rgb(tok)
		newTok := token.New([]byte(fmt.Sprintf("\x1B[48;2;%d;%d;%dm", r, g, b)), TokenKindColor)
		t.previousToken = newTok
		return []*token.Token{newTok}
	} else if t.previousToken.Kind == TokenKindColor {
		t.previousToken = resetToken
		return []*token.Token{tok, resetToken}
	}
	t.previousToken = tok
	return []*token.Token{tok}
}

// rgb extracts the RGB components of tok.
func (t *Transformer) rgb(tok *token.Token) (r, g, b byte) {
	var c color.RGB
	c.UnmarshalLexeme([4]byte(tok.Lexeme))
	return c.Components()
}

// tokenKind is a type for token kinds speceific to this package.
type tokenKind int

var (
	tokenKindColor = tokenKind(0)
	tokenKindReset = tokenKind(1)
	// TokenKindColor is a kind of tokens for colors as defined by this package. One token with this kind
	// can change the color of the next token only.
	TokenKindColor token.Kind = &tokenKindColor
	// TokenKindReset represents the ansi escape code of reset of graphic redition, that is, CSI 0 m.
	TokenKindReset token.Kind = &tokenKindReset
)

// String returns a string representation of k.
func (k *tokenKind) String() string {
	if *k < 0 || int(*k) >= len(tokenKindStrings) {
		return strconv.Itoa(int(*k))
	}
	return tokenKindStrings[*k]
}

// IsKeyword reports whether this kind is of a keyword.
func (k *tokenKind) IsKeyword() bool {
	return false
}

// tokenKindStrings contains the string representation of the token kinds specific to this package.
// Note that the value of a tokenKind is the index of your string representation.
var tokenKindStrings = []string{
	"Color", "Reset",
}
