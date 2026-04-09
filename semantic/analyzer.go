package semantic

import (
	"Kairo/frontend"
	"Kairo/stdlib"
	"fmt"
	"strings"
)

const (
	typeUnknown  = "unknown"
	typeAny      = "any"
	typeNull     = "null"
	typeBool     = "bool"
	typeInt      = "int"
	typeNumber   = "number"
	typeString   = "string"
	typeArray    = "array"
	typeMap      = "map"
	typeFunction = "function"
	typeError    = "error"
)

type Symbol struct {
	Name       string
	Type       string
	Constant   bool
	IsFunction bool
	Arity      int
	ParamTypes []string
	ReturnType string
}

type Scope struct {
	symbols map[string]Symbol
}

type functionContext struct {
	name       string
	returnType string
	isAsync    bool
}

type Analyzer struct {
	scopes      []Scope
	diagnostics []frontend.Diagnostic
	loopDepth   int
	currentFn   *functionContext
}

func NewAnalyzer() *Analyzer {
	a := &Analyzer{}
	a.pushScope()
	a.declareBuiltin("print", -1, nil, typeNull)
	a.declareBuiltin("len", 1, []string{typeAny}, typeNumber)
	return a
}

func Analyze(program frontend.Program) []frontend.Diagnostic {
	analyzer := NewAnalyzer()
	return analyzer.AnalyzeProgram(program)
}

func (a *Analyzer) AnalyzeProgram(program frontend.Program) []frontend.Diagnostic {
	a.predeclareFunctions(program.Body)
	for _, stmt := range program.Body {
		a.analyzeStatement(stmt)
	}
	return a.diagnostics
}

func (a *Analyzer) pushScope() {
	a.scopes = append(a.scopes, Scope{symbols: make(map[string]Symbol)})
}

func (a *Analyzer) popScope() {
	if len(a.scopes) == 0 {
		return
	}
	a.scopes = a.scopes[:len(a.scopes)-1]
}

func (a *Analyzer) currentScope() *Scope {
	if len(a.scopes) == 0 {
		return nil
	}
	return &a.scopes[len(a.scopes)-1]
}

func (a *Analyzer) declareBuiltin(name string, arity int, paramTypes []string, returnType string) {
	scope := a.currentScope()
	scope.symbols[name] = Symbol{
		Name:       name,
		Type:       typeFunction,
		Constant:   true,
		IsFunction: true,
		Arity:      arity,
		ParamTypes: normalizeTypeList(paramTypes),
		ReturnType: normalizeTypeName(returnType),
	}
}

func (a *Analyzer) declare(sym Symbol, line, col int) {
	scope := a.currentScope()
	if scope == nil {
		return
	}
	if _, exists := scope.symbols[sym.Name]; exists {
		a.addDiagnostic(line, col, fmt.Sprintf("redeclaration of '%s'", sym.Name))
		return
	}
	sym.Type = normalizeTypeName(sym.Type)
	sym.ReturnType = normalizeTypeName(sym.ReturnType)
	sym.ParamTypes = normalizeTypeList(sym.ParamTypes)
	if sym.Type == "" {
		sym.Type = typeUnknown
	}
	scope.symbols[sym.Name] = sym
}

func (a *Analyzer) assignType(name, newType string) {
	newType = normalizeTypeName(newType)
	if newType == "" || newType == typeUnknown {
		return
	}
	for i := len(a.scopes) - 1; i >= 0; i-- {
		sym, ok := a.scopes[i].symbols[name]
		if !ok {
			continue
		}
		if sym.Type == "" || sym.Type == typeUnknown {
			sym.Type = newType
			a.scopes[i].symbols[name] = sym
		}
		return
	}
}

func (a *Analyzer) resolve(name string) (Symbol, bool) {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if sym, ok := a.scopes[i].symbols[name]; ok {
			return sym, true
		}
	}
	return Symbol{}, false
}

