package frontend

import (
	"fmt"
	"strconv"
)

type Parser struct {
	tokens []Token
}

func (p *Parser) notEOF() bool {
	return p.tokens[0].Type != EOF
}
func (p *Parser) at() Token {
	return p.tokens[0]
}
func (p *Parser) peek() Token {
	if len(p.tokens) > 1 && p.tokens[1].Type != EOF {
		return p.tokens[1]

	}
	panic("Parsing Error: Unexpected EOF")
}
func (p *Parser) consume() Token {
	prev := p.tokens[0]
	p.tokens = p.tokens[1:]
	return prev
}

func (p *Parser) expect(tokenType TokenType, message string) Token {
	prev := p.tokens[0]
	p.tokens = p.tokens[1:]
	if prev.Type != tokenType {
		//fmt.Printf("Parsing Error: \n%s %v- Expecting %v\n", message, prev, tokenType)
		panic(fmt.Sprintf("Parsing Error: %s but got %v on line: %d column: %d ", message, prev, prev.LineNumber, prev.Column))
	}
	return prev
}
func (p *Parser) GenerateAST(tokens []Token) Program {
	p.tokens = tokens
	program := Program{
		Kind: ProgramNodeType,
		Body: []Statement{},
	}
	for p.notEOF() {
		program.Body = append(program.Body, p.parseStatement())
	}
	return program
}

func (p *Parser) Parse(tokens []Token) (program Program, diagnostics []Diagnostic) {
	program = Program{Kind: ProgramNodeType, Body: []Statement{}}
	diagnostics = []Diagnostic{}

	defer func() {
		if r := recover(); r != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprint(r),
				Phase:   "parse",
			})
		}
	}()

	program = p.GenerateAST(tokens)
	return program, diagnostics
}
func (p *Parser) parseStatement() Statement {
	switch p.at().Type {

	case Var, Const:
		return p.parseVariableDeclaration()
	case Fn, Async:
		fnDecl := p.parseFunctionDeclaration()
		if p.at().Type == SemiColon {
			p.consume()
		}
		return fnDecl
	case Return:
		return p.parseReturnStatement()
	case Try:
		return p.parseTryCatchStatement()
	case If:
		return p.parseIfStatement()
	case While:
		return p.parseWhileStatement()
	case For:
		return p.parseForStatement()
	case Switch:
		return p.parseSwitchStatement()
	case Export:
		return p.parseExportStatement()
	case Break:
		tok := p.consume()
		return BreakStatement{
			Kind:   BreakStatementNodeType,
			Line:   tok.LineNumber,
			Column: tok.Column,
		}
	case Continue:
		tok := p.consume()
		return ContinueStatement{
			Kind:   ContinueStatementNodeType,
			Line:   tok.LineNumber,
			Column: tok.Column,
		}
	case Import:
		return p.parseImportStatement()
	default:
		expr := p.parseExpression(false)
		if p.at().Type == SemiColon {
			p.consume()
		}
		return expr

	}
}
func (p *Parser) isControlFlowKeyword(t TokenType) bool {
	switch t {
	case If, While, Fn, For, Try, Catch, Finally, Else:
		return true
	default:
		return false
	}
}
func (p *Parser) parseImportStatement() Statement {
	importTok := p.consume()
	if p.at().Type == StringLiteralToken {
		source := p.expect(StringLiteralToken, "expected string literal").Value
		if p.at().Type == SemiColon {
			p.consume()
		}
		return ImportStatement{Source: source, Kind: ImportStatementNodeType, Line: importTok.LineNumber, Column: importTok.Column}
	}
	var specifiers []ImportSpecifier
	for {
		ident := p.expect(IdentifierToken, "Expected identfier").Value
		imported := ident
		local := imported
		if p.at().Type == As {
			p.consume()
			alias := p.expect(IdentifierToken, "Expected identfier").Value
			local = alias
		}
		specifiers = append(specifiers, ImportSpecifier{
			Imported: imported,
			Local:    local,
		})
		if p.at().Type == Comma {
			p.consume()
			continue
		}
		break
	}
	p.expect(From, "Expected 'from' after import list")
	source := p.expect(StringLiteralToken, "Expected string literal").Value
	if p.at().Type == SemiColon {
		p.consume()
	}

	return ImportStatement{
		Specifiers: specifiers,
		Source:     source,
		Kind:       ImportStatementNodeType,
		Line:       importTok.LineNumber,
		Column:     importTok.Column,
	}
}
func (p *Parser) parseExportStatement() Statement {
	exportTok := p.consume()
	identifier := p.expect(IdentifierToken, "Expected identifier after export").Value
	p.expect(SemiColon, "Expected ; after export declaration")
	return ExportStatement{
		Identifier: identifier,
		Kind:       ExportStatementNodeType,
		Line:       exportTok.LineNumber,
		Column:     exportTok.Column,
	}

}
func (p *Parser) parseTryCatchStatement() Statement {
	tryKeyword := p.consume()
	tryBlock := p.parseBlock()

	var catchBlocks []CatchBlock
	if p.at().Type != Catch && p.at().Type != Finally {
		p.expect(Catch, "Expected catch or finally after try")
	}

	for p.at().Type == Catch {
		p.consume()
		errorType := p.expect(IdentifierToken, "Expected error type after 'catch'").Value
		varName := p.expect(IdentifierToken, "Expected variable name").Value
		catchBody := p.parseBlock()

		catchBlocks = append(catchBlocks, CatchBlock{
			Kind:      CatchStatementNodeType,
			ErrorType: errorType,
			VarName:   varName,
			Body:      catchBody,
		})
	}

	var finallyBlock []Statement
	if p.at().Type == Finally {
		p.consume()
		finallyBlock = p.parseBlock()
	}

	return TryCatchStatement{
		Kind:         TryCatchStatementNodeType,
		TryBlock:     tryBlock,
		CatchBlock:   catchBlocks,
		FinallyBlock: finallyBlock,
		Line:         tryKeyword.LineNumber,
		Column:       tryKeyword.Column,
	}

}

