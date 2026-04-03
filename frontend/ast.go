package frontend

type NodeType string

const (
	ProgramNodeType              NodeType = "Program"
	VariableDeclarationNodeType  NodeType = "VariableDeclaration"
	ImportDeclarationNodeType    NodeType = "ImportDeclaration"
	FunctionDeclarationNodeType  NodeType = "FunctionDeclaration"
	NewExpressionNodeType        NodeType = "NewExpression"
	NumericLiteralNodeType       NodeType = "NumericLiteral"
	FloatLiteralNodeType         NodeType = "FloatLiteral"
	BooleanLiteralNodeType       NodeType = "BooleanLiteral"
	StringLiteralNodeType        NodeType = "StringLiteral"
	ArrayLiteralNodeType         NodeType = "ArrayLiteral"
	IdentifierNodeType           NodeType = "Identifier"
	UnaryExpressionNodeType      NodeType = "UnaryExpression"
	BinaryExpressionNodeType     NodeType = "BinaryExpression"
	AssignmentExpressionNodeType NodeType = "AssignmentExpression"
	PropertyNodeType             NodeType = "Property"
	MapLiteralNodeType           NodeType = "MapLiteral"
	MemberExpressionNodeType     NodeType = "MemberExpression"
	CallExpressionNodeType       NodeType = "CallExpression"
	BlockNodeType                NodeType = "Block"
	IfStatementNodeType          NodeType = "IfStatement"
	WhileStatementNodeType       NodeType = "WhileStatement"
	ForStatementNodeType         NodeType = "ForStatement"
	ReturnStatementNodeType      NodeType = "ReturnStatement"
	TryCatchStatementNodeType    NodeType = "TryCatchStatement"
	CatchStatementNodeType       NodeType = "CatchStatement"
	FunctionExpressionNodeType   NodeType = "FunctionExpression"
	AwaitExpressionNodeType      NodeType = "AwaitExpression"
	ImportSpecifierNodeType      NodeType = "ImportSpecifier"
	ImportStatementNodeType      NodeType = "ImportStatement"
	ElseIfStatementNodeType      NodeType = "ElseIfStatement"
	BreakStatementNodeType       NodeType = "BreakStatement"
	ContinueStatementNodeType    NodeType = "ContinueStatement"
	ExportStatementNodeType      NodeType = "ExportStatement"
)

type Statement interface {
	GetKind() NodeType
}
type Expression interface {
	Statement
}

type Program struct {
	Kind NodeType
	Body []Statement
}

func (p Program) GetKind() NodeType {
	return p.Kind
}

type VariableDeclaration struct {
	Kind           NodeType
	Identifier     string
	Constant       bool
	Value          Expression
	TypeAnnotation string
	Line           int
	Column         int
}

func (v VariableDeclaration) GetKind() NodeType {
	return VariableDeclarationNodeType
}

type Block struct {
	Kind   NodeType
	Body   []Statement
	Line   int
	Column int
}

func (b Block) GetKind() NodeType {
	return BlockNodeType
}

type ElseIfStatement struct {
	Kind       NodeType
	Condition  Expression
	Consequent []Statement
}

func (e ElseIfStatement) GetKind() NodeType {
	return ElseIfStatementNodeType
}

// IfStatement represents an if statement.
type IfStatement struct {
	Kind       NodeType
	Condition  Expression
	Consequent []Statement
	Alternate  []Statement
	ElseIf     []ElseIfStatement

	Line   int
	Column int
}

func (i IfStatement) GetKind() NodeType {
	return IfStatementNodeType
}

// WhileStatement represents a while statement.
type WhileStatement struct {
	Kind      NodeType
	Condition Expression
	Body      []Statement
	Line      int
	Column    int
}

func (w WhileStatement) GetKind() NodeType {
	//TODO implement me
	return WhileStatementNodeType
}

type ForStatement struct {
	Kind      NodeType
	Initial   Statement
	Condition Expression
	Iteration Expression
	Body      []Statement
	Line      int
	Column    int
}

func (f ForStatement) GetKind() NodeType {
	//TODO implement me
	return ForStatementNodeType
}

type BreakStatement struct {
	Kind   NodeType
	Line   int
	Column int
}

func (b BreakStatement) GetKind() NodeType {
	return BreakStatementNodeType
}

type ContinueStatement struct {
	Kind   NodeType
	Line   int
	Column int
}

func (c ContinueStatement) GetKind() NodeType {
	return ContinueStatementNodeType
}

// FunctionDeclaration represents a function declaration.
type FunctionDeclaration struct {
	Kind       NodeType
	Name       string
	Parameters []Parameter
	Body       []Statement
	ReturnType string
	IsAsync    bool
	Line       int
	Column     int
}
type Parameter struct {
	Name string
	Type string
}

func (f FunctionDeclaration) GetKind() NodeType {
	//TODO implement me
	return FunctionDeclarationNodeType
}

type ReturnStatement struct {
	Kind   NodeType
	Value  Expression
	Line   int
	Column int
}

