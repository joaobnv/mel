// This package deals with the lexical scanning.
package lexer

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
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

// Next returns the next token.
func (l *Lexer) Next() *token.Token {
	l.discardWhiteSpace()
	rs, _ := l.r.peekNRunes(2)
	if len(rs) == 0 {
		return token.New(nil, token.KindEOF)
	}

	if len(rs) == 2 && (rs[0] == 'x' || rs[0] == 'X') && rs[1] == '\'' {
		return l.blob()
	} else if unicode.IsLetter(rs[0]) || strings.ContainsRune("_`\"[", rs[0]) {
		return l.word()
	} else if rs[0] == '\'' {
		return l.string()
	} else if l.isNumeric(rs[0]) {
		return l.numeric()
	} else if len(rs) == 2 && rs[0] == '.' && l.isNumeric(rs[1]) {
		return l.numeric()
	} else if len(rs) == 2 && rs[0] == '-' && rs[1] == '-' {
		return l.sqlComment()
	} else if len(rs) == 2 && rs[0] == '/' && rs[1] == '*' {
		return l.cComment()
	} else if rs[0] == '?' {
		return l.questionVariable()
	} else if rs[0] == ':' {
		return l.colonVariable()
	} else if rs[0] == '@' {
		return l.atVariable()
	} else if strings.ContainsRune("-();+*/%=<>!,&~|.", rs[0]) {
		return l.operator()
	} else {
		panic("not implemented yet")
	}
}

// discardWhiteSpace reads and discards white space.
func (l *Lexer) discardWhiteSpace() {
	var (
		eof bool
		r   rune
	)
	for r, eof = l.r.readRune(); !eof && l.isWhiteSpace(r); r, eof = l.r.readRune() {
	}
	if !eof {
		l.r.unreadRune()
	}
}

// blob scans a blob.
func (l *Lexer) blob() *token.Token {
	var (
		eof bool
		r   rune
	)
	offsetStart := l.r.getOffset()
	l.r.readRune()
	l.r.readRune()
	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if r == '\'' {
			break
		}
		if !l.isHexadecimal(r) {
			lexeme := l.r.slice(offsetStart, l.r.getOffset())
			err := string(lexeme) + ": a blob must contain only hexadecimal characters"
			return token.New([]byte(err), token.KindError)
		}
	}
	lexeme := l.r.slice(offsetStart, l.r.getOffset())
	if eof {
		err := string(lexeme) + ": unexpected EOF"
		return token.New([]byte(err), token.KindError)
	}
	return token.New(lexeme, token.KindBlob)
}

// word scans a keyword or an identifier.
func (l *Lexer) word() *token.Token {
	offsetStart := l.r.getOffset()
	r, _ := l.r.readRune()
	if l.isAlphabetic(r) {
		var eof bool
		for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
			if !l.isAlphanumeric(r) && r != '$' {
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
				if pr, peof := l.r.peekRune(); peof || pr != '"' {
					break
				}
				l.r.readRune()
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
				if pr, peof := l.r.peekRune(); peof || pr != '`' {
					break
				}
				l.r.readRune()
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

// string scans a string.
func (l *Lexer) string() *token.Token {
	var (
		eof bool
		r   rune
	)
	offsetStart := l.r.getOffset()
	l.r.readRune()
	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if r == '\'' {
			if pr, peof := l.r.peekRune(); peof || pr != '\'' {
				break
			}
			l.r.readRune()
		}
	}
	lexeme := l.r.slice(offsetStart, l.r.getOffset())
	if eof {
		err := string(lexeme) + ": unexpected EOF"
		return token.New([]byte(err), token.KindError)
	}
	return token.New(lexeme, token.KindString)
}

// numeric scans a numeric literal.
func (l *Lexer) numeric() *token.Token {
	offsetStart := l.r.getOffset()
	rs, _ := l.r.peekNRunes(2)
	if (l.isNumeric(rs[0]) && rs[0] != '0') || (l.isNumeric(rs[0]) && rs[1] != 'x' && rs[1] != 'X') {
		if l.numericDigits() {
			return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
		}

		r, _ := l.r.peekRune()
		if r == '.' {
			l.r.readRune()
			r, eof := l.r.peekRune()
			if eof || !l.isNumeric(r) {
				return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
			}
			if l.numericDigits() {
				return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
			}
		}

		l.numericExponentialPart()
		return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
	} else if rs[0] == '.' {
		l.r.readRune()
		if l.numericDigits() {
			return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
		}

		l.numericExponentialPart()
		return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
	} else { // len(rs) == 2 && rs[0] == '0' && (rs[1] == 'x' || rs[1] == 'X') {
		l.r.readRune()
		l.r.readRune()
		l.numericHexDigits()
		return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindNumeric)
	}
}

// numericDigits scans the part of a numeric literal where digits and underscores can apear.
func (l *Lexer) numericDigits() (eof bool) {
	var r rune
	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if r == '_' {
			pr, peof := l.r.peekRune()
			if peof || !l.isNumeric(pr) {
				l.r.unreadRune()
				return false
			}
		} else if !l.isNumeric(r) {
			l.r.unreadRune()
			break
		}
	}
	return
}