func (p *Parser) parseBlock() []Statement {
	p.expect(OpenBrace, "Expected '{'")

	var body []Statement
	for p.notEOF() && p.at().Type != CloseBrace {
		body = append(body, p.parseStatement())
	}
	p.expect(CloseBrace, "Expected '}'")

	return body
}
func (p *Parser) parseIfStatement() Statement {
	// ifKeyword := p.consume()

	// test := p.parseExpression()

	// consequent := p.parseBlock()
	// var alternate []Statement
	// if p.at().Type == Else {
	// 	p.consume()
	// 	if p.at().Type == If {
	// 		alternate = append(alternate, p.parseIfStatement())
	// 	} else {
	// 		alternate = p.parseBlock()
	// 	}
	// }
	// return IfStatement{Condition: test, Consequent: consequent, Alternate: alternate, Line: ifKeyword.LineNumber, Column: ifKeyword.Column}
	//ifKeyword := p.consume()
	//test := p.parseExpression()
	//var consequent []Statement
	//for p.notEOF() && p.at().Type != Else && !p.isControlFlowKeyword(p.at().Type) {
	//	consequent = append(consequent, p.parseStatement())
	//}
	//var alternate []Statement
	//
	//if p.at().Type == Else {
	//	p.consume() // consume 'else'
	//	if p.at().Type == If {
	//		alternate = append(alternate, p.parseIfStatement())
	//	} else {
	//		for p.notEOF() && !p.isControlFlowKeyword(p.at().Type) {
	//			alternate = append(alternate, p.parseStatement())
	//		}
	//	}
	//}
	//fmt.Println(p.at().Type)
	//p.expect(End, "If statement should be properly terminated using 'end' keyword")
	//return IfStatement{
	//	Condition:  test,
	//	Consequent: consequent,
	//	Alternate:  alternate,
	//	Line:       ifKeyword.LineNumber,
	//	Column:     ifKeyword.Column,
	//}
	ifKeyword := p.consume()
	test := p.parseExpression()
	consequent := p.parseBlock()

	var alternate []Statement
	if p.at().Type == Else {
		p.consume() // consume else
		if p.at().Type == If {
			alternate = []Statement{p.parseIfStatement()}
		} else {
			alternate = p.parseBlock()
		}
	}

	return IfStatement{
		Condition:  test,
		Consequent: consequent,
		Alternate:  alternate,
		Line:       ifKeyword.LineNumber,
		Column:     ifKeyword.Column,
	}

}

func (p *Parser) parseWhileStatement() Statement {
	whileKeyword := p.consume()
	condition := p.parseExpression()
	body := p.parseBlock()

	return WhileStatement{Condition: condition, Body: body, Line: whileKeyword.LineNumber, Column: whileKeyword.Column}
}
func (p *Parser) parseForStatement() Statement {
	forKeyword := p.consume()
	p.expect(OpenParam, "Expected ( after for")
	initial := p.parseVariableDeclaration()

	condition := p.parseExpression()
	p.expect(SemiColon, "Expected ; after for condition")
	iteration := p.parseExpression(false)
	p.expect(CloseParam, "Expected ) after for")
	body := p.parseBlock()
	return ForStatement{Initial: initial, Condition: condition, Iteration: iteration, Body: body, Line: forKeyword.LineNumber, Column: forKeyword.Column}

}