func (a *Analyzer) predeclareFunctions(statements []frontend.Statement) {
	for _, stmt := range statements {
		fn, ok := stmt.(frontend.FunctionDeclaration)
		if !ok {
			continue
		}
		paramTypes := make([]string, len(fn.Parameters))
		for i, p := range fn.Parameters {
			paramTypes[i] = p.Type
		}
		a.declare(Symbol{
			Name:       fn.Name,
			Type:       typeFunction,
			Constant:   true,
			IsFunction: true,
			Arity:      len(fn.Parameters),
			ParamTypes: paramTypes,
			ReturnType: fn.ReturnType,
		}, fn.Line, fn.Column)
	}
}

func (a *Analyzer) analyzeBlock(stmts []frontend.Statement) {
	a.pushScope()
	a.predeclareFunctions(stmts)
	for _, stmt := range stmts {
		a.analyzeStatement(stmt)
	}
	a.popScope()
}

func (a *Analyzer) analyzeFunctionBody(name string, params []frontend.Parameter, body []frontend.Statement, returnType string, isAsync bool, line, col int) {
	prevFn := a.currentFn
	a.currentFn = &functionContext{name: name, returnType: normalizeTypeName(returnType), isAsync: isAsync}

	a.pushScope()
	for _, p := range params {
		ptype := normalizeTypeName(p.Type)
		if ptype == "" {
			ptype = typeUnknown
		}
		a.declare(Symbol{Name: p.Name, Type: ptype, Constant: false}, line, col)
	}
	a.predeclareFunctions(body)
	for _, stmt := range body {
		a.analyzeStatement(stmt)
	}
	a.popScope()
	a.currentFn = prevFn
}

func (a *Analyzer) analyzeStatement(stmt frontend.Statement) {
	switch s := stmt.(type) {
	case frontend.VariableDeclaration:
		declType := normalizeTypeName(s.TypeAnnotation)
		valueType := typeUnknown
		if s.Value != nil {
			valueType = a.analyzeExpression(s.Value)
		} else if s.Constant {
			a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("const '%s' must be initialized", s.Identifier))
		}
		if declType == "" {
			declType = valueType
		}
		if declType == "" {
			declType = typeUnknown
		}
		if s.Value != nil && !isAssignable(declType, valueType) {
			a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("cannot assign %s to %s variable '%s'", typeLabel(valueType), typeLabel(declType), s.Identifier))
		}
		a.declare(Symbol{Name: s.Identifier, Type: declType, Constant: s.Constant}, s.Line, s.Column)

	case frontend.ImportStatement:
		a.analyzeImportStatement(s)

	case frontend.ExportStatement:
		if _, ok := a.resolve(s.Identifier); !ok {
			a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("cannot export undeclared identifier '%s'", s.Identifier))
		}
		if s.Value != nil {
			a.analyzeExpression(s.Value)
		}

	case frontend.IfStatement:
		a.analyzeExpression(s.Condition)
		a.analyzeBlock(s.Consequent)
		for _, elseIf := range s.ElseIf {
			a.analyzeExpression(elseIf.Condition)
			a.analyzeBlock(elseIf.Consequent)
		}
		if len(s.Alternate) > 0 {
			a.analyzeBlock(s.Alternate)
		}

	case frontend.WhileStatement:
		a.analyzeExpression(s.Condition)
		a.loopDepth++
		a.analyzeBlock(s.Body)
		a.loopDepth--

	case frontend.ForStatement:
		a.pushScope()
		if s.Initial != nil {
			a.analyzeStatement(s.Initial)
		}
		if s.Condition != nil {
			a.analyzeExpression(s.Condition)
		}
		a.loopDepth++
		a.predeclareFunctions(s.Body)
		for _, bodyStmt := range s.Body {
			a.analyzeStatement(bodyStmt)
		}
		a.loopDepth--
		if s.Iteration != nil {
			a.analyzeExpression(s.Iteration)
		}
		a.popScope()

	case frontend.SwitchStatement:
		switchType := a.analyzeExpression(s.Expr)
		for _, clause := range s.Cases {
			caseType := a.analyzeExpression(clause.Test)
			_ = switchType
			_ = caseType
			a.analyzeBlock(clause.Consequent)
		}
		if len(s.Default) > 0 {
			a.analyzeBlock(s.Default)
		}

	case frontend.FunctionDeclaration:
		a.analyzeFunctionBody(s.Name, s.Parameters, s.Body, s.ReturnType, s.IsAsync, s.Line, s.Column)

	case frontend.ReturnStatement:
		a.analyzeReturnStatement(s)

	case frontend.TryCatchStatement:
		a.analyzeBlock(s.TryBlock)
		for _, catchBlock := range s.CatchBlock {
			a.pushScope()
			if catchBlock.VarName != "" {
				a.declare(Symbol{Name: catchBlock.VarName, Type: typeError, Constant: true}, s.Line, s.Column)
			}
			a.predeclareFunctions(catchBlock.Body)
			for _, catchStmt := range catchBlock.Body {
				a.analyzeStatement(catchStmt)
			}
			a.popScope()
		}
		if len(s.FinallyBlock) > 0 {
			a.analyzeBlock(s.FinallyBlock)
		}

	case frontend.BreakStatement:
		if a.loopDepth == 0 {
			a.addDiagnostic(s.Line, s.Column, "break used outside of loop")
		}

	case frontend.ContinueStatement:
		if a.loopDepth == 0 {
			a.addDiagnostic(s.Line, s.Column, "continue used outside of loop")
		}

	case frontend.Expression:
		a.analyzeExpression(s)
	}
}

