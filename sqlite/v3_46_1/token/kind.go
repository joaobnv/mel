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

var (
	// key words
	kindAbort            kind = 0
	kindAction           kind = 1
	kindAdd              kind = 2
	kindAfter            kind = 3
	kindAll              kind = 4
	kindAlter            kind = 5
	kindAlways           kind = 6
	kindAnalyze          kind = 7
	kindAnd              kind = 8
	kindAs               kind = 9
	kindAsc              kind = 10
	kindAttach           kind = 11
	kindAutoincrement    kind = 12
	kindBefore           kind = 13
	kindBegin            kind = 14
	kindBetween          kind = 15
	kindBy               kind = 16
	kindCascade          kind = 17
	kindCase             kind = 18
	kindCast             kind = 19
	kindCheck            kind = 20
	kindCollate          kind = 21
	kindColumn           kind = 22
	kindCommit           kind = 23
	kindConflict         kind = 24
	kindConstraint       kind = 25
	kindCreate           kind = 26
	kindCross            kind = 27
	kindCurrent          kind = 28
	kindCurrentDate      kind = 29
	kindCurrentTime      kind = 30
	kindCurrentTimestamp kind = 31
	kindDatabase         kind = 32
	kindDefault          kind = 33
	kindDeferrable       kind = 34
	kindDeferred         kind = 35
	kindDelete           kind = 36
	kindDesc             kind = 37
	kindDetach           kind = 38
	kindDistinct         kind = 39
	kindDo               kind = 40
	kindDrop             kind = 41
	kindEach             kind = 42
	kindElse             kind = 43
	kindEnd              kind = 44
	kindEscape           kind = 45
	kindExcept           kind = 46
	kindExclude          kind = 47
	kindExclusive        kind = 48
	kindExists           kind = 49
	kindExplain          kind = 50
	kindFail             kind = 51
	kindFilter           kind = 52
	kindFirst            kind = 53
	kindFollowing        kind = 54
	kindFor              kind = 55
	kindForeign          kind = 56
	kindFrom             kind = 57
	kindFull             kind = 58
	kindGenerated        kind = 59
	kindGlob             kind = 60
	kindGroup            kind = 61
	kindGroups           kind = 62
	kindHaving           kind = 63
	kindIf               kind = 64
	kindIgnore           kind = 65
	kindImmediate        kind = 66
	kindIn               kind = 67
	kindIndex            kind = 68
	kindIndexed          kind = 69
	kindInitially        kind = 70
	kindInner            kind = 71
	kindInsert           kind = 72
	kindInstead          kind = 73
	kindIntersect        kind = 74
	kindInto             kind = 75
	kindIs               kind = 76
	kindIsnull           kind = 77
	kindJoin             kind = 78
	kindKey              kind = 79
	kindLast             kind = 80
	kindLeft             kind = 81
	kindLike             kind = 82
	kindLimit            kind = 83
	kindMatch            kind = 84
	kindMaterialized     kind = 85
	kindNatural          kind = 86
	kindNo               kind = 87
	kindNot              kind = 88
	kindNothing          kind = 89
	kindNotnull          kind = 90
	kindNull             kind = 91
	kindNulls            kind = 92
	kindOf               kind = 93
	kindOffset           kind = 94
	kindOn               kind = 95
	kindOr               kind = 96
	kindOrder            kind = 97
	kindOthers           kind = 98
	kindOuter            kind = 99
	kindOver             kind = 100
	kindPartition        kind = 101
	kindPlan             kind = 102
	kindPragma           kind = 103
	kindPreceding        kind = 104
	kindPrimary          kind = 105
	kindQuery            kind = 106
	kindRaise            kind = 107
	kindRange            kind = 108
	kindRecursive        kind = 109
	kindReferences       kind = 110
	kindRegexp           kind = 111
	kindReindex          kind = 112
	kindRelease          kind = 113
	kindRename           kind = 114
	kindReplace          kind = 115
	kindRestrict         kind = 116
	kindReturning        kind = 117
	kindRight            kind = 118
	kindRollback         kind = 119
	kindRow              kind = 120
	kindRows             kind = 121
	kindSavepoint        kind = 122
	kindSelect           kind = 123
	kindSet              kind = 124
	kindTable            kind = 125
	kindTemp             kind = 126
	kindTemporary        kind = 127
	kindThen             kind = 128
	kindTies             kind = 129
	kindTo               kind = 130
	kindTransaction      kind = 131
	kindTrigger          kind = 132
	kindUnbounded        kind = 133
	kindUnion            kind = 134
	kindUnique           kind = 135
	kindUpdate           kind = 136
	kindUsing            kind = 137
	kindVacuum           kind = 138
	kindValues           kind = 139
	kindView             kind = 140
	kindVirtual          kind = 141
	kindWhen             kind = 142
	kindWhere            kind = 143
	kindWindow           kind = 144
	kindWith             kind = 145
	kindWithout          kind = 146
	// kind =  end of keywords
	kindIdentifier                 kind = 147
	kindString                     kind = 148
	kindBlob                       kind = 149
	kindNumeric                    kind = 150
	kindSQLComment                 kind = 151
	kindCComment                   kind = 152
	kindQuestionVariable           kind = 153
	kindColonVariable              kind = 154
	kindAtVariable                 kind = 155
	kindDollarVariable             kind = 156
	kindMinus                      kind = 157
	kindLeftParen                  kind = 158
	kindRightParen                 kind = 159
	kindSemicolon                  kind = 160
	kindPlus                       kind = 161
	kindAsterisk                   kind = 162
	kindSlash                      kind = 163
	kindPercent                    kind = 164
	kindEqual                      kind = 165
	kindEqualEqual                 kind = 166
	kindLessThanOrEqual            kind = 167
	kindLessThanGreaterThan        kind = 168
	kindLessThanLessThan           kind = 169
	kindLessThan                   kind = 170
	kindGreaterThanEqual           kind = 171
	kindGreaterThanGreaterThan     kind = 172
	kindGreaterThan                kind = 173
	kindExclamationEqual           kind = 174
	kindComma                      kind = 175
	kindAmpersand                  kind = 176
	kindTilde                      kind = 177
	kindPipe                       kind = 178
	kindPipePipe                   kind = 179
	kindDot                        kind = 180
	kindWhiteSpace                 kind = 181
	kindErrorUnexpectedEOF         kind = 182
	kindErrorBlobNotHexadecimal    kind = 183
	kindErrorInvalidCharacter      kind = 184
	kindErrorInvalidCharacterAfter kind = 185
	kindEOF                        kind = 186
)

