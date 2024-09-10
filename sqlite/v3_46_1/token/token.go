// package token deals with tokens.
package token

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
	KindError
)

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