func (a *Analyzer) analyzeImportStatement(s frontend.ImportStatement) {
	if len(s.Specifiers) == 0 {
		if _, ok := stdlib.BuiltinModules[s.Source]; !ok {
			a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("unknown module '%s'", s.Source))
		}
		return
	}

	module, ok := stdlib.BuiltinModules[s.Source]
	if !ok {
		a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("unknown module '%s'", s.Source))
		return
	}

	for _, spec := range s.Specifiers {
		if _, exists := module.Exports[spec.Imported]; !exists {
			a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("module '%s' has no export '%s'", s.Source, spec.Imported))
			continue
		}
		a.declare(Symbol{Name: spec.Local, Type: typeUnknown, Constant: true}, s.Line, s.Column)
	}
}

func (a *Analyzer) analyzeReturnStatement(s frontend.ReturnStatement) {
	if a.currentFn == nil {
		a.addDiagnostic(s.Line, s.Column, "return used outside of function")
		if s.Value != nil {
			a.analyzeExpression(s.Value)
		}
		return
	}

	actualType := typeNull
	if s.Value != nil {
		actualType = a.analyzeExpression(s.Value)
	}
	expectedType := normalizeTypeName(a.currentFn.returnType)
	if expectedType == "" || expectedType == typeUnknown || expectedType == typeAny {
		return
	}
	if !isAssignable(expectedType, actualType) {
		a.addDiagnostic(s.Line, s.Column, fmt.Sprintf("function '%s' returns %s but declared %s", a.currentFn.name, typeLabel(actualType), typeLabel(expectedType)))
	}
}

