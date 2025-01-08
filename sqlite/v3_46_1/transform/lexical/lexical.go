// This package deals with transformations of the SQL code using only lexical information.
package lexical

import (
	"bytes"
	"strconv"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

// TokenProvider is a interface for tokens providers, like the v3_46_1/lexer.Lexer and the Transformer.
type TokenProvider interface {
	// Next returns the next token.
	Next() *token.Token
}

// Transformer transforms tokens.
type Transformer interface {
	// Transform transforms tok.
	Transform(tok *token.Token) []*token.Token
}

// Chain creates a Transformer that applies t in sequence.
func Chain(t ...Transformer) Transformer {
	return &chain{t: t}
}

// chain is a Transformer that apply t in sequence.
type chain struct {
	// t are the transformers that will be applied in sequence.
	t []Transformer
}

// Transform implements Transformer.
func (c *chain) Transform(tok *token.Token) []*token.Token {
	return c.apply(tok, c.t...)
}

// apply transforms tok applying chain in sequence.
func (c *chain) apply(tok *token.Token, chain ...Transformer) (result []*token.Token) {
	toks := chain[0].Transform(tok)
	if len(chain) == 1 {
		return toks
	}
	for i := range toks {
		result = append(result, c.apply(toks[i], chain[1:]...)...)
	}
	return
}

// tokenProvider is a TokenProvider that tranforms the tokens.
type tokenProvider struct {
	// tp is where the tokens that will be transformed come from.
	tp TokenProvider
	// t transforms the tokens returned by tp.
	t Transformer
	// tokens is the next tokens that will be returned by Next. A execution of t can
	// return more than one token but Next can return only one.
	tokens []*token.Token
	// pos is the position on tokens
	pos int
}

// NewTokenProvider creates a new TokenProvider that applies t to the tokens provided by tp.
func NewTokenProvider(tp TokenProvider, t Transformer) TokenProvider {
	return &tokenProvider{tp: tp, t: t}
}

// Next returns the next token after applying the transformer to the token returned by the underlying TokenProvider.
func (tp *tokenProvider) Next() *token.Token {
	if len(tp.tokens) > 0 && tp.pos < len(tp.tokens) {
		tok := tp.tokens[tp.pos]
		tp.pos++
		return tok
	}

	if tp.pos == len(tp.tokens) { // if the t.tokens was totally consumed
		tp.tokens = nil
		tp.pos = 0
	}

	tok := tp.tp.Next()
	tp.tokens = tp.t.Transform(tok)
	tok = tp.tokens[0]
	tp.pos++
	return tok
}

// lexemeTransformer is a Transformer that transforms the lexemes of the tokens.
type lexemeTransformer struct {
	// kindPredicate determines for which kinds of tokens the transformation should take place.
	kindPredicate func(token.Kind) bool
	// transformer is the function that will be apllied to transform the lexeme of the token.
	transformer func([]byte) []byte
}

// NewLexemeTransformer creates a Transformer that transforms the lexeme of the tokens for which the kindPredicate returns true.
// The returned Transformer transforms the lexeme applying the function transformer.
func NewLexemeTransformer(kindPredicate func(token.Kind) bool, transformer func([]byte) []byte) Transformer {
	return &lexemeTransformer{kindPredicate: kindPredicate, transformer: transformer}
}

// Transform implements Transformer.
func (lt *lexemeTransformer) Transform(tok *token.Token) []*token.Token {
	if lt.kindPredicate(tok.Kind) {
		return []*token.Token{token.New(lt.transformer(tok.Lexeme), tok.Kind)}
	}
	return []*token.Token{tok}
}

// KeywordToUppercase creates a Transformer that changes the token lexeme to uppercase if the token is a keyword.
// Determining whether a token is a keyword uses the IsKeyword function.
func KeywordToUppercase() Transformer {
	return NewLexemeTransformer(IsKeyword, bytes.ToUpper)
}

// TabToSpace creates a transformer that changes the '\t' character of white spaces to n spaces.
func TabToSpace(n int) Transformer {
	return NewLexemeTransformer(
		func(k token.Kind) bool { return k == token.KindWhiteSpace },
		func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("\t"), bytes.Repeat([]byte(" "), n))
		},
	)
}

