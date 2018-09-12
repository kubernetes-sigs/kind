/*
Copyright 2016 Google Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package build

// Walk walks the expression tree v, calling f on all subexpressions
// in a preorder traversal.
//
// The stk argument is the stack of expressions in the recursion above x,
// from outermost to innermost.
//
func Walk(v Expr, f func(x Expr, stk []Expr)) {
	var stack []Expr
	walk1(&v, &stack, func(x Expr, stk []Expr) Expr {
		f(x, stk)
		return nil
	})
}

// Edit walks the expression tree v, calling f on all subexpressions
// in a preorder traversal. If f returns a non-nil value, the tree is mutated.
// The new value replaces the old one.
//
// The stk argument is the stack of expressions in the recursion above x,
// from outermost to innermost.
//
func Edit(v Expr, f func(x Expr, stk []Expr) Expr) Expr {
	var stack []Expr
	return walk1(&v, &stack, f)
}

// walk1 is a helper function for Walk, WalkWithPostfix, and Edit.
func walk1(v *Expr, stack *[]Expr, f func(x Expr, stk []Expr) Expr) Expr {
	if v == nil {
		return nil
	}

	if res := f(*v, *stack); res != nil {
		*v = res
	}
	*stack = append(*stack, *v)

	WalkOnce(*v, func(x *Expr) {
		walk1(x, stack, f)
	})

	*stack = (*stack)[:len(*stack)-1]
	return *v
}

// WalkOnce calls f on every child of v.
func WalkOnce(v Expr, f func(x *Expr)) {
	switch v := v.(type) {
	case *File:
		for _, stmt := range v.Stmt {
			f(&stmt)
		}
	case *DotExpr:
		f(&v.X)
	case *IndexExpr:
		f(&v.X)
		f(&v.Y)
	case *KeyValueExpr:
		f(&v.Key)
		f(&v.Value)
	case *SliceExpr:
		f(&v.X)
		if v.From != nil {
			f(&v.From)
		}
		if v.To != nil {
			f(&v.To)
		}
		if v.Step != nil {
			f(&v.Step)
		}
	case *ParenExpr:
		f(&v.X)
	case *UnaryExpr:
		f(&v.X)
	case *BinaryExpr:
		f(&v.X)
		f(&v.Y)
	case *LambdaExpr:
		for i := range v.Params {
			f(&v.Params[i])
		}
		for i := range v.Body {
			f(&v.Body[i])
		}
	case *CallExpr:
		f(&v.X)
		for i := range v.List {
			f(&v.List[i])
		}
	case *ListExpr:
		for i := range v.List {
			f(&v.List[i])
		}
	case *SetExpr:
		for i := range v.List {
			f(&v.List[i])
		}
	case *TupleExpr:
		for i := range v.List {
			f(&v.List[i])
		}
	case *DictExpr:
		for i := range v.List {
			f(&v.List[i])
		}
	case *Comprehension:
		f(&v.Body)
		for _, c := range v.Clauses {
			f(&c)
		}
	case *IfClause:
		f(&v.Cond)
	case *ForClause:
		f(&v.Vars)
		f(&v.X)
	case *ConditionalExpr:
		f(&v.Then)
		f(&v.Test)
		f(&v.Else)
	case *DefStmt:
		for _, p := range v.Params {
			f(&p)
		}
		for _, s := range v.Body {
			f(&s)
		}
	case *IfStmt:
		f(&v.Cond)
		for _, s := range v.True {
			f(&s)
		}
		for _, s := range v.False {
			f(&s)
		}
	case *ForStmt:
		f(&v.Vars)
		f(&v.X)
		for _, s := range v.Body {
			f(&s)
		}
	case *ReturnStmt:
		if v.Result != nil {
			f(&v.Result)
		}
	}
}
