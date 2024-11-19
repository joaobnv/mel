package token

import (
	"fmt"
	"strconv"
)

// Kind is the type of a token.
type Kind interface {
	fmt.Stringer
}

// kind is the type of a token needed by this package.
type kind int

const (
	// key words
	kindAbort kind = iota
	kindAction
	kindAdd
	kindAfter
	kindAll
	kindAlter
	kindAlways
	kindAnalyze
	kindAnd
	kindAs
	kindAsc
	kindAttach
	kindAutoincrement
	kindBefore
	kindBegin
	kindBetween
	kindBy
	kindCascade
	kindCase
	kindCast
	kindCheck
	kindCollate
	kindColumn
	kindCommit
	kindConflict
	kindConstraint
	kindCreate
	kindCross
	kindCurrent
	kindCurrentDate
	kindCurrentTime
	kindCurrentTimestamp
	kindDatabase
	kindDefault
	kindDeferrable
	kindDeferred
	kindDelete
	kindDesc
	kindDetach
	kindDistinct
	kindDo
	kindDrop
	kindEach
	kindElse
	kindEnd
	kindEscape
	kindExcept
	kindExclude
	kindExclusive
	kindExists
	kindExplain
	kindFail
	kindFilter
	kindFirst
	kindFollowing
	kindFor
	kindForeign
	kindFrom
	kindFull
	kindGenerated
	kindGlob
	kindGroup
	kindGroups
	kindHaving
	kindIf
	kindIgnore
	kindImmediate
	kindIn
	kindIndex
	kindIndexed
	kindInitially
	kindInner
	kindInsert
	kindInstead
	kindIntersect
	kindInto
	kindIs
	kindIsnull
	kindJoin
	kindKey
	kindLast
	kindLeft
	kindLike
	kindLimit
	kindMatch
	kindMaterialized
	kindNatural
	kindNo
	kindNot
	kindNothing
	kindNotnull
	kindNull
	kindNulls
	kindOf
	kindOffset
	kindOn
	kindOr
	kindOrder
	kindOthers
	kindOuter
	kindOver
	kindPartition
	kindPlan
	kindPragma
	kindPreceding
	kindPrimary
	kindQuery
	kindRaise
	kindRange
	kindRecursive
	kindReferences
	kindRegexp
	kindReindex
	kindRelease
	kindRename
	kindReplace
	kindRestrict
	kindReturning
	kindRight
	kindRollback
	kindRow
	kindRows
	kindSavepoint
	kindSelect
	kindSet
	KindStored
	kindTable
	kindTemp
	kindTemporary
	kindThen
	kindTies
	kindTo
	kindTransaction
	kindTrigger
	kindUnbounded
	kindUnion
	kindUnique
	kindUpdate
	kindUsing
	kindVacuum
	kindValues
	kindView
	kindVirtual
	kindWhen
	kindWhere
	kindWindow
	kindWith
	kindWithout
	// kind =  end of keywords
	kindIdentifier
	kindString
	kindBlob
	kindNumeric
	kindSQLComment
	kindCComment
	kindQuestionVariable
	kindColonVariable
	kindAtVariable
	kindDollarVariable
	kindMinus
	kindLeftParen
	kindRightParen
	kindSemicolon
	kindPlus
	kindAsterisk
	kindSlash
	kindPercent
	kindEqual
	kindEqualEqual
	kindLessThanOrEqual
	kindLessThanGreaterThan
	kindLessThanLessThan
	kindLessThan
	kindGreaterThanEqual
	kindGreaterThanGreaterThan
	kindGreaterThan
	kindExclamationEqual
	kindComma
	kindAmpersand
	kindTilde
	kindPipe
	kindPipePipe
	kindDot
	kindWhiteSpace
	kindErrorUnexpectedEOF
	kindErrorBlobNotHexadecimal
	kindErrorInvalidCharacter
	kindErrorInvalidCharacterAfter
	kindEOF
)