func (p *Parser) parseSwitchClauseBody() []Statement {
	if p.at().Type == FatArrow {
		p.consume()
		return []Statement{p.parseStatement()}
	}
	if p.at().Type == OpenBrace {
		return p.parseBlock()
	}
	panic(fmt.Sprintf("Parsing Error: Expected '=>' or '{' in switch clause but got %v on line: %d column: %d", p.at().Type, p.at().LineNumber, p.at().Column))
}

func (p *Parser) parseSwitchStatement() Statement {
	switchTok := p.consume()
	expr := p.parseExpression(false)
	p.expect(OpenBrace, "Expected '{' after switch expression")

	var cases []CaseClause
	var defaultBody []Statement
	hasDefault := false

	for p.notEOF() && p.at().Type != CloseBrace {
		switch p.at().Type {
		case Case:
			p.consume()
			test := p.parseExpression(false)
			consequent := p.parseSwitchClauseBody()
			cases = append(cases, CaseClause{
				Kind:       CaseClauseNodeType,
				Test:       test,
				Consequent: consequent,
			})
		case Default:
			if hasDefault {
				panic(fmt.Sprintf("Parsing Error: Duplicate default clause in switch on line: %d column: %d", p.at().LineNumber, p.at().Column))
			}
			hasDefault = true
			p.consume()
			defaultBody = p.parseSwitchClauseBody()
		default:
			panic(fmt.Sprintf("Parsing Error: Expected 'case', 'default', or '}' in switch but got %v on line: %d column: %d", p.at().Type, p.at().LineNumber, p.at().Column))
		}
	}

	p.expect(CloseBrace, "Expected '}' to close switch statement")

	return SwitchStatement{
		Kind:    SwitchStatementNodeType,
		Expr:    expr,
		Cases:   cases,
		Default: defaultBody,
		Line:    switchTok.LineNumber,
		Column:  switchTok.Column,
	}
}

func (p *Parser) parseVariableDeclaration() Statement {
	declTok := p.consume()
	isConstant := declTok.Type == Const
	identifier := p.expect(IdentifierToken, "Expected Identifier").Value
	var typeAnnotation string
	if p.at().Type == Colon {
		p.consume()
		typeToken := p.expect(IdentifierToken, "Expected type after ")
		typeAnnotation = typeToken.Value
	}
	if p.at().Type == SemiColon {
		p.consume()
		if isConstant {
			panic("Constant must be initialized")
		}
		return VariableDeclaration{Kind: VariableDeclarationNodeType, Identifier: identifier, Constant: false, TypeAnnotation: typeAnnotation, Line: declTok.LineNumber, Column: declTok.Column}
	}
	p.expect(Equals, "Expected =")
	declaration := VariableDeclaration{
		Kind:           VariableDeclarationNodeType,
		Identifier:     identifier,
		Constant:       isConstant,
		Value:          p.parseExpression(false),
		TypeAnnotation: typeAnnotation,
		Line:           declTok.LineNumber,
		Column:         declTok.Column,
	}
	p.expect(SemiColon, "Expected ;")
	return declaration
}

func (p *Parser) parseParameters() []Parameter {
	p.expect(OpenParam, "Expected (")
	var params []Parameter
	if p.at().Type != CloseParam {
		for {
			name := p.expect(IdentifierToken, "Expected parameter name").Value
			var paramType string
			if p.at().Type == Colon {
				p.consume()
				typeToken := p.expect(IdentifierToken, "Expected type annotation")
				paramType = typeToken.Value
			}
			params = append(params, Parameter{Name: name, Type: paramType})
			if p.at().Type == Comma {
				p.consume()
				continue
			}
			break
		}
	}
	p.expect(CloseParam, "Expected )")
	return params
}

