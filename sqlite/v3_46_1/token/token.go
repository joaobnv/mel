// This package deals with tokens.
package token

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Kind is the type of a token.
type Kind int

const (
	// key words
	KindAbort Kind = iota
	KindAction
	KindAdd
	KindAfter
	KindAll
	KindAlter
	KindAlways
	KindAnalyze
	KindAnd
	KindAs
	KindAsc
	KindAttach
	KindAutoincrement
	KindBefore
	KindBegin
	KindBetween
	KindBy
	KindCascade
	KindCase
	KindCast
	KindCheck
	KindCollate
	KindColumn
	KindCommit
	KindConflict
	KindConstraint
	KindCreate
	KindCross
	KindCurrent
	KindCurrentDate
	KindCurrentTime
	KindCurrentTimestamp
	KindDatabase
	KindDefault
	KindDeferrable
	KindDeferred
	KindDelete
	KindDesc
	KindDetach
	KindDistinct
	KindDo
	KindDrop
	KindEach
	KindElse
	KindEnd
	KindEscape
	KindExcept
	KindExclude
	KindExclusive
	KindExists
	KindExplain
	KindFail
	KindFilter
	KindFirst
	KindFollowing
	KindFor
	KindForeign
	KindFrom
	KindFull
	KindGenerated
	KindGlob
	KindGroup
	KindGroups
	KindHaving
	KindIf
	KindIgnore
	KindImmediate
	KindIn
	KindIndex
	KindIndexed
	KindInitially
	KindInner
	KindInsert
	KindInstead
	KindIntersect
	KindInto
	KindIs
	KindIsnull
	KindJoin
	KindKey
	KindLast
	KindLeft
	KindLike
	KindLimit
	KindMatch
	KindMaterialized
	KindNatural
	KindNo
	KindNot
	KindNothing
	KindNotnull
	KindNull
	KindNulls
	KindOf
	KindOffset
	KindOn
	KindOr
	KindOrder
	KindOthers
	KindOuter
	KindOver
	KindPartition
	KindPlan
	KindPragma
	KindPreceding
	KindPrimary
	KindQuery
	KindRaise
	KindRange
	KindRecursive
	KindReferences
	KindRegexp
	KindReindex
	KindRelease
	KindRename
	KindReplace
	KindRestrict
	KindReturning
	KindRight
	KindRollback
	KindRow
	KindRows
	KindSavepoint
	KindSelect
	KindSet
	KindTable
	KindTemp
	KindTemporary
	KindThen
	KindTies
	KindTo
	KindTransaction
	KindTrigger
	KindUnbounded
	KindUnion
	KindUnique
	KindUpdate
	KindUsing
	KindVacuum
	KindValues
	KindView
	KindVirtual
	KindWhen
	KindWhere
	KindWindow
	KindWith
	KindWithout
	// end of keywords
	KindIdentifier
	KindString
	KindBlob
	KindNumeric
	KindSQLComment
	KindCComment
	KindQuestionVariable
	KindMinus
	KindLeftParen
	KindRightParen
	KindSemicolon
	KindPlus
	KindAsterisk
	KindSlash
	KindPercent
	KindEqual
	KindEqualEqual
	KindLessThanOrEqual
	KindLessThanGreaterThan
	KindLessThanLessThan
	KindLessThan
	KindGreaterThanEqual
	KindGreaterThanGreaterThan
	KindGreaterThan
	KindExclamationEqual
	KindComma
	KindAmpersand
	KindTilde
	KindPipe
	KindPipePipe
	KindDot
	KindError
	KindEOF
)

// String returns a string representation of k.
func (k Kind) String() string {
	if k < 0 || int(k) >= len(kindString) {
		return strconv.Itoa(int(k))
	}
	return kindString[k]
}

