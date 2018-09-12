//line build/parse.y:13
package build

import __yyfmt__ "fmt"

//line build/parse.y:13
//line build/parse.y:18
type yySymType struct {
	yys int
	// input tokens
	tok    string   // raw input syntax
	str    string   // decoding of quoted string
	pos    Position // position of token
	triple bool     // was string triple quoted?

	// partial syntax trees
	expr    Expr
	exprs   []Expr
	string  *StringExpr
	strings []*StringExpr
	ifstmt  *IfStmt
	loadarg *struct {
		from Ident
		to   Ident
	}
	loadargs []*struct {
		from Ident
		to   Ident
	}

	// supporting information
	comma    Position // position of trailing comma in list, if present
	lastStmt Expr     // most recent rule, to attach line comments to
}

const _AUGM = 57346
const _AND = 57347
const _COMMENT = 57348
const _EOF = 57349
const _EQ = 57350
const _FOR = 57351
const _GE = 57352
const _IDENT = 57353
const _NUMBER = 57354
const _IF = 57355
const _ELSE = 57356
const _ELIF = 57357
const _IN = 57358
const _IS = 57359
const _LAMBDA = 57360
const _LOAD = 57361
const _LE = 57362
const _NE = 57363
const _STAR_STAR = 57364
const _NOT = 57365
const _OR = 57366
const _STRING = 57367
const _DEF = 57368
const _RETURN = 57369
const _INDENT = 57370
const _UNINDENT = 57371
const ShiftInstead = 57372
const _ASSERT = 57373
const _UNARY = 57374

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"'%'",
	"'('",
	"')'",
	"'*'",
	"'+'",
	"','",
	"'-'",
	"'.'",
	"'/'",
	"':'",
	"'<'",
	"'='",
	"'>'",
	"'['",
	"']'",
	"'{'",
	"'}'",
	"'|'",
	"_AUGM",
	"_AND",
	"_COMMENT",
	"_EOF",
	"_EQ",
	"_FOR",
	"_GE",
	"_IDENT",
	"_NUMBER",
	"_IF",
	"_ELSE",
	"_ELIF",
	"_IN",
	"_IS",
	"_LAMBDA",
	"_LOAD",
	"_LE",
	"_NE",
	"_STAR_STAR",
	"_NOT",
	"_OR",
	"_STRING",
	"_DEF",
	"_RETURN",
	"_INDENT",
	"_UNINDENT",
	"ShiftInstead",
	"'\\n'",
	"_ASSERT",
	"_UNARY",
	"';'",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line build/parse.y:921

// Go helper code.

// unary returns a unary expression with the given
// position, operator, and subexpression.
func unary(pos Position, op string, x Expr) Expr {
	return &UnaryExpr{
		OpStart: pos,
		Op:      op,
		X:       x,
	}
}

// binary returns a binary expression with the given
// operands, position, and operator.
func binary(x Expr, pos Position, op string, y Expr) Expr {
	_, xend := x.Span()
	ystart, _ := y.Span()
	return &BinaryExpr{
		X:         x,
		OpStart:   pos,
		Op:        op,
		LineBreak: xend.Line < ystart.Line,
		Y:         y,
	}
}

// isSimpleExpression returns whether an expression is simple and allowed to exist in
// compact forms of sequences.
// The formal criteria are the following: an expression is considered simple if it's
// a literal (variable, string or a number), a literal with a unary operator or an empty sequence.
func isSimpleExpression(expr *Expr) bool {
	switch x := (*expr).(type) {
	case *LiteralExpr, *StringExpr, *Ident:
		return true
	case *UnaryExpr:
		_, literal := x.X.(*LiteralExpr)
		_, ident := x.X.(*Ident)
		return literal || ident
	case *ListExpr:
		return len(x.List) == 0
	case *TupleExpr:
		return len(x.List) == 0
	case *DictExpr:
		return len(x.List) == 0
	case *SetExpr:
		return len(x.List) == 0
	default:
		return false
	}
}

// forceCompact returns the setting for the ForceCompact field for a call or tuple.
//
// NOTE 1: The field is called ForceCompact, not ForceSingleLine,
// because it only affects the formatting associated with the call or tuple syntax,
// not the formatting of the arguments. For example:
//
//	call([
//		1,
//		2,
//		3,
//	])
//
// is still a compact call even though it runs on multiple lines.
//
// In contrast the multiline form puts a linebreak after the (.
//
//	call(
//		[
//			1,
//			2,
//			3,
//		],
//	)
//
// NOTE 2: Because of NOTE 1, we cannot use start and end on the
// same line as a signal for compact mode: the formatting of an
// embedded list might move the end to a different line, which would
// then look different on rereading and cause buildifier not to be
// idempotent. Instead, we have to look at properties guaranteed
// to be preserved by the reformatting, namely that the opening
// paren and the first expression are on the same line and that
// each subsequent expression begins on the same line as the last
// one ended (no line breaks after comma).
func forceCompact(start Position, list []Expr, end Position) bool {
	if len(list) <= 1 {
		// The call or tuple will probably be compact anyway; don't force it.
		return false
	}

	// If there are any named arguments or non-string, non-literal
	// arguments, cannot force compact mode.
	line := start.Line
	for _, x := range list {
		start, end := x.Span()
		if start.Line != line {
			return false
		}
		line = end.Line
		if !isSimpleExpression(&x) {
			return false
		}
	}
	return end.Line == line
}

