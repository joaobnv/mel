package lexer

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

// TestLexer tests the lexer.
func TestLexer(t *testing.T) {
	cases := []struct {
		code   string
		tokens []*token.Token
	}{
		{code: "", tokens: parseTokens(`<EOF>`)},
		{code: "\t\n\x0C\x0D\x20", tokens: parseTokens(`<"\t\n\x0C\x0D\x20", WhiteSpace> <EOF>`)},
		{code: "abc ", tokens: parseTokens(`<"abc", Identifier> <" ", WhiteSpace> <EOF>`)},
		{code: "TABLE", tokens: parseTokens(`<"TABLE", Table> <EOF>`)},
		{code: "\"TABLE\"", tokens: parseTokens(`<"\"TABLE\"", Identifier> <EOF>`)},
		{code: "\"TAB\"\"LE\"", tokens: parseTokens(`<"\"TAB\"\"LE\"", Identifier> <EOF>`)},
		{code: "\"TABLE", tokens: parseTokens(`<"\"TABLE", ErrorUnexpectedEOF> <EOF>`)},
		{code: "[TABLE]", tokens: parseTokens(`<"[TABLE]", Identifier> <EOF>`)},
		{code: "[TABLE", tokens: parseTokens(`<"[TABLE", ErrorUnexpectedEOF> <EOF>`)},
		{code: "`TABLE`", tokens: parseTokens("<\"`TABLE`\", Identifier> <EOF>")},
		{code: "`TAB``LE`", tokens: parseTokens("<\"`TAB``LE`\", Identifier> <EOF>")},
		{code: "`TABLE", tokens: parseTokens("<\"`TABLE\", ErrorUnexpectedEOF> <EOF>")},
		{code: "'TABLE'", tokens: parseTokens("<\"'TABLE'\", String> <EOF>")},
		{code: "'TAB''LE'", tokens: parseTokens("<\"'TAB''LE'\", String> <EOF>")},
		{code: "X'CAFE'", tokens: parseTokens("<\"X'CAFE'\", Blob> <EOF>")},
		{code: "x'cafe'", tokens: parseTokens("<\"x'cafe'\", Blob> <EOF>")},
		{code: "x'cafe", tokens: parseTokens("<\"x'cafe\", ErrorUnexpectedEOF> <EOF>")},
		// the lexer doesn't care if the number of hexadecimal runes is even
		{code: "x'C0FEE'", tokens: parseTokens("<\"x'C0FEE'\", Blob> <EOF>")},
		{code: "x'CAR'", tokens: parseTokens("<\"x'CAR\", ErrorBlobNotHexadecimal> <\"'\", ErrorUnexpectedEOF> <EOF>")},
		{code: "1", tokens: parseTokens(`<"1", Numeric> <EOF>`)},
		{code: "1234", tokens: parseTokens(`<"1234", Numeric> <EOF>`)},
		{code: "1_2", tokens: parseTokens(`<"1_2", Numeric> <EOF>`)},
		{code: "_1_2", tokens: parseTokens(`<"_1_2", Identifier> <EOF>`)},
		{code: "1_2_", tokens: parseTokens(`<"1_2", Numeric> <"_", Identifier> <EOF>`)},
		{code: "1.", tokens: parseTokens(`<"1.", Numeric> <EOF>`)},
		{code: "1234.", tokens: parseTokens(`<"1234.", Numeric> <EOF>`)},
		{code: "1_23_4.", tokens: parseTokens(`<"1_23_4.", Numeric> <EOF>`)},
		{code: "1.0", tokens: parseTokens(`<"1.0", Numeric> <EOF>`)},
		{code: "1.9_1", tokens: parseTokens(`<"1.9_1", Numeric> <EOF>`)},
		{code: "1.9E", tokens: parseTokens(`<"1.9", Numeric> <"E", Identifier> <EOF>`)},
		{code: "1.9e", tokens: parseTokens(`<"1.9", Numeric> <"e", Identifier> <EOF>`)},
		{code: "1.9EA", tokens: parseTokens(`<"1.9", Numeric> <"EA", Identifier> <EOF>`)},
		{code: "1.9E8", tokens: parseTokens(`<"1.9E8", Numeric> <EOF>`)},
		{code: "1.9e8_0", tokens: parseTokens(`<"1.9e8_0", Numeric> <EOF>`)},
		{code: "1.9E+8_0", tokens: parseTokens(`<"1.9E+8_0", Numeric> <EOF>`)},
		{code: "1.9e-00", tokens: parseTokens(`<"1.9e-00", Numeric> <EOF>`)},
		{code: "1.9E-", tokens: parseTokens(`<"1.9", Numeric> <"E", Identifier> <"-", Minus> <EOF>`)},
		{code: "1.9e+", tokens: parseTokens(`<"1.9", Numeric> <"e", Identifier> <"+", Plus> <EOF>`)},
		{code: "1.9e+A", tokens: parseTokens(`<"1.9", Numeric> <"e", Identifier> <"+", Plus> <"A", Identifier> <EOF>`)},
		{code: ".99", tokens: parseTokens(`<".99", Numeric> <EOF>`)},
		{code: ".9e+3", tokens: parseTokens(`<".9e+3", Numeric> <EOF>`)},
		{code: ".9_1e+3_2", tokens: parseTokens(`<".9_1e+3_2", Numeric> <EOF>`)},
		{code: "0X92A", tokens: parseTokens(`<"0X92A", Numeric> <EOF>`)},
		{code: "0x0_F", tokens: parseTokens(`<"0x0_F", Numeric> <EOF>`)},
		{code: "0x0_F_", tokens: parseTokens(`<"0x0_F", Numeric> <"_", Identifier> <EOF>`)},
		{code: "0x0_FQ", tokens: parseTokens(`<"0x0_F", Numeric> <"Q", Identifier> <EOF>`)},
		{code: "-- comment", tokens: parseTokens(`<"-- comment", SQLComment> <EOF>`)},
		{code: "-- comment\ntable", tokens: parseTokens(`<"-- comment", SQLComment> <"\n", WhiteSpace> <"table", Table> <EOF>`)},
		{code: "/**/", tokens: parseTokens(`<"/**/", CComment> <EOF>`)},
		{code: "/*****/ table", tokens: parseTokens(`<"/*****/", CComment> <" ", WhiteSpace> <"table", Table> <EOF>`)},
		{code: "/**", tokens: parseTokens(`<"/**", CComment> <EOF>`)},
		{code: "/* comment\n table", tokens: parseTokens(`<"/* comment\n table", CComment> <EOF>`)},
		{code: "/* comment*/ table", tokens: parseTokens(`<"/* comment*/", CComment> <" ", WhiteSpace> <"table", Table> <EOF>`)},
		{code: "/* comment* **/ table", tokens: parseTokens(`<"/* comment* **/", CComment> <" ", WhiteSpace> <"table", Table> <EOF>`)},
		{code: "?", tokens: parseTokens(`<"?", QuestionVariable> <EOF>`)},
		{code: "?0", tokens: parseTokens(`<"?0", QuestionVariable> <EOF>`)},
		{code: "?90", tokens: parseTokens(`<"?90", QuestionVariable> <EOF>`)},
		{code: "?75a", tokens: parseTokens(`<"?75", QuestionVariable> <"a", Identifier> <EOF>`)},
		{code: ":table", tokens: parseTokens(`<":table", ColonVariable> <EOF>`)},
		{code: ":tab$le", tokens: parseTokens(`<":tab$le", ColonVariable> <EOF>`)},
		{code: ":_tab$20", tokens: parseTokens(`<":_tab$20", ColonVariable> <EOF>`)},
		{code: ":select *", tokens: parseTokens(`<":select", ColonVariable> <" ", WhiteSpace> <"*", Asterisk> <EOF>`)},
		{code: ":", tokens: parseTokens(`<":", ErrorUnexpectedEOF> <EOF>`)},
		{code: ":2", tokens: parseTokens(`<":", ErrorInvalidCharacterAfter> <"2", Numeric> <EOF>`)},
		{code: "@table", tokens: parseTokens(`<"@table", AtVariable> <EOF>`)},
		{code: "@tab$le", tokens: parseTokens(`<"@tab$le", AtVariable> <EOF>`)},
		{code: "@_tab$20", tokens: parseTokens(`<"@_tab$20", AtVariable> <EOF>`)},
		{code: "@select *", tokens: parseTokens(`<"@select", AtVariable> <" ", WhiteSpace> <"*", Asterisk> <EOF>`)},
		{code: "@", tokens: parseTokens(`<"@", ErrorUnexpectedEOF> <EOF>`)},
		{code: "@2", tokens: parseTokens(`<"@", ErrorInvalidCharacterAfter> <"2", Numeric> <EOF>`)},
		{code: "$table", tokens: parseTokens(`<"$table", DollarVariable> <EOF>`)},
		{code: "$tab$le", tokens: parseTokens(`<"$tab$le", DollarVariable> <EOF>`)},
		{code: "$_tab$20", tokens: parseTokens(`<"$_tab$20", DollarVariable> <EOF>`)},
		{code: "$select *", tokens: parseTokens(`<"$select", DollarVariable> <" ", WhiteSpace> <"*", Asterisk> <EOF>`)},
		{code: "$", tokens: parseTokens(`<"$", ErrorUnexpectedEOF> <EOF>`)},
		{code: "$2", tokens: parseTokens(`<"$", ErrorInvalidCharacterAfter> <"2", Numeric> <EOF>`)},
		{code: "$tab::le", tokens: parseTokens(`<"$tab::le", DollarVariable> <EOF>`)},
		{code: "$tab:le", tokens: parseTokens(`<"$tab", DollarVariable> <":le", ColonVariable> <EOF>`)},
		{code: "$tab::le::(variable)", tokens: parseTokens(`<"$tab::le::(variable)", DollarVariable> <EOF>`)},
		{code: "$tab::le::()", tokens: parseTokens(`<"$tab::le::()", DollarVariable> <EOF>`)},
		{code: "$tab::le::(variable", tokens: parseTokens(`<"$tab::le::(variable", ErrorUnexpectedEOF> <EOF>`)},
		{code: "$tab::le::(variable)(ok)",
			tokens: parseTokens(`<"$tab::le::(variable)", DollarVariable> <"(", LeftParen> <"ok", Identifier> <")", RightParen> <EOF>`)},
		{code: "-", tokens: parseTokens(`<"-", Minus> <EOF>`)},
		{code: "->", tokens: parseTokens(`<"->", MinusGreaterThan> <EOF>`)},
		{code: "->>", tokens: parseTokens(`<"->>", MinusGreaterThanGreaterThan> <EOF>`)},
		{code: "(", tokens: parseTokens(`<"(", LeftParen> <EOF>`)},
		{code: ")", tokens: parseTokens(`<")", RightParen> <EOF>`)},
		{code: ";", tokens: parseTokens(`<";", Semicolon> <EOF>`)},
		{code: "+", tokens: parseTokens(`<"+", Plus> <EOF>`)},
		{code: "*", tokens: parseTokens(`<"*", Asterisk> <EOF>`)},
		{code: "/", tokens: parseTokens(`<"/", Slash> <EOF>`)},
		{code: "/ * */",
			tokens: parseTokens(`<"/", Slash> <" ", WhiteSpace> <"*", Asterisk> <" ", WhiteSpace> <"*", Asterisk> <"/", Slash> <EOF>`)},
		{code: "%", tokens: parseTokens(`<"%", Percent> <EOF>`)},
		{code: "=", tokens: parseTokens(`<"=", Equal> <EOF>`)},
		{code: "==", tokens: parseTokens(`<"==", EqualEqual> <EOF>`)},
		{code: "<=", tokens: parseTokens(`<"<=", LessThanOrEqual> <EOF>`)},
		{code: "<>", tokens: parseTokens(`<"<>", LessThanGreaterThan> <EOF>`)},
		{code: "<<", tokens: parseTokens(`<"<<", LessThanLessThan> <EOF>`)},
		{code: "<", tokens: parseTokens(`<"<", LessThan> <EOF>`)},
		{code: ">=", tokens: parseTokens(`<">=", GreaterThanEqual> <EOF>`)},
		{code: ">>", tokens: parseTokens(`<">>", GreaterThanGreaterThan> <EOF>`)},
		{code: ">", tokens: parseTokens(`<">", GreaterThan> <EOF>`)},
		{code: "!=", tokens: parseTokens(`<"!=", ExclamationEqual> <EOF>`)},
		{code: "!", tokens: parseTokens(`<"!", ErrorUnexpectedEOF> <EOF>`)},
		{code: "!a", tokens: parseTokens(`<"!", ErrorInvalidCharacterAfter> <"a", Identifier> <EOF>`)},
		{code: ",", tokens: parseTokens(`<",", Comma> <EOF>`)},
		{code: "&", tokens: parseTokens(`<"&", Ampersand> <EOF>`)},
		{code: "~", tokens: parseTokens(`<"~", Tilde> <EOF>`)},
		{code: "|", tokens: parseTokens(`<"|", Pipe> <EOF>`)},
		{code: "||", tokens: parseTokens(`<"||", PipePipe> <EOF>`)},
		{code: ".", tokens: parseTokens(`<".", Dot> <EOF>`)},
		{code: "\x00", tokens: parseTokens(`<"\x00", ErrorInvalidCharacter> <EOF>`)},
		{code: "^", tokens: parseTokens(`<"^", ErrorInvalidCharacter> <EOF>`)},
	}

	for _, c := range cases {
		l := New([]byte(c.code))

		var scanned []*token.Token
		for {
			tok := l.Next()
			scanned = append(scanned, tok)
			if tok.Kind == token.KindEOF {
				break
			}
		}

		equals := slices.EqualFunc(c.tokens, scanned, func(a, b *token.Token) bool {
			if a.Kind != b.Kind {
				return false
			}
			return bytes.Equal(a.Lexeme, b.Lexeme)
		})

		if !equals {
			t.Errorf("code=\"%s\": tokens differ", c.code)
			var b strings.Builder
			printTokens(&b, c.tokens, scanned)
			t.Log(b.String())
		}
	}
}