// String returns a string representation of k.
func (k kind) String() string {
	if k < 0 || int(k) >= len(kindStrings) {
		return strconv.Itoa(int(k))
	}
	return kindStrings[k]
}

// kindStrings contains the string representation of the kinds. Note that the value of a kind is the index of your string
// representation.
var kindStrings = []string{
	"Abort", "Action", "Add", "After", "All", "Alter", "Always", "Analyze", "And", "As", "Asc", "Attach", "Autoincrement", "Before", "Begin",
	"Between", "By", "Cascade", "Case", "Cast", "Check", "Collate", "Column", "Commit", "Conflict", "Constraint", "Create", "Cross", "Current",
	"CurrentDate", "CurrentTime", "CurrentTimestamp", "Database", "Default", "Deferrable", "Deferred", "Delete", "Desc", "Detach",
	"Distinct", "Do", "Drop", "Each", "Else", "End", "Escape", "Except", "Exclude", "Exclusive", "Exists", "Explain", "Fail", "Filter",
	"First", "Following", "For", "Foreign", "From", "Full", "Generated", "Glob", "Group", "Groups", "Having", "If", "Ignore", "Immediate",
	"In", "Index", "Indexed", "Initially", "Inner", "Insert", "Instead", "Intersect", "Into", "Is", "Isnull", "Join", "Key", "Last", "Left",
	"Like", "Limit", "Match", "Materialized", "Natural", "No", "Not", "Nothing", "Notnull", "Null", "Nulls", "Of", "Offset", "On", "Or", "Order",
	"Others", "Outer", "Over", "Partition", "Plan", "Pragma", "Preceding", "Primary", "Query", "Raise", "Range", "Recursive", "References", "Regexp",
	"Reindex", "Release", "Rename", "Replace", "Restrict", "Returning", "Right", "Rollback", "Row", "Rows", "Savepoint", "Select", "Set",
	"Stored", "Table", "Temp", "Temporary", "Then", "Ties", "To", "Transaction", "Trigger", "Unbounded", "Union", "Unique", "Update", "Using",
	"Vacuum", "Values", "View", "Virtual", "When", "Where", "Window", "With", "Without", "Identifier", "String", "Blob", "Numeric", "SQLComment",
	"CComment", "QuestionVariable", "ColonVariable", "AtVariable", "DollarVariable", "Minus", "LeftParen", "RightParen", "Semicolon", "Plus",
	"Asterisk", "Slash", "Percent", "Equal", "EqualEqual", "LessThanOrEqual", "LessThanGreaterThan", "LessThanLessThan", "LessThan", "GreaterThanEqual",
	"GreaterThanGreaterThan", "GreaterThan", "ExclamationEqual", "Comma", "Ampersand", "Tilde", "Pipe", "PipePipe", "Dot", "WhiteSpace", "ErrorUnexpectedEOF",
	"ErrorBlobNotHexadecimal", "ErrorInvalidCharacter", "ErrorInvalidCharacterAfter", "EOF",
}

