package parser

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/joaobnv/mel/sqlite/v3_46_1/parsetree"
)

// treeKinds maps a identifier, in the language of specifying the parse tree, to your corresponding
// kind.
var treeKinds = map[string]parsetree.Kind{}

// init initializes treeKinds.
func init() {
	for i := parsetree.KindAdd; i <= parsetree.KindWithClause; i++ {
		treeKinds[i.String()] = i
	}
	treeKinds["S"] = parsetree.KindSQLStatement
	treeKinds["E"] = parsetree.KindExpression
	treeKinds["ParenE"] = parsetree.KindParenExpression
	treeKinds["ColConstr"] = parsetree.KindColumnConstraint
	treeKinds["ColDef"] = parsetree.KindColumnDefinition
	treeKinds["ColRef"] = parsetree.KindColumnReference
	treeKinds["ColName"] = parsetree.KindColumnName
	treeKinds["FnName"] = parsetree.KindFunctionName
	treeKinds["FnCall"] = parsetree.KindFunctionCall
	treeKinds["FnArgs"] = parsetree.KindFunctionArguments
	treeKinds["JoinOp"] = parsetree.KindJoinOperator
	treeKinds["JoinConstr"] = parsetree.KindJoinConstraint
}

func toParsetree(code string) (parsetree.Construction, error) {
	var w wal
	return w.toParsetree(code)
}

type walTree struct {
	father      *walTree
	id          string
	nonTerminal bool
	open        rune
	children    []*walTree
	close       rune
}

func (wt *walTree) String() string {
	var b strings.Builder
	b.WriteString(wt.id)
	if wt.open != 0 {
		b.WriteRune(wt.open)
	}
	for i, c := range wt.children {
		if i > 0 {
			b.WriteRune(' ')
		}
		b.WriteString(c.String())
	}
	if wt.close != 0 {
		b.WriteRune(wt.close)
	}
	return b.String()
}

type wal struct {
	tokens []string
}

func (w *wal) toParsetree(code string) (parsetree.Construction, error) {
	wt, err := w.parse(code)
	if err != nil {
		return nil, err
	}
	return w.convert(wt)
}

func (w *wal) convert(wt *walTree) (parsetree.Construction, error) {
	if wt.id == "@" {
		return parsetree.NewTerminal(parsetree.KindToken, nil), nil
	}
	if wt.nonTerminal {
		k, err := w.kind(wt.id)
		if err != nil {
			return nil, err
		}
		nt := parsetree.NewNonTerminal(k)
		for _, c := range wt.children {
			pc, err := w.convert(c)
			if err != nil {
				return nil, err
			}
			nt.AddChild(pc)
		}
		return nt, nil
	}

	k, err := w.kind(wt.id)
	return parsetree.NewTerminal(k, nil), err
}

func (w *wal) kind(id string) (parsetree.Kind, error) {
	k, ok := treeKinds[id]
	if !ok {
		return k, fmt.Errorf("unknown kind: %s", id)
	}
	return k, nil
}

func (w *wal) parse(code string) (*walTree, error) {
	var result *walTree
	var fatherStack []*walTree
	var father *walTree
	var lastID *walTree

	if err := w.scan(code); err != nil {
		return nil, err
	}

	for _, tok := range w.tokens {
		switch tok {
		default:
			lastID = &walTree{id: tok}
			if father == nil && result != nil {
				return result, w.createParseError(`unexpected "`+tok+`"`, code)
			}
			if father == nil {
				result = lastID
			} else {
				father.children = append(father.children, lastID)
			}
		case "{":
			if lastID == nil {
				return result, w.createParseError(`unexpected "{"`, code)
			}
			fatherStack = append(fatherStack, lastID)
			father = lastID
			lastID.nonTerminal = true
		case "}":
			if father == nil {
				return result, w.createParseError(`unexpected "}"`, code)
			}
			fatherStack = fatherStack[:len(fatherStack)-1]
			if len(fatherStack) == 0 {
				father = nil
			} else {
				father = fatherStack[len(fatherStack)-1]
			}
		case "@":
			if father == nil {
				return result, w.createParseError(`unexpected "T"`, code)
			}
			t := &walTree{id: tok}
			father.children = append(father.children, t)
		}
	}

	if father != nil {
		return result, w.createParseError("syntax error: father.id = "+father.id, code)
	}

	return result, nil
}

func (w *wal) createParseError(msg string, code string) error {
	var (
		wdp walDebugParser
		wd  walDebugger
		b   strings.Builder
	)

	wt := wdp.parse(w)
	wd.print(&b, wt, 100)
	return &walParseError{msg: msg, debugString: b.String()}
}