// printTokens writes the tokens to b in tabular form.
func printTokens(b *strings.Builder, expected, scanned []*token.Token) {
	// we use a strings.Builder because it dont returns errors like the more general io.Writer.
	tw := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintln(tw, "\nexpected\tscanned")
	for i := range expected {
		fmt.Fprintf(tw, "%s\t", expected[i])
		if i < len(scanned) {
			fmt.Fprintf(tw, "%s\n", scanned[i])
		} else {
			fmt.Fprintln(tw)
		}
	}
	for i := len(expected); i < len(scanned); i++ {
		fmt.Fprintf(tw, "\t%s\n", scanned[i])
	}
}

// TestReaderReadPanic tests the case where the reader reads from a invalid UTF-8 encoded byte slice.
func TestReaderReadPanic(t *testing.T) {
	defer func() {
		resultPanic := recover()
		if resultPanic == nil {
			t.Errorf("not panic")
			return
		}
		err := resultPanic.(error)
		if err.Error() != "utf-8 encoding invalid" {
			t.Errorf("invalid error message: %s", err)
		}
	}()
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := newReader(data)
	r.readRune()
}

// TestReaderPeekPanic tests the case where the reader peeks from a invalid UTF-8 encoded byte slice.
func TestReaderPeekPanic(t *testing.T) {
	defer func() {
		resultPanic := recover()
		if resultPanic == nil {
			t.Errorf("not panic")
			return
		}
		err := resultPanic.(error)
		if err.Error() != "utf-8 encoding invalid" {
			t.Errorf("invalid error message: %s", err)
		}
	}()
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := newReader(data)
	r.peekRune()
}