var (
	// key words
	KindAbort            Kind = kindAbort
	KindAction           Kind = kindAction
	KindAdd              Kind = kindAdd
	KindAfter            Kind = kindAfter
	KindAll              Kind = kindAll
	KindAlter            Kind = kindAlter
	KindAlways           Kind = kindAlways
	KindAnalyze          Kind = kindAnalyze
	KindAnd              Kind = kindAnd
	KindAs               Kind = kindAs
	KindAsc              Kind = kindAsc
	KindAttach           Kind = kindAttach
	KindAutoincrement    Kind = kindAutoincrement
	KindBefore           Kind = kindBefore
	KindBegin            Kind = kindBegin
	KindBetween          Kind = kindBetween
	KindBy               Kind = kindBy
	KindCascade          Kind = kindCascade
	KindCase             Kind = kindCase
	KindCast             Kind = kindCast
	KindCheck            Kind = kindCheck
	KindCollate          Kind = kindCollate
	KindColumn           Kind = kindColumn
	KindCommit           Kind = kindCommit
	KindConflict         Kind = kindConflict
	KindConstraint       Kind = kindConstraint
	KindCreate           Kind = kindCreate
	KindCross            Kind = kindCross
	KindCurrent          Kind = kindCurrent
	KindCurrentDate      Kind = kindCurrentDate
	KindCurrentTime      Kind = kindCurrentTime
	KindCurrentTimestamp Kind = kindCurrentTimestamp
	KindDatabase         Kind = kindDatabase
	KindDefault          Kind = kindDefault
	KindDeferrable       Kind = kindDeferrable
	KindDeferred         Kind = kindDeferred
	KindDelete           Kind = kindDelete
	KindDesc             Kind = kindDesc
	KindDetach           Kind = kindDetach
	KindDistinct         Kind = kindDistinct
	KindDo               Kind = kindDo
	KindDrop             Kind = kindDrop
	KindEach             Kind = kindEach
	KindElse             Kind = kindElse
	KindEnd              Kind = kindEnd
	KindEscape           Kind = kindEscape
	KindExcept           Kind = kindExcept
	KindExclude          Kind = kindExclude
	KindExclusive        Kind = kindExclusive
	KindExists           Kind = kindExists
	KindExplain          Kind = kindExplain
	KindFail             Kind = kindFail
	KindFilter           Kind = kindFilter
	KindFirst            Kind = kindFirst
	KindFollowing        Kind = kindFollowing
	KindFor              Kind = kindFor
	KindForeign          Kind = kindForeign
	KindFrom             Kind = kindFrom
	KindFull             Kind = kindFull
	KindGenerated        Kind = kindGenerated
	KindGlob             Kind = kindGlob
	KindGroup            Kind = kindGroup
	KindGroups           Kind = kindGroups
	KindHaving           Kind = kindHaving
	KindIf               Kind = kindIf
	KindIgnore           Kind = kindIgnore
	KindImmediate        Kind = kindImmediate
	KindIn               Kind = kindIn
	KindIndex            Kind = kindIndex
	KindIndexed          Kind = kindIndexed
	KindInitially        Kind = kindInitially
	KindInner            Kind = kindInner
	KindInsert           Kind = kindInsert
	KindInstead          Kind = kindInstead
	KindIntersect        Kind = kindIntersect
	KindInto             Kind = kindInto
	KindIs               Kind = kindIs
	KindIsnull           Kind = kindIsnull
	KindJoin             Kind = kindJoin
	KindKey              Kind = kindKey
	KindLast             Kind = kindLast
	KindLeft             Kind = kindLeft
	KindLike             Kind = kindLike
	KindLimit            Kind = kindLimit
	KindMatch            Kind = kindMatch
	KindMaterialized     Kind = kindMaterialized
	KindNatural          Kind = kindNatural
	KindNo               Kind = kindNo
	KindNot              Kind = kindNot
	KindNothing          Kind = kindNothing
	KindNotnull          Kind = kindNotnull
	KindNull             Kind = kindNull
	KindNulls            Kind = kindNulls
	KindOf               Kind = kindOf
	KindOffset           Kind = kindOffset
	KindOn               Kind = kindOn
	KindOr               Kind = kindOr
	KindOrder            Kind = kindOrder
	KindOthers           Kind = kindOthers
	KindOuter            Kind = kindOuter
	KindOver             Kind = kindOver
	KindPartition        Kind = kindPartition
	KindPlan             Kind = kindPlan
	KindPragma           Kind = kindPragma
	KindPreceding        Kind = kindPreceding
	KindPrimary          Kind = kindPrimary
	KindQuery            Kind = kindQuery
	KindRaise            Kind = kindRaise
	KindRange            Kind = kindRange
	KindRecursive        Kind = kindRecursive
	KindReferences       Kind = kindReferences
	KindRegexp           Kind = kindRegexp
	KindReindex          Kind = kindReindex
	KindRelease          Kind = kindRelease
	KindRename           Kind = kindRename
	KindReplace          Kind = kindReplace
	KindRestrict         Kind = kindRestrict
	KindReturning        Kind = kindReturning
	KindRight            Kind = kindRight
	KindRollback         Kind = kindRollback
	KindRow              Kind = kindRow
	KindRows             Kind = kindRows
	KindSavepoint        Kind = kindSavepoint
	KindSelect           Kind = kindSelect
	KindSet              Kind = kindSet
	KindTable            Kind = kindTable
	KindTemp             Kind = kindTemp
	KindTemporary        Kind = kindTemporary
	KindThen             Kind = kindThen
	KindTies             Kind = kindTies
	KindTo               Kind = kindTo
	KindTransaction      Kind = kindTransaction
	KindTrigger          Kind = kindTrigger
	KindUnbounded        Kind = kindUnbounded
	KindUnion            Kind = kindUnion
	KindUnique           Kind = kindUnique
	KindUpdate           Kind = kindUpdate
	KindUsing            Kind = kindUsing
	KindVacuum           Kind = kindVacuum
	KindValues           Kind = kindValues
	KindView             Kind = kindView
	KindVirtual          Kind = kindVirtual
	KindWhen             Kind = kindWhen
	KindWhere            Kind = kindWhere
	KindWindow           Kind = kindWindow
	KindWith             Kind = kindWith
	KindWithout          Kind = kindWithout
	// end of keywords
	KindIdentifier                 Kind = kindIdentifier
	KindString                     Kind = kindString
	KindBlob                       Kind = kindBlob
	KindNumeric                    Kind = kindNumeric
	KindSQLComment                 Kind = kindSQLComment
	KindCComment                   Kind = kindCComment
	KindQuestionVariable           Kind = kindQuestionVariable
	KindColonVariable              Kind = kindColonVariable
	KindAtVariable                 Kind = kindAtVariable
	KindDollarVariable             Kind = kindDollarVariable
	KindMinus                      Kind = kindMinus
	KindLeftParen                  Kind = kindLeftParen
	KindRightParen                 Kind = kindRightParen
	KindSemicolon                  Kind = kindSemicolon
	KindPlus                       Kind = kindPlus
	KindAsterisk                   Kind = kindAsterisk
	KindSlash                      Kind = kindSlash
	KindPercent                    Kind = kindPercent
	KindEqual                      Kind = kindEqual
	KindEqualEqual                 Kind = kindEqualEqual
	KindLessThanOrEqual            Kind = kindLessThanOrEqual
	KindLessThanGreaterThan        Kind = kindLessThanGreaterThan
	KindLessThanLessThan           Kind = kindLessThanLessThan
	KindLessThan                   Kind = kindLessThan
	KindGreaterThanEqual           Kind = kindGreaterThanEqual
	KindGreaterThanGreaterThan     Kind = kindGreaterThanGreaterThan
	KindGreaterThan                Kind = kindGreaterThan
	KindExclamationEqual           Kind = kindExclamationEqual
	KindComma                      Kind = kindComma
	KindAmpersand                  Kind = kindAmpersand
	KindTilde                      Kind = kindTilde
	KindPipe                       Kind = kindPipe
	KindPipePipe                   Kind = kindPipePipe
	KindDot                        Kind = kindDot
	KindWhiteSpace                 Kind = kindWhiteSpace
	KindErrorUnexpectedEOF         Kind = kindErrorUnexpectedEOF
	KindErrorBlobNotHexadecimal    Kind = kindErrorBlobNotHexadecimal
	KindErrorInvalidCharacter      Kind = kindErrorInvalidCharacter
	KindErrorInvalidCharacterAfter Kind = kindErrorInvalidCharacterAfter
	KindEOF                        Kind = kindEOF
)
