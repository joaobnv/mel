// This package deals with a parse tree of the SQL.
package parsetree

import (
	"strconv"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

// Kind is the kind of the tree.
type Kind int

const (
	KindAdd Kind = iota
	KindAddColumn
	KindAlterTable
	KindAnalyze
	KindAnd
	KindAttach
	KindBetween
	KindBindParameter
	KindBitAnd
	KindBitNot
	KindBitOr
	KindCase
	KindCast
	KindCollate
	KindCollationName
	KindColumnConstraint
	KindColumnDefinition
	KindColumnName
	KindColumnReference
	KindCommaList
	KindConcatenate
	KindConflictClause
	KindConstraintName
	KindDivide
	KindDropColumn
	KindElse
	KindEqual
	KindErrorExpecting // is for when there was more than one possibility
	KindErrorMessage
	KindErrorMissing // is for when there was only one possibility
	KindErrorUnexpectedEOF
	KindExists
	KindExplain
	KindExplainQueryPlan
	KindExpression
	KindExtract1
	KindExtract2
	KindFilterClause
	KindForeignKeyClause
	KindFrameSpec
	KindFrameSpecBetween
	KindFunctionArguments
	KindFunctionCall
	KindFunctionName
	KindGlob
	KindGreaterThan
	KindGreaterThanOrEqual
	KindIn
	KindIs
	KindIsDistinctFrom
	KindIsNot
	KindIsNotDistinctFrom
	KindIsNull
	KindLeftShift
	KindLessThan
	KindLessThanOrEqual
	KindLike
	KindMatch
	KindMod
	KindMultiply
	KindNegate
	KindNot
	KindNotBetween
	KindNotEqual
	KindNotGlob
	KindNotIn
	KindNotLike
	KindNotMatch
	KindNotnull
	KindNotNull
	KindNotRegexp
	KindOr
	KindOrderBy
	KindOrderingTerm
	KindOverClause
	KindParenExpression
	KindPartitionBy
	KindPrefixPlus
	KindRaise
	KindRegexp
	KindRenameColumn
	KindRenameTo
	KindRightShift
	KindSchemaIndexOrTableName
	KindSchemaName
	KindSelect
	KindSkipped
	KindSQLStatement
	KindSubtract
	KindTableFunctionName
	KindTableName
	KindTableOrIndexName
	KindToken
	KindTypeName
	KindWhen
	KindWindowName
)

// String returns a string representation of k.
func (k Kind) String() string {
	if k < 0 || int(k) >= len(kindStrings) {
		return strconv.Itoa(int(k))
	}
	return kindStrings[k]
}

// kindStrings contains the string representation of the kinds. Note that the value of a kind is the index of your string
// representation.
var kindStrings = []string{
	"Add", "AddColumn", "AlterTable", "Analyze", "And", "Attach", "Between", "BindParameter", "BitAnd", "BitNot", "BitOr", "Case", "Cast", "Collate", "CollationName",
	"ColumnConstraint", "ColumnDefinition", "ColumnName", "ColumnReference", "CommaList", "Concatenate", "ConflictClause", "ConstraintName", "Divide",
	"DropColumn", "Else", "Equal", "ErrorExpecting", "ErrorMessage", "ErrorMissing", "ErrorUnexpectedEOF", "Exists", "Explain", "ExplainQueryPlan",
	"Expression", "Extract1", "Extract2", "FilterClause", "ForeignKeyClause", "FrameSpec", "FrameSpecBetween", "FunctionArguments", "FunctionCall",
	"FunctionName", "Glob", "GreaterThan", "GreaterThanOrEqual", "In", "Is", "IsDistinctFrom", "IsNot", "IsNotDistinctFrom", "IsNull", "LeftShift",
	"LessThan", "LessThanOrEqual", "Like", "Match", "Mod", "Multiply", "Negate", "Not", "NotBetween", "NotEqual", "NotGlob", "NotIn", "NotLike",
	"NotMatch", "Notnull", "NotNull", "NotRegexp", "Or", "OrderBy", "OrderingTerm", "OverClause", "ParenExpression", "PartitionBy", "PrefixPlus", "Raise",
	"Regexp", "RenameColumn", "RenameTo", "RightShift", "SchemaIndexOrTableName", "SchemaName", "Select", "Skipped", "SQLStatement", "Subtract", "TableFunctionName", "TableName",
	"TableOrIndexName", "Token", "TypeName", "When", "WindowName",
}

// Construction is a construction in SQL grammar.
type Construction interface {
	// Kind returns the kind of the terminal.
	Kind() Kind
}

// NonTerminal terminal represents a non terminal in the SQL grammar.
type NonTerminal interface {
	Construction
	// AddChild add a child to the non terminal.
	AddChild(Construction)
	// NumberOfChildren returns the number of children of the non terminal.
	NumberOfChildren() int
	// Children is a iterator for the children of the non terminal.
	Children(func(Construction) bool)
}

// NewNonTerminal creates a NonTerminal.
func NewNonTerminal(kind Kind) NonTerminal {
	return &nonTerminal{
		kind: kind,
	}
}

// nonTerminal is a non terminal for SQL.
type nonTerminal struct {
	// kind is the kind of the tree.
	kind Kind
	// children contains the children of this tree.
	children []Construction
}

// Kind returns the kind of the non terminal.
func (nt *nonTerminal) Kind() Kind {
	return nt.kind
}

// AddChild add a child to nt.
func (nt *nonTerminal) AddChild(c Construction) {
	nt.children = append(nt.children, c)
}

// NumberOfChildren returns the number of children of nt.
func (nt *nonTerminal) NumberOfChildren() int {
	return len(nt.children)
}

// Children is a iterator for the children of nt.
func (nt *nonTerminal) Children(yield func(Construction) bool) {
	for i := range nt.children {
		if !yield(nt.children[i]) {
			return
		}
	}
}

// Terminal is a terminal of the parse tree.
type Terminal interface {
	Construction
	// Token returns the token of the terminal.
	Token() *token.Token
}

// NewLeaf creates a Terminal.
func NewTerminal(kind Kind, tok *token.Token) Terminal {
	return &terminal{kind: kind, tok: tok}
}

// terminal is a terminal of a tree.
type terminal struct {
	// kind is the kind of the terminal.
	kind Kind
	// tok is the token of the leaf.
	tok *token.Token
}

// Kind returns the kind of the terminal.
func (t *terminal) Kind() Kind {
	return t.kind
}

// Token returns the token of the terminal.
func (t *terminal) Token() *token.Token {
	return t.tok
}

// Error is a error found in the parsing.
type Error interface {
	Construction
	error
}

// NewError creates a Error.
func NewError(Kind Kind, err error) Error {
	return &treeError{kind: Kind, error: err}
}

// treeError is a error found in the parsing.
type treeError struct {
	// kind is the kind of the error.
	kind Kind
	error
}

// Kind implements COnstruction.
func (te *treeError) Kind() Kind {
	return te.kind
}