func (w *wal) scan(code string) error {
	var state int
	const (
		stateNormal int = iota
		stateFirstT
		stateTokens
		stateKind
	)
	var kind strings.Builder
	for _, r := range code {
		if r == '{' {
			if state == stateKind {
				w.tokens = append(w.tokens, kind.String())
				kind.Reset()
			} else if state == stateFirstT {
				w.tokens = append(w.tokens, "@")
			}
			state = stateNormal
			w.tokens = append(w.tokens, "{")
		} else if r == '}' {
			if state == stateKind {
				w.tokens = append(w.tokens, kind.String())
				kind.Reset()
			} else if state == stateFirstT {
				w.tokens = append(w.tokens, "@")
				kind.Reset()
			}
			state = stateNormal
			w.tokens = append(w.tokens, "}")
		} else if r == 'T' {
			if state == stateNormal {
				state = stateFirstT
				kind.WriteRune(r)
			} else if state == stateFirstT {
				state = stateTokens
				kind.Reset()
				w.tokens = append(w.tokens, "@")
				w.tokens = append(w.tokens, "@")
			} else if state == stateTokens {
				w.tokens = append(w.tokens, "@")
			} else if state == stateKind {
				kind.WriteRune(r)
			}
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if state == stateNormal {
				state = stateKind
				kind.WriteRune(r)
			} else if state == stateFirstT {
				state = stateKind
				kind.WriteRune(r)
			} else if state == stateTokens {
				return fmt.Errorf("unexpected %q", r)
			} else if state == stateKind {
				kind.WriteRune(r)
			}
		} else if unicode.IsSpace(r) {
			if state == stateFirstT {
				w.tokens = append(w.tokens, "@")
				kind.Reset()
			} else if state == stateKind {
				w.tokens = append(w.tokens, kind.String())
				kind.Reset()
			}
			state = stateNormal
		} else {
			return fmt.Errorf("unexpected %q", r)
		}
	}

	if state == stateKind {
		w.tokens = append(w.tokens, kind.String())
	} else if state == stateFirstT {
		w.tokens = append(w.tokens, "@")
	}

	return nil
}

type walParseError struct {
	msg         string
	debugString string
}

func (we *walParseError) Error() string {
	return we.msg + "\n" + we.debugString
}

type walDebugger struct {
	a *walDebugArea
}

func (wd *walDebugger) print(b *strings.Builder, wt *walTree, width int) {
	wd.a = newWalDebugArea(b, width)
	if wd.a.fits(wd.lenOneLine(wt)) {
		wd.printOneLine(wt)
	} else {
		wd.printMultiline(wt)
	}
	wd.a.flush()
}

func (wd *walDebugger) lenOneLine(wt *walTree) int {
	var r int
	if len(wt.children) == 0 {
		r = utf8.RuneCountInString(wt.id)
		if wt.open != 0 {
			r += 2
		}
		return r
	}

	r = utf8.RuneCountInString(wt.id)
	if len(wt.children) == 1 {
		r += 2
	}

	for i, child := range wt.children {
		if i > 0 {
			r += 1
		}
		r += wd.lenOneLine(child)
	}

	if wt.close != 0 {
		r += 2
	}
	return r
}

func (wd *walDebugger) printOneLine(wt *walTree) {
	wd.a.writeID(wt)
	wd.printOpen(wt)

	if len(wt.children) == 1 {
		wd.a.writeSpace()
	}

	for i, child := range wt.children {
		if i > 0 {
			wd.a.writeSpace()
		}
		wd.printOneLine(child)
	}

	if len(wt.children) == 1 {
		wd.a.writeSpace()
	}

	wd.printClose(wt)
}

func (wd *walDebugger) printOpen(wt *walTree) {
	if wt.open != 0 {
		wd.a.writeOpen(wt)
	}
}

func (wd *walDebugger) printClose(wt *walTree) {
	if wt.close != 0 {
		wd.a.writeClose(wt)
	}
}

func (wd *walDebugger) printMultiline(wt *walTree) {
	if wd.isRoot(wt) && len(wt.children) == 0 {
		wd.a.writeID(wt)
		wd.printOpen(wt)
		wd.printClose(wt)
		return
	}

	if wd.isRoot(wt) {
		wd.a.writeID(wt)
		wd.printOpen(wt)
		wd.printChildren(wt)
		wd.printClose(wt)
		return
	}

	headLen := utf8.RuneCountInString(wt.id) + 1
	if wd.isOnlyChild(wt) && len(wt.children) > 0 && wd.a.fits(headLen) {
		wd.a.writeID(wt)
		wd.printOpen(wt)
		wd.printChildren(wt)
		wd.printClose(wt)
		return
	}

	if wd.isFirstChild(wt) {
		wd.a.writeNewLine()
	}
	wd.a.writeID(wt)
	wd.printOpen(wt)
	wd.printChildren(wt)
	wd.printClose(wt)
}