func (a *Analyzer) analyzeExpression(expr frontend.Expression) string {
	switch e := expr.(type) {
	case frontend.NumericLiteral:
		return typeInt
	case frontend.FloatLiteral:
		return typeNumber
	case frontend.BooleanLiteral:
		return typeBool
	case frontend.StringLiteral:
		return typeString
	case frontend.ArrayLiteral:
		for _, elem := range e.Elements {
			a.analyzeExpression(elem)
		}
		return typeArray
	case frontend.MapLiteral:
		for _, prop := range e.Properties {
			a.analyzeExpression(prop.Value)
		}
		return typeMap
	case frontend.Identifier:
		sym, ok := a.resolve(e.Symbol)
		if !ok {
			// Dynamic global reads are allowed; runtime resolves missing names.
			return typeUnknown
		}
		return sym.Type
	case frontend.UnaryExpression:
		opType := a.analyzeExpression(e.Operand)
		switch e.Operator {
		case "!":
			return typeBool
		case "-":
			if opType == typeInt {
				return typeInt
			}
			return typeNumber
		default:
			line, col := expressionLocation(e)
			a.addDiagnostic(line, col, fmt.Sprintf("unsupported unary operator '%s'", e.Operator))
			return typeUnknown
		}
	case frontend.BinaryExpression:
		leftType := a.analyzeExpression(e.Left)
		rightType := a.analyzeExpression(e.Right)
		return a.checkBinaryExpression(e, leftType, rightType)
	case frontend.AssignmentExpression:
		return a.analyzeAssignmentExpression(e)
	case frontend.MemberExpression:
		objType := a.analyzeExpression(e.Object)
		if e.Computed {
			a.analyzeExpression(e.Property)
		}
		if objType == typeUnknown {
			return typeUnknown
		}
		return typeUnknown
	case frontend.CallExpression:
		return a.analyzeCallExpression(e)
	case frontend.FunctionExpression:
		a.analyzeFunctionBody("<anonymous>", e.Parameters, e.Body, e.ReturnType, e.IsAsync, e.Line, e.Column)
		return typeFunction
	case frontend.AwaitExpression:
		line, col := expressionLocation(e)
		a.addDiagnostic(line, col, "await expressions are not supported by the compiler yet")
		if e.Argument != nil {
			a.analyzeExpression(e.Argument)
		}
		return typeUnknown
	default:
		line, col := expressionLocation(expr)
		a.addDiagnostic(line, col, "unsupported expression node")
		return typeUnknown
	}
}

func (a *Analyzer) analyzeAssignmentExpression(e frontend.AssignmentExpression) string {
	valueType := a.analyzeExpression(e.Value)

	switch target := e.Assignee.(type) {
	case frontend.Identifier:
		sym, ok := a.resolve(target.Symbol)
		if !ok {
			// Language semantics allow implicit global creation on assignment.
			if len(a.scopes) > 0 {
				a.scopes[0].symbols[target.Symbol] = Symbol{Name: target.Symbol, Type: typeUnknown, Constant: false}
			}
			return valueType
		}
		if sym.Constant {
			a.addDiagnostic(target.Line, target.Column, fmt.Sprintf("cannot assign to constant '%s'", target.Symbol))
		}

		if e.Operator == "=" {
			if !isAssignable(sym.Type, valueType) {
				a.addDiagnostic(target.Line, target.Column, fmt.Sprintf("cannot assign %s to %s variable '%s'", typeLabel(valueType), typeLabel(sym.Type), target.Symbol))
			}
			a.assignType(target.Symbol, valueType)
			return valueType
		}

		if isNumeric(sym.Type) && isNumeric(valueType) {
			if sym.Type == typeInt && valueType == typeInt && e.Operator != "/=" {
				return typeInt
			}
			return typeNumber
		}
		if e.Operator == "+=" && (sym.Type == typeString || valueType == typeString) {
			return typeString
		}
		return sym.Type

	case frontend.MemberExpression:
		a.analyzeExpression(target.Object)
		if target.Computed {
			a.analyzeExpression(target.Property)
		}
		return valueType
	default:
		line, col := expressionLocation(e.Assignee)
		a.addDiagnostic(line, col, "invalid assignment target")
		return valueType
	}
}

func (a *Analyzer) checkBinaryExpression(e frontend.BinaryExpression, leftType, rightType string) string {
	switch e.Operator {
	case "+":
		if leftType == typeString || rightType == typeString {
			return typeString
		}
		if isNumeric(leftType) && isNumeric(rightType) {
			if leftType == typeInt && rightType == typeInt {
				return typeInt
			}
			return typeNumber
		}
		return typeUnknown
	case "-", "*", "/", "%":
		if isNumeric(leftType) && isNumeric(rightType) {
			if e.Operator != "/" && leftType == typeInt && rightType == typeInt {
				return typeInt
			}
			return typeNumber
		}
		return typeUnknown
	case ">", "<", ">=", "<=":
		return typeBool
	case "==", "!=":
		return typeBool
	case "&&", "||":
		if leftType == rightType {
			return leftType
		}
		if leftType == typeUnknown || rightType == typeUnknown {
			return typeUnknown
		}
		return typeUnknown
	default:
		line, col := expressionLocation(e)
		a.addDiagnostic(line, col, fmt.Sprintf("unsupported binary operator '%s'", e.Operator))
		return typeUnknown
	}
}

