// package lexer deals with the lexical scanning.
package lexer

import (
	"mel/sqlite/v3_46_1/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// keywords is a map from the keywords of the language to the kind of the keyword token.
var keywords = map[string]token.Kind{
	"ABORT":             token.KindAbort,
	"ACTION":            token.KindAction,
	"ADD":               token.KindAdd,
	"AFTER":             token.KindAfter,
	"ALL":               token.KindAll,
	"ALTER":             token.KindAlter,
	"ALWAYS":            token.KindAlways,
	"ANALYZE":           token.KindAnalyze,
	"AND":               token.KindAnd,
	"AS":                token.KindAs,
	"ASC":               token.KindAsc,
	"ATTACH":            token.KindAttach,
	"AUTOINCREMENT":     token.KindAutoincrement,
	"BEFORE":            token.KindBefore,
	"BEGIN":             token.KindBegin,
	"BETWEEN":           token.KindBetween,
	"BY":                token.KindBy,
	"CASCADE":           token.KindCascade,
	"CASE":              token.KindCase,
	"CAST":              token.KindCast,
	"CHECK":             token.KindCheck,
	"COLLATE":           token.KindCollate,
	"COLUMN":            token.KindColumn,
	"COMMIT":            token.KindCommit,
	"CONFLICT":          token.KindConflict,
	"CONSTRAINT":        token.KindConstraint,
	"CREATE":            token.KindCreate,
	"CROSS":             token.KindCross,
	"CURRENT":           token.KindCurrent,
	"CURRENT_DATE":      token.KindCurrentDate,
	"CURRENT_TIME":      token.KindCurrentTime,
	"CURRENT_TIMESTAMP": token.KindCurrentTimestamp,
	"DATABASE":          token.KindDatabase,
	"DEFAULT":           token.KindDefault,
	"DEFERRABLE":        token.KindDeferrable,
	"DEFERRED":          token.KindDeferred,
	"DELETE":            token.KindDelete,
	"DESC":              token.KindDesc,
	"DETACH":            token.KindDetach,
	"DISTINCT":          token.KindDistinct,
	"DO":                token.KindDo,
	"DROP":              token.KindDrop,
	"EACH":              token.KindEach,
	"ELSE":              token.KindElse,
	"END":               token.KindEnd,
	"ESCAPE":            token.KindEscape,
	"EXCEPT":            token.KindExcept,
	"EXCLUDE":           token.KindExclude,
	"EXCLUSIVE":         token.KindExclusive,
	"EXISTS":            token.KindExists,
	"EXPLAIN":           token.KindExplain,
	"FAIL":              token.KindFail,
	"FILTER":            token.KindFilter,
	"FIRST":             token.KindFirst,
	"FOLLOWING":         token.KindFollowing,
	"FOR":               token.KindFor,
	"FOREIGN":           token.KindForeign,
	"FROM":              token.KindFrom,
	"FULL":              token.KindFull,
	"GENERATED":         token.KindGenerated,
	"GLOB":              token.KindGlob,
	"GROUP":             token.KindGroup,
	"GROUPS":            token.KindGroups,
	"HAVING":            token.KindHaving,
	"IF":                token.KindIf,
	"IGNORE":            token.KindIgnore,
	"IMMEDIATE":         token.KindImmediate,
	"IN":                token.KindIn,
	"INDEX":             token.KindIndex,
	"INDEXED":           token.KindIndexed,
	"INITIALLY":         token.KindInitially,
	"INNER":             token.KindInner,
	"INSERT":            token.KindInsert,
	"INSTEAD":           token.KindInstead,
	"INTERSECT":         token.KindIntersect,
	"INTO":              token.KindInto,
	"IS":                token.KindIs,
	"ISNULL":            token.KindIsnull,
	"JOIN":              token.KindJoin,
	"KEY":               token.KindKey,
	"LAST":              token.KindLast,
	"LEFT":              token.KindLeft,
	"LIKE":              token.KindLike,
	"LIMIT":             token.KindLimit,
	"MATCH":             token.KindMatch,
	"MATERIALIZED":      token.KindMaterialized,
	"NATURAL":           token.KindNatural,
	"NO":                token.KindNo,
	"NOT":               token.KindNot,
	"NOTHING":           token.KindNothing,
	"NOTNULL":           token.KindNotnull,
	"NULL":              token.KindNull,
	"NULLS":             token.KindNulls,
	"OF":                token.KindOf,
	"OFFSET":            token.KindOffset,
	"ON":                token.KindOn,
	"OR":                token.KindOr,
	"ORDER":             token.KindOrder,
	"OTHERS":            token.KindOthers,
	"OUTER":             token.KindOuter,
	"OVER":              token.KindOver,
	"PARTITION":         token.KindPartition,
	"PLAN":              token.KindPlan,
	"PRAGMA":            token.KindPragma,
	"PRECEDING":         token.KindPreceding,
	"PRIMARY":           token.KindPrimary,
	"QUERY":             token.KindQuery,
	"RAISE":             token.KindRaise,
	"RANGE":             token.KindRange,
	"RECURSIVE":         token.KindRecursive,
	"REFERENCES":        token.KindReferences,
	"REGEXP":            token.KindRegexp,
	"REINDEX":           token.KindReindex,
	"RELEASE":           token.KindRelease,
	"RENAME":            token.KindRename,
	"REPLACE":           token.KindReplace,
	"RESTRICT":          token.KindRestrict,
	"RETURNING":         token.KindReturning,
	"RIGHT":             token.KindRight,
	"ROLLBACK":          token.KindRollback,
	"ROW":               token.KindRow,
	"ROWS":              token.KindRows,
	"SAVEPOINT":         token.KindSavepoint,
	"SELECT":            token.KindSelect,
	"SET":               token.KindSet,
	"TABLE":             token.KindTable,
	"TEMP":              token.KindTemp,
	"TEMPORARY":         token.KindTemporary,
	"THEN":              token.KindThen,
	"TIES":              token.KindTies,
	"TO":                token.KindTo,
	"TRANSACTION":       token.KindTransaction,
	"TRIGGER":           token.KindTrigger,
	"UNBOUNDED":         token.KindUnbounded,
	"UNION":             token.KindUnion,
	"UNIQUE":            token.KindUnique,
	"UPDATE":            token.KindUpdate,
	"USING":             token.KindUsing,
	"VACUUM":            token.KindVacuum,
	"VALUES":            token.KindValues,
	"VIEW":              token.KindView,
	"VIRTUAL":           token.KindVirtual,
	"WHEN":              token.KindWhen,
	"WHERE":             token.KindWhere,
	"WINDOW":            token.KindWindow,
	"WITH":              token.KindWith,
	"WITHOUT":           token.KindWithout,
}

// Lexer is a lexical scanner
type Lexer struct {
	// r is the reader that the lexer uses for reading the runes from the code.
	r *reader
}

// New creates a new Lexer that reads from code.
func New(code []byte) *Lexer {
	return &Lexer{r: newReader(code)}
}

// discardWhiteSpace reads and discards white space.
func (l *Lexer) discardWhiteSpace() {
	var (
		eof bool
		r   rune
	)
	for r, eof = l.r.readRune(); !eof && unicode.IsSpace(r); r, eof = l.r.readRune() {
	}
	if !eof {
		l.r.unreadRune()
	}
}

// word scans a keyword or a identifier.
func (l *Lexer) word() *token.Token {
	offsetStart := l.r.getOffset()
	r, _ := l.r.readRune()
	if unicode.IsLetter(r) {
		var eof bool
		for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
			if !unicode.IsLetter(r) && r != '_' && !strings.ContainsRune("0123456789", r) {
				break
			}
		}
		if !eof {
			l.r.unreadRune()
		}
		lexeme := l.r.slice(offsetStart, l.r.getOffset())
		if kind, isKeyword := keywords[strings.ToUpper(string(lexeme))]; isKeyword {
			return token.New(lexeme, kind)
		}
		return token.New(lexeme, token.KindIdentifier)
	} else if r == '"' {
		var eof bool
		for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
			if r == '"' {
				break
			}
		}
		lexeme := l.r.slice(offsetStart, l.r.getOffset())
		if eof {
			err := string(lexeme) + ": unexpected EOF"
			return token.New([]byte(err), token.KindError)
		}
		return token.New(lexeme, token.KindIdentifier)
	} else if r == '[' {
		var eof bool
		for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
			if r == ']' {
				break
			}
		}
		lexeme := l.r.slice(offsetStart, l.r.getOffset())
		if eof {
			err := string(lexeme) + ": unexpected EOF"
			return token.New([]byte(err), token.KindError)
		}
		return token.New(lexeme, token.KindIdentifier)
	} else { // r == '`'
		var eof bool
		for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
			if r == '`' {
				break
			}
		}
		lexeme := l.r.slice(offsetStart, l.r.getOffset())
		if eof {
			err := string(lexeme) + ": unexpected EOF"
			return token.New([]byte(err), token.KindError)
		}
		return token.New(lexeme, token.KindIdentifier)
	}
}