// forceMultiLine returns the setting for the ForceMultiLine field.
func forceMultiLine(start Position, list []Expr, end Position) bool {
	if len(list) > 1 {
		// The call will be multiline anyway, because it has multiple elements. Don't force it.
		return false
	}

	if len(list) == 0 {
		// Empty list: use position of brackets.
		return start.Line != end.Line
	}

	// Single-element list.
	// Check whether opening bracket is on different line than beginning of
	// element, or closing bracket is on different line than end of element.
	elemStart, elemEnd := list[0].Span()
	return start.Line != elemStart.Line || end.Line != elemEnd.Line
}

// forceMultiLineComprehension returns the setting for the ForceMultiLine field for a comprehension.
func forceMultiLineComprehension(start Position, expr Expr, clauses []Expr, end Position) bool {
	// Return true if there's at least one line break between start, expr, each clause, and end
	exprStart, exprEnd := expr.Span()
	if start.Line != exprStart.Line {
		return true
	}
	previousEnd := exprEnd
	for _, clause := range clauses {
		clauseStart, clauseEnd := clause.Span()
		if previousEnd.Line != clauseStart.Line {
			return true
		}
		previousEnd = clauseEnd
	}
	return previousEnd.Line != end.Line
}

// extractTrailingComments extracts trailing comments of an indented block starting with the first
// comment line with indentation less than the block indentation.
// The comments can either belong to CommentBlock statements or to the last non-comment statement
// as After-comments.
func extractTrailingComments(stmt Expr) []Expr {
	body := getLastBody(stmt)
	var comments []Expr
	if body != nil && len(*body) > 0 {
		// Get the current indentation level
		start, _ := (*body)[0].Span()
		indentation := start.LineRune

		// Find the last non-comment statement
		lastNonCommentIndex := -1
		for i, stmt := range *body {
			if _, ok := stmt.(*CommentBlock); !ok {
				lastNonCommentIndex = i
			}
		}
		if lastNonCommentIndex == -1 {
			return comments
		}

		// Iterate over the trailing comments, find the first comment line that's not indented enough,
		// dedent it and all the following comments.
		for i := lastNonCommentIndex; i < len(*body); i++ {
			stmt := (*body)[i]
			if comment := extractDedentedComment(stmt, indentation); comment != nil {
				// This comment and all the following CommentBlock statements are to be extracted.
				comments = append(comments, comment)
				comments = append(comments, (*body)[i+1:]...)
				*body = (*body)[:i+1]
				// If the current statement is a CommentBlock statement without any comment lines
				// it should be removed too.
				if i > lastNonCommentIndex && len(stmt.Comment().After) == 0 {
					*body = (*body)[:i]
				}
			}
		}
	}
	return comments
}

// extractDedentedComment extract the first comment line from `stmt` which indentation is smaller
// than `indentation`, and all following comment lines, and returns them in a newly created
// CommentBlock statement.
func extractDedentedComment(stmt Expr, indentation int) Expr {
	for i, line := range stmt.Comment().After {
		// line.Start.LineRune == 0 can't exist in parsed files, it indicates that the comment line
		// has been added by an AST modification. Don't take such lines into account.
		if line.Start.LineRune > 0 && line.Start.LineRune < indentation {
			// This and all the following lines should be dedented
			cb := &CommentBlock{
				Start:    line.Start,
				Comments: Comments{After: stmt.Comment().After[i:]},
			}
			stmt.Comment().After = stmt.Comment().After[:i]
			return cb
		}
	}
	return nil
}

// getLastBody returns the last body of a block statement (the only body for For- and DefStmt
// objects, the last in a if-elif-else chain
func getLastBody(stmt Expr) *[]Expr {
	switch block := stmt.(type) {
	case *DefStmt:
		return &block.Body
	case *ForStmt:
		return &block.Body
	case *IfStmt:
		if len(block.False) == 0 {
			return &block.True
		} else if len(block.False) == 1 {
			if next, ok := block.False[0].(*IfStmt); ok {
				// Recursively find the last block of the chain
				return getLastBody(next)
			}
		}
		return &block.False
	}
	return nil
}

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyPrivate = 57344

const yyLast = 632