// TestReaderPeekPanic tests the case where the reader peekNRunes reads from a invalid UTF-8 encoded byte slice.
func TestReaderPeekNPanic(t *testing.T) {
	defer func() {
		resultPanic := recover()
		if resultPanic == nil {
			t.Errorf("not panic")
			return
		}
		err := resultPanic.(error)
		if err.Error() != "utf-8 encoding invalid" {
			t.Errorf("invalid error message: %s", err)
		}
	}()
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := newReader(data)
	r.peekNRunes(1)
}

// TestReaderUnreadOnStart tests the case where the read is at the start and we try a unreadRune.
func TestReaderUnreadOnStart(t *testing.T) {
	data := []byte{0x00}
	r := newReader(data)
	onStart := r.unreadRune()
	if !onStart {
		t.Error("not on start")
	}
}

// TestReaderUnreadOnStart tests the case where the unreadRune is called more than one times.
func TestReaderMoreThanOneUnread(t *testing.T) {
	r := newReader([]byte("abcde"))

	r.readRune()
	r.readRune()
	r.readRune()
	r.readRune()
	r.readRune()

	r.unreadRune()
	r.unreadRune()
	r.unreadRune()
	r.unreadRune()
	r.unreadRune()

	rn, eof := r.readRune()
	if eof {
		t.Errorf("at EOF")
		return
	}

	if rn != 'a' {
		t.Errorf("exected 'a', got %q", rn)
	}
}

