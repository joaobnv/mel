// This package transforms a sequence of tokens by adding tokens that represent colors in the format of ANSI escape codes.
// It supports the colors with code between 30 and 37, inclusive, that is, color for wich n is between 30 and 37,
// inclusive, in the code CSI n m. It also support the colors with code between 40 and 47. This package uses CSI 0 m after
// an token that received a color.
// See https://en.wikipedia.org/wiki/ANSI_escape_code.
package terminal

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/joaobnv/mel/dit/token"
)

// Foreground represents a foreground color.
type Foreground int

const (
	ForegroundBlack Foreground = iota + 30
	ForegroundRed
	ForegroundGreen
	ForegroundYellow
	ForegroundBlue
	ForegroundMagenta
	ForegroundCyan
	ForegroundWhite
	// the foreground color must not be set.
	ForegroundNil Foreground = 0
)

// Background represents a foreground color.
type Background int

const (
	BackgroundBlack Background = iota + 40
	BackgroundRed
	BackgroundGreen
	BackgroundYellow
	BackgroundBlue
	BackgroundMagenta
	BackgroundCyan
	BackgroundWhite
	// the background color must not be set.
	BackgroundNil Background = 0
)

// resetToken is the token used for removing the color that was applied to a token.
var resetToken = token.New([]byte("\x1B[0m"), TokenKindReset)

// Transformer is a lexical.Transformer that apply colors to tokens.
type Transformer struct {
	// kindPredicate determines for which kinds of tokens the transformation should take place.
	kindPredicate func(token.Kind) bool
	// foreground is the foreground color that will be applied to the next token. If it is ForegroundNil then
	// the foreground color of the next token will not be set.
	foreground Foreground
	// background is the background color that will be applied to the next token. If it is BackgroundNil then
	// the background color of the next token will not be Set.
	background Background
}

// NewTransformer creates a ColorTransformer.
func NewTransformer(kindPredicate func(token.Kind) bool, fg Foreground, bg Background) *Transformer {
	return &Transformer{kindPredicate: kindPredicate, foreground: fg, background: bg}
}

// Transform implements lexical.Transformer. It adds tokens with Color if kindPredicate returns true for the kind of tok.
// Also it adds tokens for resetting the color after the token that did have the color changed.
func (ct *Transformer) Transform(tok *token.Token) []*token.Token {
	if ct.kindPredicate(tok.Kind) {
		return ct.transform(tok)
	}
	return []*token.Token{tok}
}

// transform apply the transformation without considering kindPredicate.
func (ct *Transformer) transform(tok *token.Token) []*token.Token {
	var result []*token.Token
	var colorChanged bool
	var code string
	if ct.foreground != ForegroundNil {
		code = fmt.Sprintf("%d", ct.foreground)
		colorChanged = true
	}
	if ct.background != BackgroundNil {
		if colorChanged {
			code += ";"
		}
		code += fmt.Sprintf("%d", ct.background)
		colorChanged = true
	}
	if colorChanged {
		result = append(result, token.New([]byte("\x1B["+code+"m"), TokenKindColor))
	}
	result = append(result, tok)
	if colorChanged {
		result = append(result, resetToken)
	}
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
	ind := slices.IndexFunc(cts, func(ct *Transformer) bool { return ct.kindPredicate(tok.Kind) })
	if ind == -1 {
		return []*token.Token{tok}
	}
	return cts[ind].transform(tok)
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