func (td walDebugger) isRoot(wt *walTree) bool {
	return wt.father == nil
}

func (td walDebugger) isOnlyChild(wt *walTree) bool {
	return wt.father != nil && len(wt.father.children) == 1
}

func (td walDebugger) isFirstChild(wt *walTree) bool {
	return wt.father != nil && wt.father.children[0] == wt
}

func (td walDebugger) isLastChild(wt *walTree) bool {
	if wt.father == nil {
		return false
	}
	i := len(wt.father.children) - 1
	return wt.father.children[i] == wt
}

func (wd *walDebugger) printChildren(wt *walTree) {
	for _, child := range wt.children {
		wd.printChild(child)
		if !wd.isLastChild(child) {
			wd.a.flushCloses()
			wd.a.writeNewLine()
		}
	}
}

func (wd *walDebugger) printChild(wt *walTree) {
	if wd.a.fitsIfNewLine(wd.lenOneLine(wt)) && (!wd.isOnlyChild(wt) || len(wt.children) > 1) {
		if wd.isFirstChild(wt) {
			wd.a.writeNewLine()
		}
		wd.printOneLine(wt)
	} else {
		wd.printMultiline(wt)
	}
}

type closeState int

const (
	closeStateNil closeState = iota
	closeStateFirstCloseWithMultiline
	closeStateFirstCloseWithNotMultiline
	closeStateAppend
)

type walDebugArea struct {
	col            int
	width          int
	prev           [2]string
	childMultiline []bool
	indent         int
	cs             closeState
	closes         []rune
	b              *strings.Builder
}

func newWalDebugArea(b *strings.Builder, width int) *walDebugArea {
	return &walDebugArea{
		col: 1, width: width, childMultiline: []bool{false},
		b: b,
	}
}

func (wda *walDebugArea) fits(n int) bool {
	switch wda.prev[0] {
	case "", "OPEN":
		return wda.col-1+n <= wda.width
	default:
		return 2*wda.indent+n <= wda.width
	}
}

func (wda *walDebugArea) fitsIfNewLine(n int) bool {
	return 2*wda.indent+n <= wda.width
}

func (wda *walDebugArea) writeID(wt *walTree) {
	wda.cs = closeStateNil
	if wda.prev[0] == "OPEN" || (wda.prev[0] == "SPACE" && wda.prev[1] == "OPEN") {
		wda.childMultiline = append(wda.childMultiline, false)
	} else if wda.prev[0] == "NL" {
		wda.writeIndent()
	} else if wda.prev[0] == "CLOSE" {
		if wda.topChildMultiline() {
			wda.writeNewLine()
			wda.writeIndent()
		}
	}
	wda.b.WriteString(wt.id)
	wda.col += utf8.RuneCountInString(wt.id)
	wda.putPrev("ID")
}

func (wda *walDebugArea) writeOpen(wt *walTree) {
	wda.b.WriteRune(wt.open)
	wda.indent++
	wda.col++
	wda.putPrev("OPEN")
}

func (wda *walDebugArea) writeNewLine() {
	wda.childMultiline = append(wda.childMultiline, true)
	wda.b.WriteByte('\n')
	wda.col = 1
	wda.putPrev("NL")
}

func (wda *walDebugArea) writeClose(wt *walTree) {
	var cm bool
	if wda.prev[0] != "OPEN" {
		cm = wda.popChildMultiline()
	}
	wda.indent--
	switch wda.cs {
	case closeStateNil:
		if cm {
			wda.closes = append(wda.closes, '\n', wt.close)
			wda.cs = closeStateFirstCloseWithMultiline
		} else {
			wda.b.WriteRune(wt.close)
			wda.cs = closeStateFirstCloseWithNotMultiline
		}
	case closeStateFirstCloseWithMultiline:
		wda.closes = append(wda.closes, wt.close)
		wda.cs = closeStateAppend
	case closeStateFirstCloseWithNotMultiline:
		if cm {
			wda.closes = append(wda.closes, '\n', wt.close)
			wda.cs = closeStateFirstCloseWithMultiline
		} else {
			wda.b.WriteRune(wt.close)
		}
	case closeStateAppend:
		wda.closes = append(wda.closes, wt.close)
	}
	wda.putPrev("CLOSE")
}

func (wda *walDebugArea) topChildMultiline() bool {
	return wda.childMultiline[len(wda.childMultiline)-1]
}

func (wda *walDebugArea) popChildMultiline() bool {
	cm := wda.childMultiline[len(wda.childMultiline)-1]
	wda.childMultiline = wda.childMultiline[:len(wda.childMultiline)-1]
	return cm
}