func (a *Analyzer) analyzeCallExpression(e frontend.CallExpression) string {
	var calleeSym Symbol
	hasCalleeSym := false

	switch callee := e.Callee.(type) {
	case frontend.Identifier:
		sym, ok := a.resolve(callee.Symbol)
		if ok {
			calleeSym = sym
			hasCalleeSym = true
			if !sym.IsFunction && sym.Type != typeFunction && sym.Type != typeUnknown {
				a.addDiagnostic(callee.Line, callee.Column, fmt.Sprintf("'%s' is not callable", callee.Symbol))
			}
		}
	case frontend.MemberExpression:
		a.analyzeExpression(callee.Object)
		if callee.Computed {
			a.analyzeExpression(callee.Property)
		}
	default:
		a.analyzeExpression(e.Callee)
	}

	argTypes := make([]string, len(e.Arguments))
	for i, arg := range e.Arguments {
		argTypes[i] = a.analyzeExpression(arg)
	}

	if hasCalleeSym && calleeSym.IsFunction {
		if calleeSym.Arity >= 0 && calleeSym.Arity != len(argTypes) {
			line, col := expressionLocation(e)
			a.addDiagnostic(line, col, fmt.Sprintf("function '%s' expects %d arguments but got %d", calleeSym.Name, calleeSym.Arity, len(argTypes)))
		}
		for i := 0; i < len(calleeSym.ParamTypes) && i < len(argTypes); i++ {
			expected := calleeSym.ParamTypes[i]
			if expected == "" || expected == typeUnknown || expected == typeAny {
				continue
			}
			if !isAssignable(expected, argTypes[i]) {
				line, col := expressionLocation(e.Arguments[i])
				a.addDiagnostic(line, col, fmt.Sprintf("argument %d for '%s' must be %s, got %s", i+1, calleeSym.Name, typeLabel(expected), typeLabel(argTypes[i])))
			}
		}
		if calleeSym.ReturnType != "" {
			return calleeSym.ReturnType
		}
	}

	return typeUnknown
}

func (a *Analyzer) addDiagnostic(line, col int, message string) {
	a.diagnostics = append(a.diagnostics, frontend.Diagnostic{
		Message: message,
		Phase:   "semantic",
		Line:    line,
		Column:  col,
	})
}

func normalizeTypeList(list []string) []string {
	out := make([]string, len(list))
	for i, t := range list {
		out[i] = normalizeTypeName(t)
	}
	return out
}

func normalizeTypeName(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "":
		return ""
	case "int", "integer":
		return typeInt
	case "float", "double", "number":
		return typeNumber
	case "string":
		return typeString
	case "bool", "boolean":
		return typeBool
	case "array", "list":
		return typeArray
	case "map", "object", "dict", "dictionary":
		return typeMap
	case "function", "fn":
		return typeFunction
	case "error":
		return typeError
	case "null", "nil":
		return typeNull
	case "any":
		return typeAny
	default:
		return t
	}
}

func isNumeric(t string) bool {
	return t == typeInt || t == typeNumber
}

func isAssignable(targetType, valueType string) bool {
	targetType = normalizeTypeName(targetType)
	valueType = normalizeTypeName(valueType)
	if targetType == "" || targetType == typeUnknown || targetType == typeAny {
		return true
	}
	if valueType == "" || valueType == typeUnknown {
		return true
	}
	if targetType == valueType {
		return true
	}
	if targetType == typeNumber && valueType == typeInt {
		return true
	}
	return false
}