// numericHexDigits scans the part of a hexadecimal numeric literal where hexadecimal digits and underscores can apear.
func (l *Lexer) numericHexDigits() (eof bool) {
	var r rune
	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if r == '_' {
			pr, peof := l.r.peekRune()
			if peof || !l.isHexadecimal(pr) {
				l.r.unreadRune()
				return false
			}
		} else if !l.isHexadecimal(r) {
			l.r.unreadRune()
			break
		}
	}
	return
}

// numericExponentialPart scans the exponential part of a numeric literal. It doesn't expect the next runes to be of an
// exponential part of a numeric literal. If they are not then match will be false and the reader will not be advanced.
func (l *Lexer) numericExponentialPart() (match bool, eof bool) {
	if match = l.isStartOfNumericExponentialPart(); !match {
		return
	}
	l.r.readRune()
	r, _ := l.r.peekRune()
	if r == '-' || r == '+' {
		l.r.readRune()
	}
	eof = l.numericDigits()
	return
}

// isStartOfNumericExponentialPart reports if the next runes are the start of the exponential part of a numeric literal.
// It dont advances the reader.
func (l *Lexer) isStartOfNumericExponentialPart() (is bool) {
	rs, _ := l.r.peekNRunes(3)
	if len(rs) < 2 || rs[0] != 'e' && rs[0] != 'E' {
		return
	}
	if !l.isNumeric(rs[1]) && rs[1] != '-' && rs[1] != '+' {
		return
	}
	var withSign bool
	if rs[1] == '+' || rs[1] == '-' {
		withSign = true
	}
	if withSign && len(rs) < 3 {
		return
	}
	if withSign && !l.isNumeric(rs[2]) {
		return
	}
	return true
}

// sqlComment scans a SQL-style comment.
func (l *Lexer) sqlComment() *token.Token {
	offsetStart := l.r.getOffset()
	l.r.readRune()
	l.r.readRune()
	for r, eof := l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if r == '\n' {
			l.r.unreadRune()
			break
		}
	}
	return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindSQLComment)
}

// cComment scans a C-style comment.
func (l *Lexer) cComment() *token.Token {
	offsetStart := l.r.getOffset()
	l.r.readRune()
	l.r.readRune()
	for r, eof := l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if r != '*' {
			continue
		}

		pr, peof := l.r.peekRune()
		if peof {
			break
		}
		if pr == '/' {
			l.r.readRune()
			break
		}
	}
	return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindCComment)
}

// questionVariable scans a question variable.
func (l *Lexer) questionVariable() *token.Token {
	var r rune
	var eof bool
	offsetStart := l.r.getOffset()
	l.r.readRune()
	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if !l.isNumeric(r) {
			break
		}
	}
	if !eof {
		l.r.unreadRune()
	}
	return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindQuestionVariable)
}

// colonVariable scans a colon variable.
func (l *Lexer) colonVariable() *token.Token {
	var r rune
	var eof bool
	offsetStart := l.r.getOffset()
	l.r.readRune()
	if r, eof = l.r.peekRune(); eof {
		err := string(l.r.slice(offsetStart, l.r.getOffset())) + ": unexpected EOF"
		return token.New([]byte(err), token.KindError)
	} else if !l.isAlphabetic(r) {
		err := string(l.r.slice(offsetStart, l.r.getOffset())) + ": invalid character after colon"
		return token.New([]byte(err), token.KindError)
	}

	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if !l.isAlphanumeric(r) && r != '$' {
			break
		}
	}
	if !eof {
		l.r.unreadRune()
	}

	return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindColonVariable)
}

// atVariable scans a at variable.
func (l *Lexer) atVariable() *token.Token {
	var r rune
	var eof bool
	offsetStart := l.r.getOffset()
	l.r.readRune()
	if r, eof = l.r.peekRune(); eof {
		err := string(l.r.slice(offsetStart, l.r.getOffset())) + ": unexpected EOF"
		return token.New([]byte(err), token.KindError)
	} else if !l.isAlphabetic(r) {
		err := string(l.r.slice(offsetStart, l.r.getOffset())) + ": invalid character after at"
		return token.New([]byte(err), token.KindError)
	}

	for r, eof = l.r.readRune(); !eof; r, eof = l.r.readRune() {
		if !l.isAlphanumeric(r) && r != '$' {
			break
		}
	}
	if !eof {
		l.r.unreadRune()
	}

	return token.New(l.r.slice(offsetStart, l.r.getOffset()), token.KindAtVariable)
}