func (wda *walDebugArea) flushCloses() {
	for _, r := range wda.closes {
		if r == '\n' {
			wda.writeNewLine()
			wda.writeIndent()
		} else {
			wda.b.WriteRune(r)
		}
	}
	if len(wda.closes) > 0 {
		wda.closes = wda.closes[:0]
		wda.putPrev("CLOSE")
	}
}

func (wda *walDebugArea) writeIndent() {
	wda.b.WriteString(strings.Repeat(". ", wda.indent))
	wda.col += wda.indent * 2
}

func (wda *walDebugArea) writeSpace() {
	if wda.cs == closeStateNil || wda.cs == closeStateFirstCloseWithNotMultiline {
		wda.b.WriteByte(' ')
		wda.col++
	} else {
		wda.closes = append(wda.closes, ' ')
	}
	wda.putPrev("SPACE")
}

func (wda *walDebugArea) flush() {
	wda.flushCloses()
}

func (wda *walDebugArea) putPrev(p string) {
	wda.prev[1] = wda.prev[0]
	wda.prev[0] = p
}

type walDebugParser struct {
	stack   []*walTree
	top     *walTree
	scanner *walDebugScanner
}

func (wdp *walDebugParser) parse(w *wal) *walTree {
	wdp.scanner = newWalDebugScanner(w)

	tok, eof := wdp.scanner.peek()
	if eof {
		return &walTree{}
	}

	var trees []*walTree

	for {
		if wdp.isId(tok) || tok == "{" || tok == "@" {
			trees = append(trees, wdp.tree())
		} else {
			wt := &walTree{open: '[', children: slices.Clone(trees), close: '}'}
			for _, child := range wt.children {
				child.father = wt
			}
			trees = []*walTree{wt}
			wdp.scanner.next()
		}

		tok, eof = wdp.scanner.peek()
		if eof {
			break
		}
	}

	if len(trees) == 1 {
		return trees[0]
	}

	wt := &walTree{open: '[', children: slices.Clone(trees), close: ']'}
	for _, child := range wt.children {
		child.father = wt
	}
	return wt
}

func (wdp *walDebugParser) tree() *walTree {
	tok, _ := wdp.scanner.peek()
	if wdp.isId(tok) {
		wdp.push(tok)
		wdp.scanner.next()
	} else if tok == "{" {
		wdp.push("#")
	} else {
		wdp.push("T")
		wdp.scanner.next()
		return wdp.pop()
	}

	tok, eof := wdp.scanner.peek()
	if eof {
		return wdp.pop()
	}

	if tok != "{" {
		return wdp.pop()
	}

	wdp.scanner.next()
	wdp.top.open = '{'

	tok, eof = wdp.scanner.peek()
	if eof {
		wdp.top.close = ']'
		return wdp.pop()
	}
	if tok == "}" {
		wdp.scanner.next()
		wdp.top.close = '}'
		return wdp.pop()
	}

	for wdp.isId(tok) || tok == "{" || tok == "@" {
		wdp.addChild(wdp.tree())
		tok, eof = wdp.scanner.peek()
		if eof {
			wdp.top.close = ']'
			return wdp.pop()
		}
	}

	wdp.scanner.next()
	wdp.top.close = '}'
	return wdp.pop()
}

func (wdp *walDebugParser) push(id string) {
	t := &walTree{id: id}
	wdp.stack = append(wdp.stack, t)
	wdp.top = t
}

func (wdp *walDebugParser) pop() *walTree {
	t := wdp.top
	wdp.stack = wdp.stack[:len(wdp.stack)-1]
	if len(wdp.stack) > 0 {
		wdp.top = wdp.stack[len(wdp.stack)-1]
	} else {
		wdp.top = nil
	}
	return t
}

func (wdp *walDebugParser) addChild(t *walTree) {
	t.father = wdp.top
	wdp.top.children = append(wdp.top.children, t)
}

func (wdp *walDebugParser) isId(tok string) bool {
	return tok != "{" && tok != "}" && tok != "@"
}

type walDebugScanner struct {
	tokens []string
	pos    int
}

func newWalDebugScanner(w *wal) *walDebugScanner {
	return &walDebugScanner{tokens: w.tokens}
}

func (wds *walDebugScanner) next() (tok string, eof bool) {
	if wds.pos == len(wds.tokens) {
		return tok, true
	}
	tok = wds.tokens[wds.pos]
	wds.pos++
	return
}

func (wds *walDebugScanner) peek() (tok string, eof bool) {
	if wds.pos == len(wds.tokens) {
		return tok, true
	}
	tok = wds.tokens[wds.pos]
	return
}