func canCompareEquality(leftType, rightType string) bool {
	leftType = normalizeTypeName(leftType)
	rightType = normalizeTypeName(rightType)
	if leftType == "" || rightType == "" || leftType == typeUnknown || rightType == typeUnknown {
		return true
	}
	if leftType == rightType {
		return true
	}
	if isNumeric(leftType) && isNumeric(rightType) {
		return true
	}
	if leftType == typeAny || rightType == typeAny {
		return true
	}
	return false
}

func isCompoundAssignmentCompatible(op, targetType, valueType string) bool {
	targetType = normalizeTypeName(targetType)
	valueType = normalizeTypeName(valueType)
	switch op {
	case "+=":
		if targetType == typeString || valueType == typeString {
			return true
		}
		return isNumeric(targetType) && isNumeric(valueType)
	case "-=", "*=", "/=":
		if targetType == typeUnknown || valueType == typeUnknown {
			return true
		}
		return isNumeric(targetType) && isNumeric(valueType)
	default:
		return true
	}
}

func typeLabel(t string) string {
	t = normalizeTypeName(t)
	if t == "" {
		return typeUnknown
	}
	return t
}

func statementLocation(stmt frontend.Statement) (int, int) {
	switch s := stmt.(type) {
	case frontend.VariableDeclaration:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		if s.Value != nil {
			return expressionLocation(s.Value)
		}
		return 0, 0
	case frontend.IfStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		return expressionLocation(s.Condition)
	case frontend.WhileStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		return expressionLocation(s.Condition)
	case frontend.ForStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		if s.Initial != nil {
			return statementLocation(s.Initial)
		}
		if s.Condition != nil {
			return expressionLocation(s.Condition)
		}
		if s.Iteration != nil {
			return expressionLocation(s.Iteration)
		}
		return 0, 0
	case frontend.FunctionDeclaration:
		return s.Line, s.Column
	case frontend.ReturnStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		if s.Value != nil {
			return expressionLocation(s.Value)
		}
		return 0, 0
	case frontend.TryCatchStatement:
		return s.Line, s.Column
	case frontend.BreakStatement:
		return s.Line, s.Column
	case frontend.ContinueStatement:
		return s.Line, s.Column
	case frontend.ImportStatement:
		return s.Line, s.Column
	case frontend.ExportStatement:
		return s.Line, s.Column
	case frontend.SwitchStatement:
		if s.Line > 0 {
			return s.Line, s.Column
		}
		return expressionLocation(s.Expr)
	case frontend.Expression:
		return expressionLocation(s)
	default:
		return 0, 0
	}
}

func expressionLocation(expr frontend.Expression) (int, int) {
	switch e := expr.(type) {
	case frontend.NumericLiteral:
		return e.Line, e.Column
	case frontend.FloatLiteral:
		return e.Line, e.Column
	case frontend.BooleanLiteral:
		return e.Line, e.Column
	case frontend.StringLiteral:
		return e.Line, e.Column
	case frontend.Identifier:
		return e.Line, e.Column
	case frontend.UnaryExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		return expressionLocation(e.Operand)
	case frontend.BinaryExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Left); line > 0 {
			return line, col
		}
		return expressionLocation(e.Right)
	case frontend.AssignmentExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Assignee); line > 0 {
			return line, col
		}
		return expressionLocation(e.Value)
	case frontend.ArrayLiteral:
		return e.Line, e.Column
	case frontend.MapLiteral:
		return e.Line, e.Column
	case frontend.MemberExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Object); line > 0 {
			return line, col
		}
		return expressionLocation(e.Property)
	case frontend.CallExpression:
		if e.Line > 0 {
			return e.Line, e.Column
		}
		if line, col := expressionLocation(e.Callee); line > 0 {
			return line, col
		}
		if len(e.Arguments) > 0 {
			return expressionLocation(e.Arguments[0])
		}
		return 0, 0
	case frontend.FunctionExpression:
		return e.Line, e.Column
	case frontend.AwaitExpression:
		return e.Line, e.Column
	default:
		return 0, 0
	}
}