// MarshalText implements encoding.TextMarshaler. err is always nil.
func (k *Kind) MarshalText() (text []byte, err error) {
	return []byte(k.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (k *Kind) UnmarshalText(text []byte) (err error) {
	pos := slices.Index(kindString, string(text))
	if pos == -1 {
		pos, err = strconv.Atoi(string(text))
	}
	*k = Kind(pos)
	return
}

// kindString contains the string representation of the kinds. Note that the value of a kind is the index of your string
// representation.
var kindString = []string{
	"Abort", "Action", "Add", "After", "All", "Alter", "Always", "Analyze", "And", "As", "Asc", "Attach",
	"Autoincrement", "Before", "Begin", "Between", "By", "Cascade", "Case", "Cast", "Check", "Collate",
	"Column", "Commit", "Conflict", "Constraint", "Create", "Cross", "Current", "CurrentDate", "CurrentTime",
	"CurrentTimestamp", "Database", "Default", "Deferrable", "Deferred", "Delete", "Desc", "Detach", "Distinct",
	"Do", "Drop", "Each", "Else", "End", "Escape", "Except", "Exclude", "Exclusive", "Exists", "Explain", "Fail",
	"Filter", "First", "Following", "For", "Foreign", "From", "Full", "Generated", "Glob", "Group", "Groups",
	"Having", "If", "Ignore", "Immediate", "In", "Index", "Indexed", "Initially", "Inner", "Insert", "Instead",
	"Intersect", "Into", "Is", "Isnull", "Join", "Key", "Last", "Left", "Like", "Limit", "Match", "Materialized",
	"Natural", "No", "Not", "Nothing", "Notnull", "Null", "Nulls", "Of", "Offset", "On", "Or", "Order", "Others",
	"Outer", "Over", "Partition", "Plan", "Pragma", "Preceding", "Primary", "Query", "Raise", "Range", "Recursive",
	"References", "Regexp", "Reindex", "Release", "Rename", "Replace", "Restrict", "Returning", "Right", "Rollback",
	"Row", "Rows", "Savepoint", "Select", "Set", "Table", "Temp", "Temporary", "Then", "Ties", "To", "Transaction",
	"Trigger", "Unbounded", "Union", "Unique", "Update", "Using", "Vacuum", "Values", "View", "Virtual", "When",
	"Where", "Window", "With", "Without", "Identifier", "String", "Blob", "Numeric", "SQLComment", "CComment",
	"QuestionVariable", "Minus", "LeftParen", "RightParen", "Semicolon", "Plus", "Asterisk", "Slash", "Percent",
	"Equal", "EqualEqual", "LessThanOrEqual", "LessThanGreaterThan", "LessThanLessThan", "LessThan", "GreaterThanEqual",
	"GreaterThanGreaterThan", "GreaterThan", "ExclamationEqual", "Comma", "Ampersand", "Tilde", "Pipe", "PipePipe",
	"Dot", "Error", "EOF",
}

// Token is a token from the code.
type Token struct {
	// Lexeme is the lexeme of the token.
	Lexeme []byte
	// Kind is the kind of the token.
	Kind Kind
}

// New creates a Token.
func New(lexeme []byte, kind Kind) *Token {
	return &Token{Lexeme: lexeme, Kind: kind}
}

// String returns a string representation of t.
func (t *Token) String() string {
	var b strings.Builder
	b.WriteRune('<')
	if t.Lexeme != nil {
		b.WriteString(strconv.Quote(string(t.Lexeme)))
		b.WriteString(", ")
	}
	b.WriteString(t.Kind.String())
	b.WriteRune('>')
	return b.String()
}

// MarshalText implements encoding.TextMarshaler. err is always nil. The format is <lexeme, kind>, with the
// lexeme quoted (strconv.Quote).
func (t *Token) MarshalText() (text []byte, err error) {
	return []byte(t.String()), nil
}

// reTokenText is for the text unmarshalling of a token.
var reTokenText = regexp.MustCompile(`<(?:("(?:[^\\]|\\"|\\[^"])*"),\s?)?([a-zA-Z]+)>`)

// UnmarshalText implements encoding.TextUnmarshaler.
func (t *Token) UnmarshalText(text []byte) (err error) {
	matches := reTokenText.FindAllSubmatch(text, -1)
	if len(matches) != 1 {
		return errors.New("invalid token text encoding")
	}
	for i := range matches {
		t.Lexeme = nil
		if matches[i][1] != nil {
			var lexemeStr string
			lexemeStr, err = strconv.Unquote(string(matches[i][1]))
			if err != nil {
				return
			}
			t.Lexeme = []byte(lexemeStr)
		}

		if err = t.Kind.UnmarshalText(matches[i][2]); err != nil {
			return
		}
	}
	return
}