func (p *Parser) parseFunctionDeclaration() Statement {
	isAsync := false
	if p.at().Type == Async {
		p.consume()
		isAsync = true
	}
	fnKeyword := p.expect(Fn, "Expected fn keyword")
	name := p.expect(IdentifierToken, "Expected function following def keyword").Value
	params := p.parseParameters()
	var returnType string
	if p.at().Type == Colon {
		p.consume()
		returnTypeToken := p.expect(IdentifierToken, "Expected return type")
		returnType = returnTypeToken.Value
	}
	if p.at().Type == FatArrow {
		p.consume()
		bodyExpr := p.parseExpression(false)
		line, col := expressionPosition(bodyExpr)
		if line == 0 {
			line, col = fnKeyword.LineNumber, fnKeyword.Column
		}
		return FunctionDeclaration{
			Name:       name,
			Parameters: params,
			Body: []Statement{ReturnStatement{
				Kind:   ReturnStatementNodeType,
				Value:  bodyExpr,
				Line:   line,
				Column: col,
			}},
			Kind:       FunctionDeclarationNodeType,
			ReturnType: returnType,
			IsAsync:    isAsync,
			Line:       fnKeyword.LineNumber,
			Column:     fnKeyword.Column,
		}
	} else if p.at().Type == OpenBrace {
		p.expect(OpenBrace, "Expected function body following declaration")
		var body []Statement
		for p.at().Type != EOF && p.at().Type != CloseBrace {
			body = append(body, p.parseStatement())
		}
		p.expect(CloseBrace, "Unmatched Brace")
		return FunctionDeclaration{
			Name:       name,
			Parameters: params,
			Body:       body,
			Kind:       FunctionDeclarationNodeType,
			ReturnType: returnType,
			IsAsync:    isAsync,
			Line:       fnKeyword.LineNumber,
			Column:     fnKeyword.Column,
		}
	} else {
		panic("Expected '=>' or '{' after function declaration")
	}

}
func (p *Parser) parseReturnStatement() Statement {
	returnTok := p.consume()
	var value Expression
	if p.at().Type != SemiColon {
		value = p.parseExpression()
	}
	p.expect(SemiColon, "Expected ;")
	return ReturnStatement{Kind: ReturnStatementNodeType, Value: value, Line: returnTok.LineNumber, Column: returnTok.Column}
}

func (p *Parser) parseExpression(requireSemicolon ...bool) Expression {
	enforceSemicolon := true
	if len(requireSemicolon) > 0 {
		enforceSemicolon = requireSemicolon[0]
	}
	return p.parseAssignmentExpression(enforceSemicolon)
}
func (p *Parser) parseAssignmentExpression(enforceSemicolon bool) Expression {
	left := p.parseLogicalOrExpression()
	switch p.at().Type {
	case Equals:
		opTok := p.consume()
		value := p.parseAssignmentExpression(enforceSemicolon)
		if enforceSemicolon {
			p.expect(SemiColon, "Expected ;")
		}
		return AssignmentExpression{Kind: AssignmentExpressionNodeType, Assignee: left, Value: value, Operator: "=", Line: opTok.LineNumber, Column: opTok.Column}
	case PlusEquals, MinusEquals, MultiplyEquals, DivideEquals:
		operatorToken := p.consume()
		value := p.parseAssignmentExpression(enforceSemicolon)
		if enforceSemicolon {
			p.expect(SemiColon, "Expected ;")
		}
		return AssignmentExpression{Kind: AssignmentExpressionNodeType, Assignee: left, Value: value, Operator: operatorToken.Value, Line: operatorToken.LineNumber, Column: operatorToken.Column}

	}

	return left
}
func (p *Parser) parseLogicalOrExpression() Expression {
	left := p.parseLogicalAndExpression()
	for p.at().Type == Or {
		opTok := p.consume()
		right := p.parseLogicalAndExpression()
		left = BinaryExpression{Kind: BinaryExpressionNodeType, Left: left, Right: right, Operator: "||", Line: opTok.LineNumber, Column: opTok.Column}

	}
	return left
}
func (p *Parser) parseLogicalAndExpression() Expression {
	left := p.parseEqualityExpression()
	for p.at().Type == And {
		opTok := p.consume()
		right := p.parseLogicalAndExpression()
		left = BinaryExpression{Kind: BinaryExpressionNodeType, Left: left, Right: right, Operator: "&&", Line: opTok.LineNumber, Column: opTok.Column}
	}
	return left
}
func (p *Parser) parseEqualityExpression() Expression {
	left := p.parseRelationalExpression()
	for p.at().Type == EqualsTo || p.at().Type == NotEqual {
		opTok := p.consume()
		right := p.parseRelationalExpression()
		left = BinaryExpression{Kind: BinaryExpressionNodeType, Left: left, Right: right, Operator: opTok.Value, Line: opTok.LineNumber, Column: opTok.Column}
	}
	return left
}