// TestReaderUnreadPanic tests the case where a unreadRune cannot find a valid start of rune.
func TestReaderUnreadPanic(t *testing.T) {
	defer func() {
		resultPanic := recover()
		if resultPanic == nil {
			t.Errorf("not panic")
			return
		}
		err := resultPanic.(error)
		if err.Error() != "utf-8 encoding invalid" {
			t.Errorf("invalid error message: %s", err)
		}
	}()
	data := []byte{0x00, 0xFF, 0xFF, 0xFF}
	r := newReader(data)
	r.readRune()
	data[0] = 0b10111111
	r.unreadRune()
}

// parseTokens unmarshalls tokens from code.
func parseTokens(code string) (result []*token.Token) {
	re := regexp.MustCompile(`<(?:("(?:[^\\]|\\"|\\[^"])*?"),\s?)?([a-zA-Z]+)>`)
	matches := re.FindAllStringSubmatch(code, -1)
	for i := range matches {
		kind, ok := kinds[matches[i][2]]
		if !ok {
			panic(fmt.Errorf("kind not exported by package token: %s", matches[i][2]))
		}

		var lex []byte
		if matches[i][1] != "" {
			lexStr, err := strconv.Unquote(matches[i][1])
			if err != nil {
				panic(err)
			}
			lex = []byte(lexStr)
		}

		result = append(result, token.New(lex, kind))
	}
	return
}