var yyAct = [...]int{

	16, 194, 22, 154, 31, 191, 7, 2, 145, 126,
	36, 130, 77, 117, 9, 129, 85, 152, 204, 215,
	206, 70, 71, 18, 141, 33, 32, 75, 80, 83,
	73, 29, 107, 43, 44, 37, 147, 171, 29, 88,
	203, 91, 114, 205, 35, 32, 95, 96, 97, 98,
	99, 100, 101, 102, 103, 104, 105, 106, 29, 108,
	109, 110, 111, 112, 198, 93, 118, 28, 119, 148,
	13, 174, 21, 169, 132, 60, 88, 127, 132, 26,
	137, 27, 132, 94, 136, 41, 134, 40, 63, 146,
	69, 29, 30, 135, 221, 74, 164, 208, 19, 24,
	142, 87, 207, 20, 28, 32, 128, 14, 155, 89,
	90, 153, 150, 92, 199, 211, 26, 181, 27, 40,
	161, 162, 163, 79, 82, 38, 159, 46, 29, 30,
	45, 157, 39, 165, 173, 47, 24, 122, 124, 175,
	177, 170, 32, 172, 60, 40, 210, 170, 180, 184,
	182, 183, 176, 65, 179, 168, 37, 188, 149, 64,
	40, 118, 190, 119, 151, 66, 192, 160, 40, 185,
	139, 196, 197, 195, 189, 40, 133, 125, 217, 113,
	178, 158, 201, 146, 46, 140, 86, 45, 48, 200,
	49, 72, 47, 84, 187, 166, 167, 1, 193, 212,
	186, 60, 202, 209, 25, 81, 78, 34, 42, 192,
	15, 214, 218, 196, 216, 195, 219, 213, 7, 28,
	12, 8, 4, 23, 21, 131, 67, 68, 76, 123,
	143, 26, 144, 27, 115, 116, 0, 0, 6, 0,
	0, 11, 0, 29, 30, 17, 0, 0, 28, 0,
	19, 24, 0, 21, 0, 20, 0, 32, 10, 14,
	26, 220, 27, 5, 0, 0, 0, 6, 3, 0,
	11, 0, 29, 30, 17, 0, 0, 0, 0, 19,
	24, 0, 0, 0, 20, 0, 32, 10, 14, 0,
	46, 0, 5, 45, 48, 0, 49, 0, 47, 138,
	50, 0, 51, 0, 0, 0, 0, 60, 0, 59,
	0, 0, 52, 0, 55, 0, 0, 62, 0, 0,
	56, 61, 0, 0, 53, 54, 46, 57, 58, 45,
	48, 0, 49, 0, 47, 0, 50, 0, 51, 0,
	0, 0, 0, 60, 0, 59, 0, 0, 52, 0,
	55, 0, 0, 62, 156, 0, 56, 61, 0, 0,
	53, 54, 46, 57, 58, 45, 48, 0, 49, 0,
	47, 0, 50, 0, 51, 0, 0, 0, 0, 60,
	0, 59, 0, 0, 52, 132, 55, 0, 0, 62,
	0, 0, 56, 61, 0, 0, 53, 54, 46, 57,
	58, 45, 48, 0, 49, 0, 47, 0, 50, 0,
	51, 0, 0, 0, 0, 60, 0, 59, 0, 0,
	52, 0, 55, 0, 0, 62, 0, 0, 56, 61,
	0, 0, 53, 54, 46, 57, 58, 45, 48, 0,
	49, 0, 47, 0, 50, 0, 51, 0, 0, 0,
	0, 60, 0, 59, 0, 0, 52, 0, 55, 0,
	0, 0, 0, 0, 56, 61, 0, 0, 53, 54,
	46, 57, 58, 45, 48, 0, 49, 0, 47, 0,
	50, 0, 51, 0, 0, 0, 0, 60, 0, 59,
	0, 0, 52, 0, 55, 0, 0, 0, 0, 0,
	56, 0, 0, 0, 53, 54, 46, 57, 58, 45,
	48, 0, 49, 0, 47, 0, 50, 0, 51, 0,
	0, 28, 0, 60, 0, 59, 21, 0, 52, 0,
	55, 0, 0, 26, 0, 27, 56, 0, 0, 0,
	53, 54, 28, 57, 120, 29, 30, 21, 0, 0,
	0, 0, 19, 24, 26, 0, 27, 20, 0, 32,
	0, 14, 0, 0, 0, 0, 29, 30, 0, 0,
	0, 0, 0, 19, 24, 0, 0, 121, 20, 46,
	32, 0, 45, 48, 0, 49, 0, 47, 0, 50,
	0, 51, 0, 28, 0, 0, 60, 0, 21, 0,
	0, 52, 0, 55, 0, 26, 0, 27, 0, 56,
	0, 0, 0, 53, 54, 0, 57, 29, 30, 0,
	0, 0, 0, 0, 19, 24, 0, 0, 0, 20,
	0, 32,
}
var yyPact = [...]int{

	-1000, -1000, 243, -1000, -1000, -1000, -24, -1000, -1000, -1000,
	15, 99, -1000, 110, 588, 1, 394, 588, 148, 588,
	588, 588, -1000, -1000, 186, -17, 588, 588, 588, -1000,
	-1000, -1000, -1000, -1000, -36, 181, 67, 148, 588, 588,
	588, 159, 588, 52, -1000, 588, 588, 588, 588, 588,
	588, 588, 588, 588, 588, 588, 588, -2, 588, 588,
	588, 588, 588, 166, 13, 537, 588, 125, 168, 159,
	-1000, -1000, -17, -1000, 88, 358, 167, 47, 64, 60,
	286, 161, 179, 394, -25, 516, 29, 588, 99, 159,
	159, 430, 151, 62, -1000, 54, 54, 54, 123, 123,
	180, 180, 180, 180, 180, 180, 180, 588, 502, 575,
	-1000, 466, 322, 62, -1000, 175, 158, -1000, 394, 105,
	588, 588, 78, 120, 588, 588, -1000, 146, -1000, 55,
	6, -1000, 99, 588, -1000, 51, -1000, -1000, 588, 588,
	-1000, -1000, -1000, 174, 139, -1000, 102, 9, 9, 136,
	148, 62, -1000, -1000, -1000, 180, 588, -1000, -1000, -1000,
	537, 588, 394, 394, -1000, 588, -1000, -1000, 2, -1000,
	6, 588, 30, 394, -1000, 394, -1000, 286, 101, -1000,
	29, 588, -1000, -1000, 62, -1000, -6, -29, 430, -1000,
	394, 84, 394, 137, -1000, -1000, 100, 430, 588, 62,
	-1000, 394, -1000, -1000, -30, -1000, -1000, -1000, 588, 172,
	2, -17, 430, -1000, 214, -1000, 76, -1000, -1000, -1000,
	-1000, -1000,
}
var yyPgo = [...]int{

	0, 9, 13, 235, 234, 8, 232, 230, 0, 5,
	95, 23, 70, 229, 228, 227, 226, 10, 225, 11,
	15, 2, 223, 7, 222, 221, 220, 210, 208, 3,
	14, 207, 12, 206, 205, 4, 204, 17, 200, 1,
	198, 197, 194, 193,
}
var yyR1 = [...]int{

	0, 41, 37, 37, 42, 42, 38, 38, 38, 23,
	23, 23, 23, 24, 24, 25, 25, 25, 27, 27,
	26, 26, 28, 28, 29, 31, 31, 30, 30, 30,
	30, 30, 43, 43, 11, 11, 11, 11, 11, 11,
	11, 11, 11, 11, 11, 11, 11, 11, 11, 4,
	4, 3, 3, 2, 2, 2, 2, 40, 40, 39,
	39, 7, 7, 6, 6, 5, 5, 5, 5, 12,
	12, 13, 13, 15, 15, 16, 16, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8, 8, 14,
	14, 9, 9, 10, 10, 1, 1, 32, 34, 34,
	33, 33, 17, 17, 35, 36, 36, 21, 22, 18,
	19, 19, 20, 20,
}
var yyR2 = [...]int{

	0, 2, 5, 2, 0, 2, 0, 3, 2, 0,
	2, 2, 3, 1, 1, 7, 6, 1, 4, 5,
	1, 4, 2, 1, 4, 0, 3, 1, 2, 1,
	3, 3, 0, 1, 1, 1, 3, 7, 4, 4,
	6, 8, 1, 3, 4, 4, 3, 3, 3, 0,
	2, 1, 3, 1, 3, 2, 2, 1, 3, 1,
	3, 0, 2, 1, 3, 1, 3, 2, 2, 1,
	3, 0, 1, 1, 3, 0, 2, 1, 4, 2,
	2, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 4, 3, 3, 3, 3, 5, 1,
	3, 0, 1, 0, 2, 0, 1, 3, 1, 3,
	1, 2, 1, 3, 1, 1, 2, 1, 1, 4,
	1, 3, 1, 2,
}
var yyChk = [...]int{

	-1000, -41, -23, 25, -24, 49, 24, -29, -25, -30,
	44, 27, -26, -12, 45, -27, -8, 31, -11, 36,
	41, 10, -21, -22, 37, -36, 17, 19, 5, 29,
	30, -35, 43, 49, -31, 29, -17, -11, 15, 22,
	9, -12, -28, 32, 33, 7, 4, 12, 8, 10,
	14, 16, 26, 38, 39, 28, 34, 41, 42, 23,
	21, 35, 31, -12, 11, 5, 17, -16, -15, -12,
	-8, -8, 5, -35, -10, -8, -14, -32, -33, -10,
	-8, -34, -10, -8, -43, 52, 5, 34, 9, -12,
	-12, -8, -12, 13, 31, -8, -8, -8, -8, -8,
	-8, -8, -8, -8, -8, -8, -8, 34, -8, -8,
	-8, -8, -8, 13, 29, -4, -3, -2, -8, -21,
	7, 40, -12, -13, 13, 9, -1, -35, 18, -20,
	-19, -18, 27, 9, -1, -20, 20, 20, 13, 9,
	6, 49, -30, -7, -6, -5, -21, 7, 40, -12,
	-11, 13, -37, 49, -29, -8, 32, -37, 6, -1,
	9, 15, -8, -8, 18, 13, -12, -12, 9, 18,
	-19, 31, -17, -8, 20, -8, -32, -8, 6, -1,
	9, 15, -21, -21, 13, -37, -38, -42, -8, -2,
	-8, -9, -8, -40, -39, -35, -21, -8, 34, 13,
	-5, -8, -37, 46, 24, 49, 49, 18, 13, -1,
	9, 15, -8, -37, -23, 49, -9, 6, -39, -35,
	47, 18,
}
var yyDef = [...]int{

	9, -2, 0, 1, 10, 11, 0, 13, 14, 25,
	0, 0, 17, 27, 29, 20, 69, 0, 77, 75,
	0, 0, 34, 35, 0, 42, 103, 103, 103, 117,
	118, 115, 114, 12, 32, 0, 0, 112, 0, 0,
	0, 28, 0, 0, 23, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 49, 71, 0, 105, 73,
	79, 80, 0, 116, 0, 99, 105, 108, 0, 0,
	99, 110, 0, 99, 0, 33, 61, 0, 0, 30,
	31, 70, 0, 0, 22, 81, 82, 83, 84, 85,
	86, 87, 88, 89, 90, 91, 92, 0, 94, 95,
	96, 97, 0, 0, 36, 0, 105, 51, 53, 34,
	0, 0, 72, 0, 0, 106, 76, 0, 43, 0,
	122, 120, 0, 106, 104, 0, 46, 47, 0, 111,
	48, 24, 26, 0, 105, 63, 65, 0, 0, 0,
	113, 0, 21, 6, 4, 93, 0, 18, 38, 50,
	106, 0, 55, 56, 39, 101, 78, 74, 0, 44,
	123, 0, 0, 100, 45, 107, 109, 0, 0, 62,
	106, 0, 67, 68, 0, 19, 0, 3, 98, 52,
	54, 0, 102, 105, 57, 59, 0, 121, 0, 0,
	64, 66, 16, 9, 0, 8, 5, 40, 101, 0,
	106, 0, 119, 15, 0, 7, 0, 37, 58, 60,
	2, 41,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	49, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 4, 3, 3,
	5, 6, 7, 8, 9, 10, 11, 12, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 13, 52,
	14, 15, 16, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 17, 3, 18, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 19, 21, 20,
}
var yyTok2 = [...]int{

	2, 3, 22, 23, 24, 25, 26, 27, 28, 29,
	30, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 50,
	51,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:183
		{
			yylex.(*input).file = &File{Stmt: yyDollar[1].exprs}
			return 0
		}
	case 2:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line build/parse.y:190
		{
			statements := yyDollar[4].exprs
			if yyDollar[2].exprs != nil {
				// $2 can only contain *CommentBlock objects, each of them contains a non-empty After slice
				cb := yyDollar[2].exprs[len(yyDollar[2].exprs)-1].(*CommentBlock)
				// $4 can't be empty and can't start with a comment
				stmt := yyDollar[4].exprs[0]
				start, _ := stmt.Span()
				if start.Line-cb.After[len(cb.After)-1].Start.Line == 1 {
					// The first statement of $4 starts on the next line after the last comment of $2.
					// Attach the last comment to the first statement
					stmt.Comment().Before = cb.After
					yyDollar[2].exprs = yyDollar[2].exprs[:len(yyDollar[2].exprs)-1]
				}
				statements = append(yyDollar[2].exprs, yyDollar[4].exprs...)
			}
			yyVAL.exprs = statements
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 3:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:210
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 6:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:218
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:223
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].lastStmt
			if yyVAL.lastStmt == nil {
				cb := &CommentBlock{Start: yyDollar[2].pos}
				yyVAL.exprs = append(yyVAL.exprs, cb)
				yyVAL.lastStmt = cb
			}
			com := yyVAL.lastStmt.Comment()
			com.After = append(com.After, Comment{Start: yyDollar[2].pos, Token: yyDollar[2].tok})
		}
	case 8:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:235
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:241
		{
			yyVAL.exprs = nil
			yyVAL.lastStmt = nil
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:246
		{
			// If this statement follows a comment block,
			// attach the comments to the statement.
			if cb, ok := yyDollar[1].lastStmt.(*CommentBlock); ok {
				yyVAL.exprs = append(yyDollar[1].exprs[:len(yyDollar[1].exprs)-1], yyDollar[2].exprs...)
				yyDollar[2].exprs[0].Comment().Before = cb.After
				yyVAL.lastStmt = yyDollar[2].lastStmt
				break
			}

			// Otherwise add to list.
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyDollar[2].lastStmt

			// Consider this input:
			//
			//	foo()
			//	# bar
			//	baz()
			//
			// If we've just parsed baz(), the # bar is attached to
			// foo() as an After comment. Make it a Before comment
			// for baz() instead.
			if x := yyDollar[1].lastStmt; x != nil {
				com := x.Comment()
				// stmt is never empty
				yyDollar[2].exprs[0].Comment().Before = com.After
				com.After = nil
			}
		}
	case 11:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:277
		{
			// Blank line; sever last rule from future comments.
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:283
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].lastStmt
			if yyVAL.lastStmt == nil {
				cb := &CommentBlock{Start: yyDollar[2].pos}
				yyVAL.exprs = append(yyVAL.exprs, cb)
				yyVAL.lastStmt = cb
			}
			com := yyVAL.lastStmt.Comment()
			com.After = append(com.After, Comment{Start: yyDollar[2].pos, Token: yyDollar[2].tok})
		}
	case 13:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:297
		{
			yyVAL.exprs = yyDollar[1].exprs
			yyVAL.lastStmt = yyDollar[1].exprs[len(yyDollar[1].exprs)-1]
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:302
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
			yyVAL.lastStmt = yyDollar[1].expr
			if cbs := extractTrailingComments(yyDollar[1].expr); len(cbs) > 0 {
				yyVAL.exprs = append(yyVAL.exprs, cbs...)
				yyVAL.lastStmt = cbs[len(cbs)-1]
				if yyDollar[1].lastStmt == nil {
					yyVAL.lastStmt = nil
				}
			}
		}
	case 15:
		yyDollar = yyS[yypt-7 : yypt+1]
		//line build/parse.y:316
		{
			yyVAL.expr = &DefStmt{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[4].exprs,
					Body:     yyDollar[7].exprs,
				},
				Name:           yyDollar[2].tok,
				ForceCompact:   forceCompact(yyDollar[3].pos, yyDollar[4].exprs, yyDollar[5].pos),
				ForceMultiLine: forceMultiLine(yyDollar[3].pos, yyDollar[4].exprs, yyDollar[5].pos),
			}
			yyVAL.lastStmt = yyDollar[7].lastStmt
		}
	case 16:
		yyDollar = yyS[yypt-6 : yypt+1]
		//line build/parse.y:330
		{
			yyVAL.expr = &ForStmt{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				X:    yyDollar[4].expr,
				Body: yyDollar[6].exprs,
			}
			yyVAL.lastStmt = yyDollar[6].lastStmt
		}
	case 17:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:340
		{
			yyVAL.expr = yyDollar[1].ifstmt
			yyVAL.lastStmt = yyDollar[1].lastStmt
		}
	case 18:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:348
		{
			yyVAL.ifstmt = &IfStmt{
				If:   yyDollar[1].pos,
				Cond: yyDollar[2].expr,
				True: yyDollar[4].exprs,
			}
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 19:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line build/parse.y:357
		{
			yyVAL.ifstmt = yyDollar[1].ifstmt
			inner := yyDollar[1].ifstmt
			for len(inner.False) == 1 {
				inner = inner.False[0].(*IfStmt)
			}
			inner.ElsePos = End{Pos: yyDollar[2].pos}
			inner.False = []Expr{
				&IfStmt{
					If:   yyDollar[2].pos,
					Cond: yyDollar[3].expr,
					True: yyDollar[5].exprs,
				},
			}
			yyVAL.lastStmt = yyDollar[5].lastStmt
		}
	case 21:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:378
		{
			yyVAL.ifstmt = yyDollar[1].ifstmt
			inner := yyDollar[1].ifstmt
			for len(inner.False) == 1 {
				inner = inner.False[0].(*IfStmt)
			}
			inner.ElsePos = End{Pos: yyDollar[2].pos}
			inner.False = yyDollar[4].exprs
			yyVAL.lastStmt = yyDollar[4].lastStmt
		}
	case 24:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:395
		{
			yyVAL.exprs = append([]Expr{yyDollar[1].expr}, yyDollar[2].exprs...)
			yyVAL.lastStmt = yyVAL.exprs[len(yyVAL.exprs)-1]
		}
	case 25:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:401
		{
			yyVAL.exprs = []Expr{}
		}
	case 26:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:405
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 28:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:412
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
				Result: yyDollar[2].expr,
			}
		}
	case 29:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:419
		{
			yyVAL.expr = &ReturnStmt{
				Return: yyDollar[1].pos,
			}
		}
	case 30:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:424
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 31:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:425
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 36:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:434
		{
			yyVAL.expr = &DotExpr{
				X:       yyDollar[1].expr,
				Dot:     yyDollar[2].pos,
				NamePos: yyDollar[3].pos,
				Name:    yyDollar[3].tok,
			}
		}
	case 37:
		yyDollar = yyS[yypt-7 : yypt+1]
		//line build/parse.y:443
		{
			load := &LoadStmt{
				Load:         yyDollar[1].pos,
				Module:       yyDollar[3].string,
				Rparen:       End{Pos: yyDollar[7].pos},
				ForceCompact: yyDollar[1].pos.Line == yyDollar[7].pos.Line,
			}
			for _, arg := range yyDollar[5].loadargs {
				load.From = append(load.From, &arg.from)
				load.To = append(load.To, &arg.to)
			}
			yyVAL.expr = load
		}
	case 38:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:457
		{
			yyVAL.expr = &CallExpr{
				X:              yyDollar[1].expr,
				ListStart:      yyDollar[2].pos,
				List:           yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceCompact:   forceCompact(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
				ForceMultiLine: forceMultiLine(yyDollar[2].pos, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 39:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:468
		{
			yyVAL.expr = &IndexExpr{
				X:          yyDollar[1].expr,
				IndexStart: yyDollar[2].pos,
				Y:          yyDollar[3].expr,
				End:        yyDollar[4].pos,
			}
		}
	case 40:
		yyDollar = yyS[yypt-6 : yypt+1]
		//line build/parse.y:477
		{
			yyVAL.expr = &SliceExpr{
				X:          yyDollar[1].expr,
				SliceStart: yyDollar[2].pos,
				From:       yyDollar[3].expr,
				FirstColon: yyDollar[4].pos,
				To:         yyDollar[5].expr,
				End:        yyDollar[6].pos,
			}
		}
	case 41:
		yyDollar = yyS[yypt-8 : yypt+1]
		//line build/parse.y:488
		{
			yyVAL.expr = &SliceExpr{
				X:           yyDollar[1].expr,
				SliceStart:  yyDollar[2].pos,
				From:        yyDollar[3].expr,
				FirstColon:  yyDollar[4].pos,
				To:          yyDollar[5].expr,
				SecondColon: yyDollar[6].pos,
				Step:        yyDollar[7].expr,
				End:         yyDollar[8].pos,
			}
		}
	case 42:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:501
		{
			if len(yyDollar[1].strings) == 1 {
				yyVAL.expr = yyDollar[1].strings[0]
				break
			}
			yyVAL.expr = yyDollar[1].strings[0]
			for _, x := range yyDollar[1].strings[1:] {
				_, end := yyVAL.expr.Span()
				yyVAL.expr = binary(yyVAL.expr, end, "+", x)
			}
		}
	case 43:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:513
		{
			yyVAL.expr = &ListExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 44:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:522
		{
			yyVAL.expr = &Comprehension{
				Curly:          false,
				Lbrack:         yyDollar[1].pos,
				Body:           yyDollar[2].expr,
				Clauses:        yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLineComprehension(yyDollar[1].pos, yyDollar[2].expr, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 45:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:533
		{
			yyVAL.expr = &Comprehension{
				Curly:          true,
				Lbrack:         yyDollar[1].pos,
				Body:           yyDollar[2].expr,
				Clauses:        yyDollar[3].exprs,
				End:            End{Pos: yyDollar[4].pos},
				ForceMultiLine: forceMultiLineComprehension(yyDollar[1].pos, yyDollar[2].expr, yyDollar[3].exprs, yyDollar[4].pos),
			}
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:544
		{
			yyVAL.expr = &DictExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 47:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:553
		{
			yyVAL.expr = &SetExpr{
				Start:          yyDollar[1].pos,
				List:           yyDollar[2].exprs,
				End:            End{Pos: yyDollar[3].pos},
				ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
			}
		}
	case 48:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:562
		{
			if len(yyDollar[2].exprs) == 1 && yyDollar[2].comma.Line == 0 {
				// Just a parenthesized expression, not a tuple.
				yyVAL.expr = &ParenExpr{
					Start:          yyDollar[1].pos,
					X:              yyDollar[2].exprs[0],
					End:            End{Pos: yyDollar[3].pos},
					ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
				}
			} else {
				yyVAL.expr = &TupleExpr{
					Start:          yyDollar[1].pos,
					List:           yyDollar[2].exprs,
					End:            End{Pos: yyDollar[3].pos},
					ForceCompact:   forceCompact(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
					ForceMultiLine: forceMultiLine(yyDollar[1].pos, yyDollar[2].exprs, yyDollar[3].pos),
				}
			}
		}
	case 49:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:583
		{
			yyVAL.exprs = nil
		}
	case 50:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:587
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 51:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:593
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 52:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:597
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 54:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:604
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 55:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:608
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 56:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:612
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 57:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:617
		{
			yyVAL.loadargs = []*struct {
				from Ident
				to   Ident
			}{yyDollar[1].loadarg}
		}
	case 58:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:621
		{
			yyDollar[1].loadargs = append(yyDollar[1].loadargs, yyDollar[3].loadarg)
			yyVAL.loadargs = yyDollar[1].loadargs
		}
	case 59:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:627
		{
			yyVAL.loadarg = &struct {
				from Ident
				to   Ident
			}{
				from: Ident{
					Name:    yyDollar[1].string.Value,
					NamePos: yyDollar[1].string.Start,
				},
				to: Ident{
					Name:    yyDollar[1].string.Value,
					NamePos: yyDollar[1].string.Start,
				},
			}
		}
	case 60:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:640
		{
			yyVAL.loadarg = &struct {
				from Ident
				to   Ident
			}{
				from: Ident{
					Name:    yyDollar[3].string.Value,
					NamePos: yyDollar[3].string.Start,
				},
				to: *yyDollar[1].expr.(*Ident),
			}
		}
	case 61:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:651
		{
			yyVAL.exprs = nil
		}
	case 62:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:655
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 63:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:661
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 64:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:665
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 66:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:672
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 67:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:676
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 68:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:680
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 70:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:687
		{
			tuple, ok := yyDollar[1].expr.(*TupleExpr)
			if !ok || !tuple.NoBrackets {
				tuple = &TupleExpr{
					List:           []Expr{yyDollar[1].expr},
					NoBrackets:     true,
					ForceCompact:   true,
					ForceMultiLine: false,
				}
			}
			tuple.List = append(tuple.List, yyDollar[3].expr)
			yyVAL.expr = tuple
		}
	case 71:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:702
		{
			yyVAL.expr = nil
		}
	case 73:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:709
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 74:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:713
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 75:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:718
		{
			yyVAL.exprs = nil
		}
	case 76:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:722
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 78:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:729
		{
			yyVAL.expr = &LambdaExpr{
				Function: Function{
					StartPos: yyDollar[1].pos,
					Params:   yyDollar[2].exprs,
					Body:     []Expr{yyDollar[4].expr},
				},
			}
		}
	case 79:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:738
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 80:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:739
		{
			yyVAL.expr = unary(yyDollar[1].pos, yyDollar[1].tok, yyDollar[2].expr)
		}
	case 81:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:740
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 82:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:741
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 83:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:742
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 84:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:743
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 85:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:744
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 86:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:745
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 87:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:746
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 88:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:747
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 89:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:748
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 90:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:749
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 91:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:750
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 92:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:751
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 93:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:752
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "not in", yyDollar[4].expr)
		}
	case 94:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:753
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 95:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:754
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 96:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:755
		{
			yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
		}
	case 97:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:757
		{
			if b, ok := yyDollar[3].expr.(*UnaryExpr); ok && b.Op == "not" {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, "is not", b.X)
			} else {
				yyVAL.expr = binary(yyDollar[1].expr, yyDollar[2].pos, yyDollar[2].tok, yyDollar[3].expr)
			}
		}
	case 98:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line build/parse.y:765
		{
			yyVAL.expr = &ConditionalExpr{
				Then:      yyDollar[1].expr,
				IfStart:   yyDollar[2].pos,
				Test:      yyDollar[3].expr,
				ElseStart: yyDollar[4].pos,
				Else:      yyDollar[5].expr,
			}
		}
	case 99:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:777
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 100:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:781
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 101:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:786
		{
			yyVAL.expr = nil
		}
	case 103:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:792
		{
			yyVAL.exprs, yyVAL.comma = nil, Position{}
		}
	case 104:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:796
		{
			yyVAL.exprs, yyVAL.comma = yyDollar[1].exprs, yyDollar[2].pos
		}
	case 105:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line build/parse.y:805
		{
			yyVAL.pos = Position{}
		}
	case 107:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:811
		{
			yyVAL.expr = &KeyValueExpr{
				Key:   yyDollar[1].expr,
				Colon: yyDollar[2].pos,
				Value: yyDollar[3].expr,
			}
		}
	case 108:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:821
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 109:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:825
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[3].expr)
		}
	case 110:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:831
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 111:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:835
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 113:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:842
		{
			tuple, ok := yyDollar[1].expr.(*TupleExpr)
			if !ok || !tuple.NoBrackets {
				tuple = &TupleExpr{
					List:           []Expr{yyDollar[1].expr},
					NoBrackets:     true,
					ForceCompact:   true,
					ForceMultiLine: false,
				}
			}
			tuple.List = append(tuple.List, yyDollar[3].expr)
			yyVAL.expr = tuple
		}
	case 114:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:858
		{
			yyVAL.string = &StringExpr{
				Start:       yyDollar[1].pos,
				Value:       yyDollar[1].str,
				TripleQuote: yyDollar[1].triple,
				End:         yyDollar[1].pos.add(yyDollar[1].tok),
				Token:       yyDollar[1].tok,
			}
		}
	case 115:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:870
		{
			yyVAL.strings = []*StringExpr{yyDollar[1].string}
		}
	case 116:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:874
		{
			yyVAL.strings = append(yyDollar[1].strings, yyDollar[2].string)
		}
	case 117:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:880
		{
			yyVAL.expr = &Ident{NamePos: yyDollar[1].pos, Name: yyDollar[1].tok}
		}
	case 118:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:886
		{
			yyVAL.expr = &LiteralExpr{Start: yyDollar[1].pos, Token: yyDollar[1].tok}
		}
	case 119:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line build/parse.y:892
		{
			yyVAL.expr = &ForClause{
				For:  yyDollar[1].pos,
				Vars: yyDollar[2].expr,
				In:   yyDollar[3].pos,
				X:    yyDollar[4].expr,
			}
		}
	case 120:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:902
		{
			yyVAL.exprs = []Expr{yyDollar[1].expr}
		}
	case 121:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line build/parse.y:905
		{
			yyVAL.exprs = append(yyDollar[1].exprs, &IfClause{
				If:   yyDollar[2].pos,
				Cond: yyDollar[3].expr,
			})
		}
	case 122:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line build/parse.y:914
		{
			yyVAL.exprs = yyDollar[1].exprs
		}
	case 123:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line build/parse.y:917
		{
			yyVAL.exprs = append(yyDollar[1].exprs, yyDollar[2].exprs...)
		}
	}
	goto yystack /* stack new state and value */
}