func (p *Parser) parseRelationalExpression() Expression {
	left := p.parseAdditiveExpression()
	for p.at().Type == LessThan || p.at().Type == GreaterThan || p.at().Type == LessThanOrEqual || p.at().Type == GreaterThanOrEqual || p.at().Type == NotEqual || p.at().Type == EqualsTo {
		opTok := p.consume()
		right := p.parseAdditiveExpression()
		left = BinaryExpression{Kind: BinaryExpressionNodeType, Left: left, Right: right, Operator: opTok.Value, Line: opTok.LineNumber, Column: opTok.Column}
	}
	return left
}
func (p *Parser) parseAdditiveExpression() Expression {
	left := p.parseMultiplicativeExpression()
	for p.at().Value == "+" || p.at().Value == "-" {
		opTok := p.consume()
		right := p.parseMultiplicativeExpression()
		left = BinaryExpression{Kind: BinaryExpressionNodeType, Left: left, Right: right, Operator: opTok.Value, Line: opTok.LineNumber, Column: opTok.Column}
	}
	return left
}
func (p *Parser) parseMultiplicativeExpression() Expression {
	left := p.parseCallMemberExpression()
	for p.at().Value == "/" || p.at().Value == "*" || p.at().Value == "%" {
		opTok := p.consume()
		right := p.parseCallMemberExpression()
		left = BinaryExpression{Kind: BinaryExpressionNodeType, Left: left, Right: right, Operator: opTok.Value, Line: opTok.LineNumber, Column: opTok.Column}
	}
	return left
}

func (p *Parser) parseCallMemberExpression() Expression {
	member := p.parseMemberExpression()
	if p.at().Type == OpenParam {
		return p.parseCallExpression(member)
	}
	return member
}
func (p *Parser) parseCallExpression(caller Expression) Expression {
	line, col := expressionPosition(caller)
	callExpression := CallExpression{Kind: CallExpressionNodeType, Callee: caller, Arguments: p.parseArgs(), Line: line, Column: col}
	if p.at().Type == OpenParam {
		callExpression = p.parseCallExpression(callExpression).(CallExpression)
	}

	return callExpression
}
func (p *Parser) parseArgs() []Expression {
	p.expect(OpenParam, "Expected (")
	var args []Expression
	if p.at().Type != CloseParam {
		args = p.parseArgumentList()
	}
	p.expect(CloseParam, "Expected )")
	return args
}

func (p *Parser) parseArgumentList() []Expression {
	args := []Expression{p.parseAssignmentExpression(false)}

	for p.at().Type == Comma {
		p.consume()
		args = append(args, p.parseAssignmentExpression(false))
	}
	return args
}
func (p *Parser) parseMemberExpression() Expression {
	object := p.parseUnaryExpression()
	for p.at().Type == Dot || p.at().Type == OpenBracket {
		operator := p.consume()
		var property Expression
		var computed bool
		if operator.Type == Dot {
			computed = false
			property = p.parsePrimaryExpression()
			if property.GetKind() != "Identifier" {
				panic("Expected Identifier")
			}
		} else {
			computed = true
			property = p.parseExpression()
			p.expect(CloseBracket, "Expected ]")
		}
		object = MemberExpression{Kind: MemberExpressionNodeType, Object: object, Property: property, Computed: computed, Line: operator.LineNumber, Column: operator.Column}
	}
	return object
}

func (p *Parser) parseUnaryExpression() Expression {
	token := p.at()
	if token.Type == Increment || token.Type == Decrement || token.Type == Not {
		operator := p.consume().Value
		operand := p.parsePrimaryExpression()
		return UnaryExpression{Operator: operator, Operand: operand, Kind: UnaryExpressionNodeType, Line: token.LineNumber, Column: token.Column}
	}
	return p.parsePrimaryExpression()
}

func (p *Parser) parseArrayLiteral() ArrayLiteral {
	open := p.expect(OpenBracket, "Expected [")
	var elements []Expression
	for p.notEOF() && p.at().Type != CloseBracket {
		elements = append(elements, p.parseExpression(false))
		if p.at().Type == Comma {
			p.consume()
		}
	}
	p.expect(CloseBracket, "Unmatched ]")
	return ArrayLiteral{Kind: ArrayLiteralNodeType, Elements: elements, Line: open.LineNumber, Column: open.Column}
}