// operator scans an operator.
func (l *Lexer) operator() *token.Token {
	offsetStart := l.r.getOffset()
	r, _ := l.r.readRune()
	var kind token.Kind
	switch r {
	case '-':
		kind = token.KindMinus
	case '(':
		kind = token.KindLeftParen
	case ')':
		kind = token.KindRightParen
	case ';':
		kind = token.KindSemicolon
	case '+':
		kind = token.KindPlus
	case '*':
		kind = token.KindAsterisk
	case '/':
		kind = token.KindSlash
	case '%':
		kind = token.KindPercent
	case ',':
		kind = token.KindComma
	case '&':
		kind = token.KindAmpersand
	case '~':
		kind = token.KindTilde
	case '.':
		kind = token.KindDot
	case '=':
		r, eof := l.r.peekRune()
		if eof || r != '=' {
			kind = token.KindEqual
		} else {
			kind = token.KindEqualEqual
			l.r.readRune()
		}
	case '<':
		r, eof := l.r.peekRune()
		if eof || !strings.ContainsRune("=><", r) {
			kind = token.KindLessThan
		} else if r == '=' {
			kind = token.KindLessThanOrEqual
			l.r.readRune()
		} else if r == '<' {
			kind = token.KindLessThanLessThan
			l.r.readRune()
		} else {
			kind = token.KindLessThanGreaterThan
			l.r.readRune()
		}
	case '>':
		r, eof := l.r.peekRune()
		if eof || !strings.ContainsRune("=>", r) {
			kind = token.KindGreaterThan
		} else if r == '=' {
			kind = token.KindGreaterThanEqual
			l.r.readRune()
		} else {
			kind = token.KindGreaterThanGreaterThan
			l.r.readRune()
		}
	case '!':
		r, eof := l.r.readRune()
		if eof {
			lexeme := l.r.slice(offsetStart, l.r.getOffset())
			err := string(lexeme) + ": unexpected EOF"
			return token.New([]byte(err), token.KindError)
		}
		if r == '=' {
			kind = token.KindExclamationEqual
		} else {
			l.r.unreadRune()
			lexeme := l.r.slice(offsetStart, l.r.getOffset())
			err := string(lexeme) + ": unexpected character"
			return token.New([]byte(err), token.KindError)
		}
	default: // r == '|'
		r, eof := l.r.peekRune()
		if eof || r != '|' {
			kind = token.KindPipe
		} else {
			kind = token.KindPipePipe
			l.r.readRune()
		}
	}
	lexeme := l.r.slice(offsetStart, l.r.getOffset())
	return token.New(lexeme, kind)
}

// isWhiteSpace reports whether the rune is white space (with respect to the SQLite SQL dialect).
func (l *Lexer) isWhiteSpace(r rune) bool {
	return strings.ContainsRune("\t\n\x0C\x0D ", r)
}

// isAlphabetic reports whether the rune is alphabetic (with respect to the SQLite SQL dialect).
func (l *Lexer) isAlphabetic(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 0x7F
}

// isNumeric reports whether the rune is numeric (with respect to the SQLite SQL dialect).
func (l *Lexer) isNumeric(r rune) bool {
	return r >= '0' && r <= '9'
}

// isAlphanumeric reports whether the rune is alphanumeric (with respect to the SQLite SQL dialect).
func (l *Lexer) isAlphanumeric(r rune) bool {
	return l.isAlphabetic(r) || l.isNumeric(r)
}

// isHexadecimal reports whether the rune is hexadecimal (with respect to the SQLite SQL dialect).
func (l *Lexer) isHexadecimal(r rune) bool {
	return l.isNumeric(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// isSpecial reports whether the rune is special (with respect to the SQLite SQL dialect).
func (l *Lexer) isSpecial(r rune) bool {
	return !l.isWhiteSpace(r) && !l.isAlphabetic(r) && !l.isNumeric(r)
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
			panic(errors.New("utf-8 encoding invalid"))
		}
	}
	r.offset += int64(size)
	return
}

// peekRune returns the next rune but dont advances the lexer, this means that if readRune is called it will return the same rune.
// Similarly for the EOF.
func (r *reader) peekRune() (rn rune, eof bool) {
	rn, size := utf8.DecodeRune(r.code[r.offset:])
	if rn == utf8.RuneError {
		if size == 0 {
			return 0, true
		} else {
			panic(errors.New("utf-8 encoding invalid"))
		}
	}
	return
}

// peekNRunes returns the next n runes but dont advances the lexer. It can returns less than n runes if EOF is found before
// n runes are read.
func (r *reader) peekNRunes(n int) (rs []rune, eof bool) {
	offset := r.getOffset()
	for range n {
		rn, size := utf8.DecodeRune(r.code[offset:])
		if rn == utf8.RuneError {
			if size == 0 {
				return rs, true
			} else {
				panic(errors.New("utf-8 encoding invalid"))
			}
		}
		rs = append(rs, rn)
		offset += int64(size)
	}

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
	panic(errors.New("utf-8 encoding invalid"))
}

// getOffset returns the current offset.
func (r *reader) getOffset() int64 {
	return r.offset
}

// slice returns r.code[offsetStart:offsetEnd].
func (r *reader) slice(offsetStart, offsetEnd int64) []byte {
	return r.code[offsetStart:offsetEnd]
}