// reader reads from the code.
type reader struct {
	// code is the code to be read.
	code []byte
	// offset is the current offset on code.
	offset int64
}

// newReader creates a new reader that reads from code.
func newReader(code []byte) *reader {
	return &reader{
		code: code,
	}
}

// readRune reads the next rune from the code. It panics on error.
func (r *reader) readRune() (rn rune, eof bool) {
	rn, size := utf8.DecodeRune(r.code[r.offset:])
	if rn == utf8.RuneError {
		if size == 0 {
			return 0, true
		} else {
			panic("utf-8 encoding invalid")
		}
	}
	r.offset += int64(size)
	return
}

// unreadRune seek to the start of the rune before the current offset. If the current or resulting offset is at
// the start of the code then onStart will be true.
func (r *reader) unreadRune() (onStart bool) {
	if r.offset == 0 {
		return true
	}
	for i := r.offset - 1; i >= 0; i-- {
		if utf8.RuneStart(r.code[i]) {
			r.offset = i
			return r.offset == 0
		}
	}
	panic("utf-8 encoding invalid")
}

// getOffset returns the current offset.
func (r *reader) getOffset() int64 {
	return r.offset
}

// slice returns r.code[offsetStart:offsetEnd].
func (r *reader) slice(offsetStart, offsetEnd int64) []byte {
	return r.code[offsetStart:offsetEnd]
}