// String returns a string representation of k.
func (k *kind) String() string {
	if *k < 0 || int(*k) >= len(kindStrings) {
		return strconv.Itoa(int(*k))
	}
	return kindStrings[*k]
}

// kindStrings contains the string representation of the kinds. Note that the value of a kind is the index of your string
// representation.
var kindStrings = []string{
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
	"QuestionVariable", "ColonVariable", "AtVariable", "DollarVariable", "Minus", "LeftParen", "RightParen", "Semicolon",
	"Plus", "Asterisk", "Slash", "Percent", "Equal", "EqualEqual", "LessThanOrEqual", "LessThanGreaterThan", "LessThanLessThan",
	"LessThan", "GreaterThanEqual", "GreaterThanGreaterThan", "GreaterThan", "ExclamationEqual", "Comma", "Ampersand",
	"Tilde", "Pipe", "PipePipe", "Dot", "WhiteSpace", "ErrorUnexpectedEOF", "ErrorBlobNotHexadecimal", "ErrorInvalidCharacter",
	"ErrorInvalidCharacterAfter", "EOF",
}

var (
	// key words
	KindAbort            Kind = &kindAbort
	KindAction           Kind = &kindAction
	KindAdd              Kind = &kindAdd
	KindAfter            Kind = &kindAfter
	KindAll              Kind = &kindAll
	KindAlter            Kind = &kindAlter
	KindAlways           Kind = &kindAlways
	KindAnalyze          Kind = &kindAnalyze
	KindAnd              Kind = &kindAnd
	KindAs               Kind = &kindAs
	KindAsc              Kind = &kindAsc
	KindAttach           Kind = &kindAttach
	KindAutoincrement    Kind = &kindAutoincrement
	KindBefore           Kind = &kindBefore
	KindBegin            Kind = &kindBegin
	KindBetween          Kind = &kindBetween
	KindBy               Kind = &kindBy
	KindCascade          Kind = &kindCascade
	KindCase             Kind = &kindCase
	KindCast             Kind = &kindCast
	KindCheck            Kind = &kindCheck
	KindCollate          Kind = &kindCollate
	KindColumn           Kind = &kindColumn
	KindCommit           Kind = &kindCommit
	KindConflict         Kind = &kindConflict
	KindConstraint       Kind = &kindConstraint
	KindCreate           Kind = &kindCreate
	KindCross            Kind = &kindCross
	KindCurrent          Kind = &kindCurrent
	KindCurrentDate      Kind = &kindCurrentDate
	KindCurrentTime      Kind = &kindCurrentTime
	KindCurrentTimestamp Kind = &kindCurrentTimestamp
	KindDatabase         Kind = &kindDatabase
	KindDefault          Kind = &kindDefault
	KindDeferrable       Kind = &kindDeferrable
	KindDeferred         Kind = &kindDeferred
	KindDelete           Kind = &kindDelete
	KindDesc             Kind = &kindDesc
	KindDetach           Kind = &kindDetach
	KindDistinct         Kind = &kindDistinct
	KindDo               Kind = &kindDo
	KindDrop             Kind = &kindDrop
	KindEach             Kind = &kindEach
	KindElse             Kind = &kindElse
	KindEnd              Kind = &kindEnd
	KindEscape           Kind = &kindEscape
	KindExcept           Kind = &kindExcept
	KindExclude          Kind = &kindExclude
	KindExclusive        Kind = &kindExclusive
	KindExists           Kind = &kindExists
	KindExplain          Kind = &kindExplain
	KindFail             Kind = &kindFail
	KindFilter           Kind = &kindFilter
	KindFirst            Kind = &kindFirst
	KindFollowing        Kind = &kindFollowing
	KindFor              Kind = &kindFor
	KindForeign          Kind = &kindForeign
	KindFrom             Kind = &kindFrom
	KindFull             Kind = &kindFull
	KindGenerated        Kind = &kindGenerated
	KindGlob             Kind = &kindGlob
	KindGroup            Kind = &kindGroup
	KindGroups           Kind = &kindGroups
	KindHaving           Kind = &kindHaving
	KindIf               Kind = &kindIf
	KindIgnore           Kind = &kindIgnore
	KindImmediate        Kind = &kindImmediate
	KindIn               Kind = &kindIn
	KindIndex            Kind = &kindIndex
	KindIndexed          Kind = &kindIndexed
	KindInitially        Kind = &kindInitially
	KindInner            Kind = &kindInner
	KindInsert           Kind = &kindInsert
	KindInstead          Kind = &kindInstead
	KindIntersect        Kind = &kindIntersect
	KindInto             Kind = &kindInto
	KindIs               Kind = &kindIs
	KindIsnull           Kind = &kindIsnull
	KindJoin             Kind = &kindJoin
	KindKey              Kind = &kindKey
	KindLast             Kind = &kindLast
	KindLeft             Kind = &kindLeft
	KindLike             Kind = &kindLike
	KindLimit            Kind = &kindLimit
	KindMatch            Kind = &kindMatch
	KindMaterialized     Kind = &kindMaterialized
	KindNatural          Kind = &kindNatural
	KindNo               Kind = &kindNo
	KindNot              Kind = &kindNot
	KindNothing          Kind = &kindNothing
	KindNotnull          Kind = &kindNotnull
	KindNull             Kind = &kindNull
	KindNulls            Kind = &kindNulls
	KindOf               Kind = &kindOf
	KindOffset           Kind = &kindOffset
	KindOn               Kind = &kindOn
	KindOr               Kind = &kindOr
	KindOrder            Kind = &kindOrder
	KindOthers           Kind = &kindOthers
	KindOuter            Kind = &kindOuter
	KindOver             Kind = &kindOver
	KindPartition        Kind = &kindPartition
	KindPlan             Kind = &kindPlan
	KindPragma           Kind = &kindPragma
	KindPreceding        Kind = &kindPreceding
	KindPrimary          Kind = &kindPrimary
	KindQuery            Kind = &kindQuery
	KindRaise            Kind = &kindRaise
	KindRange            Kind = &kindRange
	KindRecursive        Kind = &kindRecursive
	KindReferences       Kind = &kindReferences
	KindRegexp           Kind = &kindRegexp
	KindReindex          Kind = &kindReindex
	KindRelease          Kind = &kindRelease
	KindRename           Kind = &kindRename
	KindReplace          Kind = &kindReplace
	KindRestrict         Kind = &kindRestrict
	KindReturning        Kind = &kindReturning
	KindRight            Kind = &kindRight
	KindRollback         Kind = &kindRollback
	KindRow              Kind = &kindRow
	KindRows             Kind = &kindRows
	KindSavepoint        Kind = &kindSavepoint
	KindSelect           Kind = &kindSelect
	KindSet              Kind = &kindSet
	KindTable            Kind = &kindTable
	KindTemp             Kind = &kindTemp
	KindTemporary        Kind = &kindTemporary
	KindThen             Kind = &kindThen
	KindTies             Kind = &kindTies
	KindTo               Kind = &kindTo
	KindTransaction      Kind = &kindTransaction
	KindTrigger          Kind = &kindTrigger
	KindUnbounded        Kind = &kindUnbounded
	KindUnion            Kind = &kindUnion
	KindUnique           Kind = &kindUnique
	KindUpdate           Kind = &kindUpdate
	KindUsing            Kind = &kindUsing
	KindVacuum           Kind = &kindVacuum
	KindValues           Kind = &kindValues
	KindView             Kind = &kindView
	KindVirtual          Kind = &kindVirtual
	KindWhen             Kind = &kindWhen
	KindWhere            Kind = &kindWhere
	KindWindow           Kind = &kindWindow
	KindWith             Kind = &kindWith
	KindWithout          Kind = &kindWithout
	// end of keywords
	KindIdentifier                 Kind = &kindIdentifier
	KindString                     Kind = &kindString
	KindBlob                       Kind = &kindBlob
	KindNumeric                    Kind = &kindNumeric
	KindSQLComment                 Kind = &kindSQLComment
	KindCComment                   Kind = &kindCComment
	KindQuestionVariable           Kind = &kindQuestionVariable
	KindColonVariable              Kind = &kindColonVariable
	KindAtVariable                 Kind = &kindAtVariable
	KindDollarVariable             Kind = &kindDollarVariable
	KindMinus                      Kind = &kindMinus
	KindLeftParen                  Kind = &kindLeftParen
	KindRightParen                 Kind = &kindRightParen
	KindSemicolon                  Kind = &kindSemicolon
	KindPlus                       Kind = &kindPlus
	KindAsterisk                   Kind = &kindAsterisk
	KindSlash                      Kind = &kindSlash
	KindPercent                    Kind = &kindPercent
	KindEqual                      Kind = &kindEqual
	KindEqualEqual                 Kind = &kindEqualEqual
	KindLessThanOrEqual            Kind = &kindLessThanOrEqual
	KindLessThanGreaterThan        Kind = &kindLessThanGreaterThan
	KindLessThanLessThan           Kind = &kindLessThanLessThan
	KindLessThan                   Kind = &kindLessThan
	KindGreaterThanEqual           Kind = &kindGreaterThanEqual
	KindGreaterThanGreaterThan     Kind = &kindGreaterThanGreaterThan
	KindGreaterThan                Kind = &kindGreaterThan
	KindExclamationEqual           Kind = &kindExclamationEqual
	KindComma                      Kind = &kindComma
	KindAmpersand                  Kind = &kindAmpersand
	KindTilde                      Kind = &kindTilde
	KindPipe                       Kind = &kindPipe
	KindPipePipe                   Kind = &kindPipePipe
	KindDot                        Kind = &kindDot
	KindWhiteSpace                 Kind = &kindWhiteSpace
	KindErrorUnexpectedEOF         Kind = &kindErrorUnexpectedEOF
	KindErrorBlobNotHexadecimal    Kind = &kindErrorBlobNotHexadecimal
	KindErrorInvalidCharacter      Kind = &kindErrorInvalidCharacter
	KindErrorInvalidCharacterAfter Kind = &kindErrorInvalidCharacterAfter
	KindEOF                        Kind = &kindEOF
)