func (p *Parser) parsePrimaryExpression() Expression {
	cursor := p.at().Type
	switch cursor {
	case IdentifierToken:
		token := p.consume()
		return Identifier{Symbol: token.Value, Kind: IdentifierNodeType, Line: token.LineNumber, Column: token.Column}
	case Number:
		token := p.consume()
		value, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			fmt.Printf("Error parsing integer: %v\n", err)
			panic("Parsing Error")
		}
		return NumericLiteral{Value: value, Kind: NumericLiteralNodeType, Line: token.LineNumber, Column: token.Column}
	case True:
		token := p.consume()
		return BooleanLiteral{Value: true, Kind: BooleanLiteralNodeType, Line: token.LineNumber, Column: token.Column}
	case False:
		token := p.consume()
		return BooleanLiteral{Value: false, Kind: BooleanLiteralNodeType, Line: token.LineNumber, Column: token.Column}
	case StringLiteralToken:
		token := p.consume()
		return StringLiteral{Kind: StringLiteralNodeType, Value: token.Value, Line: token.LineNumber, Column: token.Column}
	case OpenParam:
		p.consume()
		value := p.parseExpression()
		p.expect(CloseParam, "Unmatched Parenthesis")
		return value
	case OpenBracket:
		return p.parseArrayLiteral()
	case OpenBrace:
		open := p.consume()
		var properties []Property
		for p.notEOF() && p.at().Type != CloseBrace {
			key := p.expect(IdentifierToken, "Expected Identifier")
			p.expect(Colon, "Expected :")
			value := p.parseExpression(false)
			properties = append(properties, Property{Kind: PropertyNodeType, Key: key.Value, Value: value, Line: key.LineNumber, Column: key.Column})
			if p.at().Type == Comma {
				p.consume()
			}
		}
		p.expect(CloseBrace, "Unmatched Brace")
		return MapLiteral{Properties: properties, Kind: MapLiteralNodeType, Line: open.LineNumber, Column: open.Column}
	case Fn:
		fnKeyword := p.consume()
		params := p.parseParameters()
		var returnType string
		if p.at().Type == Colon {
			p.consume()
			returnTypeToken := p.expect(IdentifierToken, "Expected return type")
			returnType = returnTypeToken.Value
		}
		if p.at().Type == FatArrow {
			p.consume()
			bodyExpr := p.parseExpression(false)
			line, col := expressionPosition(bodyExpr)
			if line == 0 {
				line, col = fnKeyword.LineNumber, fnKeyword.Column
			}
			return FunctionExpression{
				Kind:       FunctionExpressionNodeType,
				Parameters: params,
				Body: []Statement{ReturnStatement{
					Kind:   ReturnStatementNodeType,
					Value:  bodyExpr,
					Line:   line,
					Column: col,
				}},
				ReturnType: returnType,
				Line:       fnKeyword.LineNumber,
				Column:     fnKeyword.Column,
			}
		} else {
			var body = p.parseBlock()
			return FunctionExpression{
				Kind:       FunctionExpressionNodeType,
				Parameters: params,
				Body:       body,
				ReturnType: returnType,
				Line:       fnKeyword.LineNumber,
				Column:     fnKeyword.Column,
			}

		}
	case If:
		return p.parseIfStatement()
	case Await:
		awaitKeyword := p.consume()
		argument := p.parseExpression()
		return AwaitExpression{Kind: AwaitExpressionNodeType, Argument: argument, Line: awaitKeyword.LineNumber, Column: awaitKeyword.Column}
	default:
		panic(fmt.Sprintf("Parsing Error: Unexpcted token got %v on line: %d column: %d ", p.at().Type, p.at().LineNumber, p.at().Column))

	}
}

func expressionPosition(expr Expression) (int, int) {
	switch e := expr.(type) {
	case Identifier:
		return e.Line, e.Column
	case NumericLiteral:
		return e.Line, e.Column
	case FloatLiteral:
		return e.Line, e.Column
	case BooleanLiteral:
		return e.Line, e.Column
	case StringLiteral:
		return e.Line, e.Column
	case UnaryExpression:
		return e.Line, e.Column
	case BinaryExpression:
		return e.Line, e.Column
	case AssignmentExpression:
		return e.Line, e.Column
	case MemberExpression:
		return e.Line, e.Column
	case CallExpression:
		return e.Line, e.Column
	case ArrayLiteral:
		return e.Line, e.Column
	case MapLiteral:
		return e.Line, e.Column
	case FunctionExpression:
		return e.Line, e.Column
	case AwaitExpression:
		return e.Line, e.Column
	default:
		return 0, 0
	}
}