// IsKeyword reports if k represents an keyword. It uses only the kinds exported by v3_46_1/token.
func IsKeyword(k token.Kind) bool {
	switch k {
	case token.KindAbort, token.KindAction, token.KindAdd, token.KindAfter, token.KindAll, token.KindAlter, token.KindAlways,
		token.KindAnalyze, token.KindAnd, token.KindAs, token.KindAsc, token.KindAttach, token.KindAutoincrement, token.KindBefore,
		token.KindBegin, token.KindBetween, token.KindBy, token.KindCascade, token.KindCase, token.KindCast, token.KindCheck,
		token.KindCollate, token.KindColumn, token.KindCommit, token.KindConflict, token.KindConstraint, token.KindCreate,
		token.KindCross, token.KindCurrent, token.KindCurrentDate, token.KindCurrentTime, token.KindCurrentTimestamp,
		token.KindDatabase, token.KindDefault, token.KindDeferrable, token.KindDeferred, token.KindDelete, token.KindDesc,
		token.KindDetach, token.KindDistinct, token.KindDo, token.KindDrop, token.KindEach, token.KindElse, token.KindEnd,
		token.KindEscape, token.KindExcept, token.KindExclude, token.KindExclusive, token.KindExists, token.KindExplain,
		token.KindFail, token.KindFilter, token.KindFirst, token.KindFollowing, token.KindFor, token.KindForeign, token.KindFrom,
		token.KindFull, token.KindGenerated, token.KindGlob, token.KindGroup, token.KindGroups, token.KindHaving, token.KindIf,
		token.KindIgnore, token.KindImmediate, token.KindIn, token.KindIndex, token.KindIndexed, token.KindInitially,
		token.KindInner, token.KindInsert, token.KindInstead, token.KindIntersect, token.KindInto, token.KindIs, token.KindIsnull,
		token.KindJoin, token.KindKey, token.KindLast, token.KindLeft, token.KindLike, token.KindLimit, token.KindMatch,
		token.KindMaterialized, token.KindNatural, token.KindNo, token.KindNot, token.KindNothing, token.KindNotnull,
		token.KindNull, token.KindNulls, token.KindOf, token.KindOffset, token.KindOn, token.KindOr, token.KindOrder,
		token.KindOthers, token.KindOuter, token.KindOver, token.KindPartition, token.KindPlan, token.KindPragma,
		token.KindPreceding, token.KindPrimary, token.KindQuery, token.KindRaise, token.KindRange, token.KindRecursive,
		token.KindReferences, token.KindRegexp, token.KindReindex, token.KindRelease, token.KindRename, token.KindReplace,
		token.KindRestrict, token.KindReturning, token.KindRight, token.KindRollback, token.KindRow, token.KindRows,
		token.KindSavepoint, token.KindSelect, token.KindSet, token.KindTable, token.KindTemp, token.KindTemporary,
		token.KindThen, token.KindTies, token.KindTo, token.KindTransaction, token.KindTrigger, token.KindUnbounded,
		token.KindUnion, token.KindUnique, token.KindUpdate, token.KindUsing, token.KindVacuum, token.KindValues, token.KindView,
		token.KindVirtual, token.KindWhen, token.KindWhere, token.KindWindow, token.KindWith, token.KindWithout:
		return true
	}
	return false
}

// IsOperator reports if k represents an operator. It uses only the kinds exported by v3_46_1/token.
func IsOperator(k token.Kind) bool {
	switch k {
	case token.KindMinus, token.KindLeftParen, token.KindRightParen, token.KindSemicolon, token.KindPlus, token.KindAsterisk,
		token.KindSlash, token.KindPercent, token.KindEqual, token.KindEqualEqual, token.KindLessThanOrEqual,
		token.KindLessThanGreaterThan, token.KindLessThanLessThan, token.KindLessThan, token.KindGreaterThanOrEqual,
		token.KindGreaterThanGreaterThan, token.KindGreaterThan, token.KindExclamationEqual, token.KindComma, token.KindAmpersand,
		token.KindTilde, token.KindPipe, token.KindPipePipe, token.KindDot:
		return true
	}
	return false
}

// tokenKind is a type for token kinds speceific to this package.
type tokenKind int

var (
	tokenKindAnsiCode            = tokenKind(0)
	TokenKindAnsiCode token.Kind = &tokenKindAnsiCode
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
	"AnsiCode",
}
