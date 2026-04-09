package semantic

import (
	"Kairo/frontend"
	"testing"
)

func TestAnalyzerReportsReturnOutsideFunction(t *testing.T) {
	program := frontend.Program{
		Kind: frontend.ProgramNodeType,
		Body: []frontend.Statement{
			frontend.ReturnStatement{Kind: frontend.ReturnStatementNodeType, Line: 1, Column: 1},
		},
	}

	diags := Analyze(program)
	if len(diags) == 0 {
		t.Fatalf("expected semantic diagnostic for return outside function")
	}
	if diags[0].Phase != "semantic" {
		t.Fatalf("expected semantic phase diagnostic, got %+v", diags[0])
	}
}

func TestAnalyzerBreakOutsideLoop(t *testing.T) {
	program := frontend.Program{
		Kind: frontend.ProgramNodeType,
		Body: []frontend.Statement{
			frontend.BreakStatement{Kind: frontend.BreakStatementNodeType, Line: 1, Column: 1},
		},
	}

	diags := Analyze(program)
	if len(diags) == 0 {
		t.Fatalf("expected semantic diagnostic for break outside loop")
	}
}

func TestAnalyzerValidSimpleProgram(t *testing.T) {
	program := frontend.Program{
		Kind: frontend.ProgramNodeType,
		Body: []frontend.Statement{
			frontend.VariableDeclaration{
				Kind:       frontend.VariableDeclarationNodeType,
				Identifier: "x",
				Value:      frontend.NumericLiteral{Kind: frontend.NumericLiteralNodeType, Value: 5},
				Line:       1,
				Column:     1,
			},
			frontend.Expression(frontend.AssignmentExpression{
				Kind:     frontend.AssignmentExpressionNodeType,
				Assignee: frontend.Identifier{Kind: frontend.IdentifierNodeType, Symbol: "x", Line: 2, Column: 1},
				Value:    frontend.NumericLiteral{Kind: frontend.NumericLiteralNodeType, Value: 6},
				Operator: "=",
				Line:     2,
				Column:   1,
			}),
		},
	}

	diags := Analyze(program)
	if len(diags) != 0 {
		t.Fatalf("expected no semantic diagnostics, got %+v", diags)
	}
}