func (r ReturnStatement) GetKind() NodeType {
	return ReturnStatementNodeType
}

// AssignmentExpression represents an assignment expression.
type AssignmentExpression struct {
	Kind     NodeType
	Assignee Expression
	Value    Expression
	Operator string
	Line     int
	Column   int
}

func (a AssignmentExpression) GetKind() NodeType {
	//TODO implement me
	return AssignmentExpressionNodeType
}

// UnaryExpression represents a unary expression.
type UnaryExpression struct {
	Kind     NodeType
	Operator string
	Operand  Expression
	Line     int
	Column   int
}

func (u UnaryExpression) GetKind() NodeType {
	//TODO implement me
	return UnaryExpressionNodeType
}

type BinaryExpression struct {
	Kind     NodeType
	Operator string
	Left     Expression
	Right    Expression
	Line     int
	Column   int
}

func (b BinaryExpression) GetKind() NodeType {
	//TODO implement me
	return BinaryExpressionNodeType
}

type Identifier struct {
	Kind   NodeType
	Symbol string
	Line   int
	Column int
}

func (i Identifier) GetKind() NodeType {
	//TODO implement me
	return IdentifierNodeType
}

type NumericLiteral struct {
	Kind   NodeType
	Value  float64
	Line   int
	Column int
}

func (n NumericLiteral) GetKind() NodeType {
	//TODO implement me
	return NumericLiteralNodeType
}

type BooleanLiteral struct {
	Kind   NodeType
	Value  bool
	Line   int
	Column int
}

func (b BooleanLiteral) GetKind() NodeType {
	return BooleanLiteralNodeType
}

type FloatLiteral struct {
	Kind   NodeType
	Value  float64
	Line   int
	Column int
}

func (f FloatLiteral) GetKind() NodeType {

	return FloatLiteralNodeType
}

type StringLiteral struct {
	Kind   NodeType
	Value  string
	Linet  int
	Column int
}

func (s StringLiteral) GetKind() NodeType {
	//TODO implement me
	return StringLiteralNodeType
}

type ArrayLiteral struct {
	Kind     NodeType
	Elements []Expression
	Line     int
	Column   int
}

func (a ArrayLiteral) GetKind() NodeType {
	//TODO implement me
	return ArrayLiteralNodeType
}

type MapLiteral struct {
	Kind       NodeType
	Properties []Property
	Line       int
	Column     int
}

func (m MapLiteral) GetKind() NodeType {
	//TODO implement me
	return MapLiteralNodeType
}

type Property struct {
	Kind   NodeType
	Key    string
	Value  Expression
	Line   int
	Column int
}
type CallExpression struct {
	Kind      NodeType
	Callee    Expression
	Arguments []Expression
	Line      int
	Column    int
}

func (c CallExpression) GetKind() NodeType {
	//TODO implement me
	return CallExpressionNodeType
}

type MemberExpression struct {
	Kind     NodeType
	Object   Expression
	Property Expression
	Computed bool
	Line     int
	Column   int
}

func (m MemberExpression) GetKind() NodeType {
	//TODO implement me
	return MemberExpressionNodeType
}

type ModuleDeclaration struct {
	Kind   NodeType
	Name   string
	Body   []Statement
	Line   int
	Column int
}
type ImportDeclaration struct {
	Kind   NodeType
	Name   string
	Line   int
	Column int
}

type TryCatchStatement struct {
	Kind         NodeType
	TryBlock     []Statement
	CatchBlock   []CatchBlock
	FinallyBlock []Statement
	Line         int
	Column       int
}

func (t TryCatchStatement) GetKind() NodeType {
	return TryCatchStatementNodeType
}

type CatchBlock struct {
	Kind      NodeType
	ErrorType string
	VarName   string
	Body      []Statement
}

func (c CatchBlock) GetKind() NodeType {
	return CatchStatementNodeType
}

func (i ImportDeclaration) GetKind() NodeType {
	return ImportDeclarationNodeType
}

type FunctionExpression struct {
	Kind       NodeType
	Parameters []Parameter
	Body       []Statement
	ReturnType string
	IsAsync    bool
	Line       int
	Column     int
}

func (f FunctionExpression) GetKind() NodeType {
	return FunctionExpressionNodeType
}

type AwaitExpression struct {
	Kind     NodeType
	Argument Expression
	Line     int
	Column   int
}

func (a AwaitExpression) GetKind() NodeType {
	return AwaitExpressionNodeType
}

type ImportSpecifier struct {
	Imported string
	Local    string
}

func (i ImportSpecifier) GetKind() NodeType {
	return ImportSpecifierNodeType
}

type ImportStatement struct {
	Kind       NodeType
	Specifiers []ImportSpecifier
	Source     string
	Line       int
	Column     int
}

func (i ImportStatement) GetKind() NodeType {
	return ImportStatementNodeType
}

type ExportStatement struct {
	Kind       NodeType
	Identifier string
	Value      Expression
	Line       int
	Column     int
}

func (e ExportStatement) GetKind() NodeType {
	return ExportStatementNodeType
}
