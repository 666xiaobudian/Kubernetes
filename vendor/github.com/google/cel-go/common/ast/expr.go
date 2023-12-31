// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ast

import (
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// ExprKind represents the expression node kind.
type ExprKind int

const (
	// UnspecifiedKind represents an unset expression with no specified properties.
	UnspecifiedKind ExprKind = iota

	// LiteralKind represents a primitive scalar literal.
	LiteralKind

	// IdentKind represents a simple variable, constant, or type identifier.
	IdentKind

	// SelectKind represents a field selection expression.
	SelectKind

	// CallKind represents a function call.
	CallKind

	// ListKind represents a list literal expression.
	ListKind

	// MapKind represents a map literal expression.
	MapKind

	// StructKind represents a struct literal expression.
	StructKind

	// ComprehensionKind represents a comprehension expression generated by a macro.
	ComprehensionKind
)

// NavigateCheckedAST converts a CheckedAST to a NavigableExpr
func NavigateCheckedAST(ast *CheckedAST) NavigableExpr {
	return newNavigableExpr(nil, ast.Expr, ast.TypeMap)
}

// ExprMatcher takes a NavigableExpr in and indicates whether the value is a match.
//
// This function type should be use with the `Match` and `MatchList` calls.
type ExprMatcher func(NavigableExpr) bool

// ConstantValueMatcher returns an ExprMatcher which will return true if the input NavigableExpr
// is comprised of all constant values, such as a simple literal or even list and map literal.
func ConstantValueMatcher() ExprMatcher {
	return matchIsConstantValue
}

// KindMatcher returns an ExprMatcher which will return true if the input NavigableExpr.Kind() matches
// the specified `kind`.
func KindMatcher(kind ExprKind) ExprMatcher {
	return func(e NavigableExpr) bool {
		return e.Kind() == kind
	}
}

// FunctionMatcher returns an ExprMatcher which will match NavigableExpr nodes of CallKind type whose
// function name is equal to `funcName`.
func FunctionMatcher(funcName string) ExprMatcher {
	return func(e NavigableExpr) bool {
		if e.Kind() != CallKind {
			return false
		}
		return e.AsCall().FunctionName() == funcName
	}
}

// AllMatcher returns true for all descendants of a NavigableExpr, effectively flattening them into a list.
//
// Such a result would work well with subsequent MatchList calls.
func AllMatcher() ExprMatcher {
	return func(NavigableExpr) bool {
		return true
	}
}

// MatchDescendants takes a NavigableExpr and ExprMatcher and produces a list of NavigableExpr values of the
// descendants which match.
func MatchDescendants(expr NavigableExpr, matcher ExprMatcher) []NavigableExpr {
	return matchListInternal([]NavigableExpr{expr}, matcher, true)
}

// MatchSubset applies an ExprMatcher to a list of NavigableExpr values and their descendants, producing a
// subset of NavigableExpr values which match.
func MatchSubset(exprs []NavigableExpr, matcher ExprMatcher) []NavigableExpr {
	visit := make([]NavigableExpr, len(exprs))
	copy(visit, exprs)
	return matchListInternal(visit, matcher, false)
}

func matchListInternal(visit []NavigableExpr, matcher ExprMatcher, visitDescendants bool) []NavigableExpr {
	var matched []NavigableExpr
	for len(visit) != 0 {
		e := visit[0]
		if matcher(e) {
			matched = append(matched, e)
		}
		if visitDescendants {
			visit = append(visit[1:], e.Children()...)
		} else {
			visit = visit[1:]
		}
	}
	return matched
}

func matchIsConstantValue(e NavigableExpr) bool {
	if e.Kind() == LiteralKind {
		return true
	}
	if e.Kind() == StructKind || e.Kind() == MapKind || e.Kind() == ListKind {
		for _, child := range e.Children() {
			if !matchIsConstantValue(child) {
				return false
			}
		}
		return true
	}
	return false
}

// NavigableExpr represents the base navigable expression value.
//
// Depending on the `Kind()` value, the NavigableExpr may be converted to a concrete expression types
// as indicated by the `As<Kind>` methods.
//
// NavigableExpr values and their concrete expression types should be nil-safe. Conversion of an expr
// to the wrong kind should produce a nil value.
type NavigableExpr interface {
	// ID of the expression as it appears in the AST
	ID() int64

	// Kind of the expression node. See ExprKind for the valid enum values.
	Kind() ExprKind

	// Type of the expression node.
	Type() *types.Type

	// Parent returns the parent expression node, if one exists.
	Parent() (NavigableExpr, bool)

	// Children returns a list of child expression nodes.
	Children() []NavigableExpr

	// ToExpr adapts this NavigableExpr to a protobuf representation.
	ToExpr() *exprpb.Expr

	// AsCall adapts the expr into a NavigableCallExpr
	//
	// The Kind() must be equal to a CallKind for the conversion to be well-defined.
	AsCall() NavigableCallExpr

	// AsComprehension adapts the expr into a NavigableComprehensionExpr.
	//
	// The Kind() must be equal to a ComprehensionKind for the conversion to be well-defined.
	AsComprehension() NavigableComprehensionExpr

	// AsIdent adapts the expr into an identifier string.
	//
	// The Kind() must be equal to an IdentKind for the conversion to be well-defined.
	AsIdent() string

	// AsLiteral adapts the expr into a constant ref.Val.
	//
	// The Kind() must be equal to a LiteralKind for the conversion to be well-defined.
	AsLiteral() ref.Val

	// AsList adapts the expr into a NavigableListExpr.
	//
	// The Kind() must be equal to a ListKind for the conversion to be well-defined.
	AsList() NavigableListExpr

	// AsMap adapts the expr into a NavigableMapExpr.
	//
	// The Kind() must be equal to a MapKind for the conversion to be well-defined.
	AsMap() NavigableMapExpr

	// AsSelect adapts the expr into a NavigableSelectExpr.
	//
	// The Kind() must be equal to a SelectKind for the conversion to be well-defined.
	AsSelect() NavigableSelectExpr

	// AsStruct adapts the expr into a NavigableStructExpr.
	//
	// The Kind() must be equal to a StructKind for the conversion to be well-defined.
	AsStruct() NavigableStructExpr

	// marker interface method
	isNavigable()
}

// NavigableCallExpr defines an interface for inspecting a function call and its arugments.
type NavigableCallExpr interface {
	// FunctionName returns the name of the function.
	FunctionName() string

	// Target returns the target of the expression if one is present.
	Target() NavigableExpr

	// Args returns the list of call arguments, excluding the target.
	Args() []NavigableExpr

	// ReturnType returns the result type of the call.
	ReturnType() *types.Type

	// marker interface method
	isNavigable()
}

// NavigableListExpr defines an interface for inspecting a list literal expression.
type NavigableListExpr interface {
	// Elements returns the list elements as navigable expressions.
	Elements() []NavigableExpr

	// OptionalIndicies returns the list of optional indices in the list literal.
	OptionalIndices() []int32

	// Size returns the number of elements in the list.
	Size() int

	// marker interface method
	isNavigable()
}

// NavigableSelectExpr defines an interface for inspecting a select expression.
type NavigableSelectExpr interface {
	// Operand returns the selection operand expression.
	Operand() NavigableExpr

	// FieldName returns the field name being selected from the operand.
	FieldName() string

	// IsTestOnly indicates whether the select expression is a presence test generated by a macro.
	IsTestOnly() bool

	// marker interface method
	isNavigable()
}

// NavigableMapExpr defines an interface for inspecting a map expression.
type NavigableMapExpr interface {
	// Entries returns the map key value pairs as NavigableEntry values.
	Entries() []NavigableEntry

	// Size returns the number of entries in the map.
	Size() int

	// marker interface method
	isNavigable()
}

// NavigableEntry defines an interface for inspecting a map entry.
type NavigableEntry interface {
	// Key returns the map entry key expression.
	Key() NavigableExpr

	// Value returns the map entry value expression.
	Value() NavigableExpr

	// IsOptional returns whether the entry is optional.
	IsOptional() bool

	// marker interface method
	isNavigable()
}

// NavigableStructExpr defines an interfaces for inspecting a struct and its field initializers.
type NavigableStructExpr interface {
	// TypeName returns the struct type name.
	TypeName() string

	// Fields returns the set of field initializers in the struct expression as NavigableField values.
	Fields() []NavigableField

	// marker interface method
	isNavigable()
}

// NavigableField defines an interface for inspecting a struct field initialization.
type NavigableField interface {
	// FieldName returns the name of the field.
	FieldName() string

	// Value returns the field initialization expression.
	Value() NavigableExpr

	// IsOptional returns whether the field is optional.
	IsOptional() bool

	// marker interface method
	isNavigable()
}

// NavigableComprehensionExpr defines an interface for inspecting a comprehension expression.
type NavigableComprehensionExpr interface {
	// IterRange returns the iteration range expression.
	IterRange() NavigableExpr

	// IterVar returns the iteration variable name.
	IterVar() string

	// AccuVar returns the accumulation variable name.
	AccuVar() string

	// AccuInit returns the accumulation variable initialization expression.
	AccuInit() NavigableExpr

	// LoopCondition returns the loop condition expression.
	LoopCondition() NavigableExpr

	// LoopStep returns the loop step expression.
	LoopStep() NavigableExpr

	// Result returns the comprehension result expression.
	Result() NavigableExpr

	// marker interface method
	isNavigable()
}

func newNavigableExpr(parent NavigableExpr, expr *exprpb.Expr, typeMap map[int64]*types.Type) NavigableExpr {
	kind, factory := kindOf(expr)
	nav := &navigableExprImpl{
		parent:         parent,
		kind:           kind,
		expr:           expr,
		typeMap:        typeMap,
		createChildren: factory,
	}
	return nav
}

type navigableExprImpl struct {
	parent         NavigableExpr
	kind           ExprKind
	expr           *exprpb.Expr
	typeMap        map[int64]*types.Type
	createChildren childFactory
}

func (nav *navigableExprImpl) ID() int64 {
	return nav.ToExpr().GetId()
}

func (nav *navigableExprImpl) Kind() ExprKind {
	return nav.kind
}

func (nav *navigableExprImpl) Type() *types.Type {
	if t, found := nav.typeMap[nav.ID()]; found {
		return t
	}
	return types.DynType
}

func (nav *navigableExprImpl) Parent() (NavigableExpr, bool) {
	if nav.parent != nil {
		return nav.parent, true
	}
	return nil, false
}

func (nav *navigableExprImpl) Children() []NavigableExpr {
	return nav.createChildren(nav)
}

func (nav *navigableExprImpl) ToExpr() *exprpb.Expr {
	return nav.expr
}

func (nav *navigableExprImpl) AsCall() NavigableCallExpr {
	return navigableCallImpl{navigableExprImpl: nav}
}

func (nav *navigableExprImpl) AsComprehension() NavigableComprehensionExpr {
	return navigableComprehensionImpl{navigableExprImpl: nav}
}

func (nav *navigableExprImpl) AsIdent() string {
	return nav.ToExpr().GetIdentExpr().GetName()
}

func (nav *navigableExprImpl) AsLiteral() ref.Val {
	if nav.Kind() != LiteralKind {
		return nil
	}
	val, err := ConstantToVal(nav.ToExpr().GetConstExpr())
	if err != nil {
		panic(err)
	}
	return val
}

func (nav *navigableExprImpl) AsList() NavigableListExpr {
	return navigableListImpl{navigableExprImpl: nav}
}

func (nav *navigableExprImpl) AsMap() NavigableMapExpr {
	return navigableMapImpl{navigableExprImpl: nav}
}

func (nav *navigableExprImpl) AsSelect() NavigableSelectExpr {
	return navigableSelectImpl{navigableExprImpl: nav}
}

func (nav *navigableExprImpl) AsStruct() NavigableStructExpr {
	return navigableStructImpl{navigableExprImpl: nav}
}

func (nav *navigableExprImpl) createChild(e *exprpb.Expr) NavigableExpr {
	return newNavigableExpr(nav, e, nav.typeMap)
}

func (nav *navigableExprImpl) isNavigable() {}

type navigableCallImpl struct {
	*navigableExprImpl
}

func (call navigableCallImpl) FunctionName() string {
	return call.ToExpr().GetCallExpr().GetFunction()
}

func (call navigableCallImpl) Target() NavigableExpr {
	t := call.ToExpr().GetCallExpr().GetTarget()
	if t != nil {
		return call.createChild(t)
	}
	return nil
}

func (call navigableCallImpl) Args() []NavigableExpr {
	args := call.ToExpr().GetCallExpr().GetArgs()
	navArgs := make([]NavigableExpr, len(args))
	for i, a := range args {
		navArgs[i] = call.createChild(a)
	}
	return navArgs
}

func (call navigableCallImpl) ReturnType() *types.Type {
	return call.Type()
}

type navigableComprehensionImpl struct {
	*navigableExprImpl
}

func (comp navigableComprehensionImpl) IterRange() NavigableExpr {
	return comp.createChild(comp.ToExpr().GetComprehensionExpr().GetIterRange())
}

func (comp navigableComprehensionImpl) IterVar() string {
	return comp.ToExpr().GetComprehensionExpr().GetIterVar()
}

func (comp navigableComprehensionImpl) AccuVar() string {
	return comp.ToExpr().GetComprehensionExpr().GetAccuVar()
}

func (comp navigableComprehensionImpl) AccuInit() NavigableExpr {
	return comp.createChild(comp.ToExpr().GetComprehensionExpr().GetAccuInit())
}

func (comp navigableComprehensionImpl) LoopCondition() NavigableExpr {
	return comp.createChild(comp.ToExpr().GetComprehensionExpr().GetLoopCondition())
}

func (comp navigableComprehensionImpl) LoopStep() NavigableExpr {
	return comp.createChild(comp.ToExpr().GetComprehensionExpr().GetLoopStep())
}

func (comp navigableComprehensionImpl) Result() NavigableExpr {
	return comp.createChild(comp.ToExpr().GetComprehensionExpr().GetResult())
}

type navigableListImpl struct {
	*navigableExprImpl
}

func (l navigableListImpl) Elements() []NavigableExpr {
	return l.Children()
}

func (l navigableListImpl) OptionalIndices() []int32 {
	return l.ToExpr().GetListExpr().GetOptionalIndices()
}

func (l navigableListImpl) Size() int {
	return len(l.ToExpr().GetListExpr().GetElements())
}

type navigableMapImpl struct {
	*navigableExprImpl
}

func (m navigableMapImpl) Entries() []NavigableEntry {
	mapExpr := m.ToExpr().GetStructExpr()
	entries := make([]NavigableEntry, len(mapExpr.GetEntries()))
	for i, e := range mapExpr.GetEntries() {
		entries[i] = navigableEntryImpl{
			key:   m.createChild(e.GetMapKey()),
			val:   m.createChild(e.GetValue()),
			isOpt: e.GetOptionalEntry(),
		}
	}
	return entries
}

func (m navigableMapImpl) Size() int {
	return len(m.ToExpr().GetStructExpr().GetEntries())
}

type navigableEntryImpl struct {
	key   NavigableExpr
	val   NavigableExpr
	isOpt bool
}

func (e navigableEntryImpl) Key() NavigableExpr {
	return e.key
}

func (e navigableEntryImpl) Value() NavigableExpr {
	return e.val
}

func (e navigableEntryImpl) IsOptional() bool {
	return e.isOpt
}

func (e navigableEntryImpl) isNavigable() {}

type navigableSelectImpl struct {
	*navigableExprImpl
}

func (sel navigableSelectImpl) FieldName() string {
	return sel.ToExpr().GetSelectExpr().GetField()
}

func (sel navigableSelectImpl) IsTestOnly() bool {
	return sel.ToExpr().GetSelectExpr().GetTestOnly()
}

func (sel navigableSelectImpl) Operand() NavigableExpr {
	return sel.createChild(sel.ToExpr().GetSelectExpr().GetOperand())
}

type navigableStructImpl struct {
	*navigableExprImpl
}

func (s navigableStructImpl) TypeName() string {
	return s.ToExpr().GetStructExpr().GetMessageName()
}

func (s navigableStructImpl) Fields() []NavigableField {
	fieldInits := s.ToExpr().GetStructExpr().GetEntries()
	fields := make([]NavigableField, len(fieldInits))
	for i, f := range fieldInits {
		fields[i] = navigableFieldImpl{
			name:  f.GetFieldKey(),
			val:   s.createChild(f.GetValue()),
			isOpt: f.GetOptionalEntry(),
		}
	}
	return fields
}

type navigableFieldImpl struct {
	name  string
	val   NavigableExpr
	isOpt bool
}

func (f navigableFieldImpl) FieldName() string {
	return f.name
}

func (f navigableFieldImpl) Value() NavigableExpr {
	return f.val
}

func (f navigableFieldImpl) IsOptional() bool {
	return f.isOpt
}

func (f navigableFieldImpl) isNavigable() {}

func kindOf(expr *exprpb.Expr) (ExprKind, childFactory) {
	switch expr.GetExprKind().(type) {
	case *exprpb.Expr_ConstExpr:
		return LiteralKind, noopFactory
	case *exprpb.Expr_IdentExpr:
		return IdentKind, noopFactory
	case *exprpb.Expr_SelectExpr:
		return SelectKind, selectFactory
	case *exprpb.Expr_CallExpr:
		return CallKind, callArgFactory
	case *exprpb.Expr_ListExpr:
		return ListKind, listElemFactory
	case *exprpb.Expr_StructExpr:
		if expr.GetStructExpr().GetMessageName() != "" {
			return StructKind, structEntryFactory
		}
		return MapKind, mapEntryFactory
	case *exprpb.Expr_ComprehensionExpr:
		return ComprehensionKind, comprehensionFactory
	default:
		return UnspecifiedKind, noopFactory
	}
}

type childFactory func(*navigableExprImpl) []NavigableExpr

func noopFactory(*navigableExprImpl) []NavigableExpr {
	return nil
}

func selectFactory(nav *navigableExprImpl) []NavigableExpr {
	return []NavigableExpr{
		nav.createChild(nav.ToExpr().GetSelectExpr().GetOperand()),
	}
}

func callArgFactory(nav *navigableExprImpl) []NavigableExpr {
	call := nav.ToExpr().GetCallExpr()
	argCount := len(call.GetArgs())
	if call.GetTarget() != nil {
		argCount++
	}
	navExprs := make([]NavigableExpr, argCount)
	i := 0
	if call.GetTarget() != nil {
		navExprs[i] = nav.createChild(call.GetTarget())
		i++
	}
	for _, arg := range call.GetArgs() {
		navExprs[i] = nav.createChild(arg)
		i++
	}
	return navExprs
}

func listElemFactory(nav *navigableExprImpl) []NavigableExpr {
	l := nav.ToExpr().GetListExpr()
	navExprs := make([]NavigableExpr, len(l.GetElements()))
	for i, e := range l.GetElements() {
		navExprs[i] = nav.createChild(e)
	}
	return navExprs
}

func structEntryFactory(nav *navigableExprImpl) []NavigableExpr {
	s := nav.ToExpr().GetStructExpr()
	entries := make([]NavigableExpr, len(s.GetEntries()))
	for i, e := range s.GetEntries() {

		entries[i] = nav.createChild(e.GetValue())
	}
	return entries
}

func mapEntryFactory(nav *navigableExprImpl) []NavigableExpr {
	s := nav.ToExpr().GetStructExpr()
	entries := make([]NavigableExpr, len(s.GetEntries())*2)
	j := 0
	for _, e := range s.GetEntries() {
		entries[j] = nav.createChild(e.GetMapKey())
		entries[j+1] = nav.createChild(e.GetValue())
		j += 2
	}
	return entries
}

func comprehensionFactory(nav *navigableExprImpl) []NavigableExpr {
	compre := nav.ToExpr().GetComprehensionExpr()
	return []NavigableExpr{
		nav.createChild(compre.GetIterRange()),
		nav.createChild(compre.GetAccuInit()),
		nav.createChild(compre.GetLoopCondition()),
		nav.createChild(compre.GetLoopStep()),
		nav.createChild(compre.GetResult()),
	}
}