// kinds is a map string representations of token kind and the corresponding kind. It contains only kinds
// exported by the package token.
var kinds = map[string]token.Kind{
	"Abort":                       token.KindAbort,
	"Action":                      token.KindAction,
	"Add":                         token.KindAdd,
	"After":                       token.KindAfter,
	"All":                         token.KindAll,
	"Alter":                       token.KindAlter,
	"Always":                      token.KindAlways,
	"Analyze":                     token.KindAnalyze,
	"And":                         token.KindAnd,
	"As":                          token.KindAs,
	"Asc":                         token.KindAsc,
	"Attach":                      token.KindAttach,
	"Autoincrement":               token.KindAutoincrement,
	"Before":                      token.KindBefore,
	"Begin":                       token.KindBegin,
	"Between":                     token.KindBetween,
	"By":                          token.KindBy,
	"Cascade":                     token.KindCascade,
	"Case":                        token.KindCase,
	"Cast":                        token.KindCast,
	"Check":                       token.KindCheck,
	"Collate":                     token.KindCollate,
	"Column":                      token.KindColumn,
	"Commit":                      token.KindCommit,
	"Conflict":                    token.KindConflict,
	"Constraint":                  token.KindConstraint,
	"Create":                      token.KindCreate,
	"Cross":                       token.KindCross,
	"Current":                     token.KindCurrent,
	"CurrentDate":                 token.KindCurrentDate,
	"CurrentTime":                 token.KindCurrentTime,
	"CurrentTimestamp":            token.KindCurrentTimestamp,
	"Database":                    token.KindDatabase,
	"Default":                     token.KindDefault,
	"Deferrable":                  token.KindDeferrable,
	"Deferred":                    token.KindDeferred,
	"Delete":                      token.KindDelete,
	"Desc":                        token.KindDesc,
	"Detach":                      token.KindDetach,
	"Distinct":                    token.KindDistinct,
	"Do":                          token.KindDo,
	"Drop":                        token.KindDrop,
	"Each":                        token.KindEach,
	"Else":                        token.KindElse,
	"End":                         token.KindEnd,
	"Escape":                      token.KindEscape,
	"Except":                      token.KindExcept,
	"Exclude":                     token.KindExclude,
	"Exclusive":                   token.KindExclusive,
	"Exists":                      token.KindExists,
	"Explain":                     token.KindExplain,
	"Fail":                        token.KindFail,
	"Filter":                      token.KindFilter,
	"First":                       token.KindFirst,
	"Following":                   token.KindFollowing,
	"For":                         token.KindFor,
	"Foreign":                     token.KindForeign,
	"From":                        token.KindFrom,
	"Full":                        token.KindFull,
	"Generated":                   token.KindGenerated,
	"Glob":                        token.KindGlob,
	"Group":                       token.KindGroup,
	"Groups":                      token.KindGroups,
	"Having":                      token.KindHaving,
	"If":                          token.KindIf,
	"Ignore":                      token.KindIgnore,
	"Immediate":                   token.KindImmediate,
	"In":                          token.KindIn,
	"Index":                       token.KindIndex,
	"Indexed":                     token.KindIndexed,
	"Initially":                   token.KindInitially,
	"Inner":                       token.KindInner,
	"Insert":                      token.KindInsert,
	"Instead":                     token.KindInstead,
	"Intersect":                   token.KindIntersect,
	"Into":                        token.KindInto,
	"Is":                          token.KindIs,
	"Isnull":                      token.KindIsnull,
	"Join":                        token.KindJoin,
	"Key":                         token.KindKey,
	"Last":                        token.KindLast,
	"Left":                        token.KindLeft,
	"Like":                        token.KindLike,
	"Limit":                       token.KindLimit,
	"Match":                       token.KindMatch,
	"Materialized":                token.KindMaterialized,
	"Natural":                     token.KindNatural,
	"No":                          token.KindNo,
	"Not":                         token.KindNot,
	"Nothing":                     token.KindNothing,
	"Notnull":                     token.KindNotnull,
	"Null":                        token.KindNull,
	"Nulls":                       token.KindNulls,
	"Of":                          token.KindOf,
	"Offset":                      token.KindOffset,
	"On":                          token.KindOn,
	"Or":                          token.KindOr,
	"Order":                       token.KindOrder,
	"Others":                      token.KindOthers,
	"Outer":                       token.KindOuter,
	"Over":                        token.KindOver,
	"Partition":                   token.KindPartition,
	"Plan":                        token.KindPlan,
	"Pragma":                      token.KindPragma,
	"Preceding":                   token.KindPreceding,
	"Primary":                     token.KindPrimary,
	"Query":                       token.KindQuery,
	"Raise":                       token.KindRaise,
	"Range":                       token.KindRange,
	"Recursive":                   token.KindRecursive,
	"References":                  token.KindReferences,
	"Regexp":                      token.KindRegexp,
	"Reindex":                     token.KindReindex,
	"Release":                     token.KindRelease,
	"Rename":                      token.KindRename,
	"Replace":                     token.KindReplace,
	"Restrict":                    token.KindRestrict,
	"Returning":                   token.KindReturning,
	"Right":                       token.KindRight,
	"Rollback":                    token.KindRollback,
	"Row":                         token.KindRow,
	"Rows":                        token.KindRows,
	"Savepoint":                   token.KindSavepoint,
	"Select":                      token.KindSelect,
	"Set":                         token.KindSet,
	"Table":                       token.KindTable,
	"Temp":                        token.KindTemp,
	"Temporary":                   token.KindTemporary,
	"Then":                        token.KindThen,
	"Ties":                        token.KindTies,
	"To":                          token.KindTo,
	"Transaction":                 token.KindTransaction,
	"Trigger":                     token.KindTrigger,
	"Unbounded":                   token.KindUnbounded,
	"Union":                       token.KindUnion,
	"Unique":                      token.KindUnique,
	"Update":                      token.KindUpdate,
	"Using":                       token.KindUsing,
	"Vacuum":                      token.KindVacuum,
	"Values":                      token.KindValues,
	"View":                        token.KindView,
	"Virtual":                     token.KindVirtual,
	"When":                        token.KindWhen,
	"Where":                       token.KindWhere,
	"Window":                      token.KindWindow,
	"With":                        token.KindWith,
	"Without":                     token.KindWithout,
	"Identifier":                  token.KindIdentifier,
	"String":                      token.KindString,
	"Blob":                        token.KindBlob,
	"Numeric":                     token.KindNumeric,
	"SQLComment":                  token.KindSQLComment,
	"CComment":                    token.KindCComment,
	"QuestionVariable":            token.KindQuestionVariable,
	"ColonVariable":               token.KindColonVariable,
	"AtVariable":                  token.KindAtVariable,
	"DollarVariable":              token.KindDollarVariable,
	"Minus":                       token.KindMinus,
	"MinusGreaterThan":            token.KindMinusGreaterThan,
	"MinusGreaterThanGreaterThan": token.KindMinusGreaterThanGreaterThan,
	"LeftParen":                   token.KindLeftParen,
	"RightParen":                  token.KindRightParen,
	"Semicolon":                   token.KindSemicolon,
	"Plus":                        token.KindPlus,
	"Asterisk":                    token.KindAsterisk,
	"Slash":                       token.KindSlash,
	"Percent":                     token.KindPercent,
	"Equal":                       token.KindEqual,
	"EqualEqual":                  token.KindEqualEqual,
	"LessThanOrEqual":             token.KindLessThanOrEqual,
	"LessThanGreaterThan":         token.KindLessThanGreaterThan,
	"LessThanLessThan":            token.KindLessThanLessThan,
	"LessThan":                    token.KindLessThan,
	"GreaterThanEqual":            token.KindGreaterThanOrEqual,
	"GreaterThanGreaterThan":      token.KindGreaterThanGreaterThan,
	"GreaterThan":                 token.KindGreaterThan,
	"ExclamationEqual":            token.KindExclamationEqual,
	"Comma":                       token.KindComma,
	"Ampersand":                   token.KindAmpersand,
	"Tilde":                       token.KindTilde,
	"Pipe":                        token.KindPipe,
	"PipePipe":                    token.KindPipePipe,
	"Dot":                         token.KindDot,
	"WhiteSpace":                  token.KindWhiteSpace,
	"ErrorUnexpectedEOF":          token.KindErrorUnexpectedEOF,
	"ErrorBlobNotHexadecimal":     token.KindErrorBlobNotHexadecimal,
	"ErrorInvalidCharacter":       token.KindErrorInvalidCharacter,
	"ErrorInvalidCharacterAfter":  token.KindErrorInvalidCharacterAfter,
	"EOF":                         token.KindEOF,
}
