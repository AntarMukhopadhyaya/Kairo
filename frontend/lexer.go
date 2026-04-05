package frontend

import (
	"errors"
	"strings"
	"unicode"
)

type TokenType string

const (
	If                 TokenType = "IF"
	Else               TokenType = "ELSE"
	Number             TokenType = "NUMBER"
	Float              TokenType = "FLOAT"
	IdentifierToken    TokenType = "IDENTIFIER"
	Equals             TokenType = "EQUALS"
	OpenParam          TokenType = "OPEN_PARAM"
	CloseParam         TokenType = "CLOSE_PARAM"
	BinaryOperator     TokenType = "BINARY_OPERATOR"
	Var                TokenType = "VAR"
	EOF                TokenType = "EOF"
	While              TokenType = "WHILE"
	Const              TokenType = "CONST"
	SemiColon          TokenType = "SEMICOLON"
	Comma              TokenType = "COMMA"
	Colon              TokenType = "COLON"
	OpenBrace          TokenType = "OPEN_BRACE"
	CloseBrace         TokenType = "CLOSE_BRACE"
	OpenBracket        TokenType = "OPEN_BRACKET"
	CloseBracket       TokenType = "CLOSE_BRACKET"
	PlusEquals         TokenType = "PLUS_EQUALS"
	MinusEquals        TokenType = "MINUS_EQUALS"
	MultiplyEquals     TokenType = "MULTIPLY_EQUALS"
	DivideEquals       TokenType = "DIVIDE_EQUALS"
	Dot                TokenType = "DOT"
	And                TokenType = "AND"
	Or                 TokenType = "OR"
	Not                TokenType = "NOT"
	Fn                 TokenType = "Fn"
	StringLiteralToken TokenType = "STRING_LITERAL"
	LessThan           TokenType = "LESS_THAN"
	GreaterThan        TokenType = "GREATER_THAN"
	LessThanOrEqual    TokenType = "LESS_THAN_OR_EQUAL"
	GreaterThanOrEqual TokenType = "GREATER_THAN_OR_EQUAL"
	NotEqual           TokenType = "NOT_EQUALS"
	EqualsTo           TokenType = "EQUALS_TO"
	Increment          TokenType = "INCREMENT"
	Decrement          TokenType = "DECREMENT"
	Switch             TokenType = "SWITCH"
	Case               TokenType = "CASE"
	Default            TokenType = "DEFAULT"
	Return             TokenType = "RETURN"
	For                TokenType = "FOR"
	Break              TokenType = "BREAK"
	Continue           TokenType = "CONTINUE"
	Try                TokenType = "TRY"
	Catch              TokenType = "CATCH"
	Finally            TokenType = "FINALLY"
	Await              TokenType = "AWAIT"
	Async              TokenType = "ASYNC"
	FatArrow           TokenType = "FAT_ARROW"
	Import             TokenType = "IMPORT"
	As                 TokenType = "AS"
	From               TokenType = "FROM"
	End                TokenType = "END"
	Do                 TokenType = "DO"
	True               TokenType = "TRUE"
	False              TokenType = "FALSE"
	Export             TokenType = "EXPORT"
)

type Token struct {
	Type       TokenType
	Value      string
	LineNumber int
	Column     int
}
type Lexer struct {
	keyword     map[string]TokenType
	lineNumber  int
	columNumber int
}

func NewLexer() *Lexer {
	keyword := map[string]TokenType{
		"if":    If,
		"else":  Else,
		"var":   Var,
		"while": While,

		"const":    Const,
		"fn":       Fn,
		"try":      Try,
		"catch":    Catch,
		"finally":  Finally,
		"switch":   Switch,
		"case":     Case,
		"default":  Default,
		"return":   Return,
		"for":      For,
		"break":    Break,
		"continue": Continue,
		"async":    Async,
		"await":    Await,
		"import":   Import,
		"as":       As,
		"from":     From,
		"true":     True,
		"false":    False,
		"export":   Export,
	}
	return &Lexer{keyword: keyword, lineNumber: 1, columNumber: 1}
}

func (l *Lexer) Tokenize(sourceCode string) ([]Token, error) {
	// Reset position for each fresh tokenization pass.
	l.lineNumber = 1
	l.columNumber = 1

	var tokens []Token
	src := strings.Split(sourceCode, "")
	for len(src) > 0 {
		switch src[0] {
		case "\n":
			l.lineNumber++
			l.columNumber = 1
			src = src[1:]
		case "\r":
			// Ignore Windows CR in CRLF; newline is handled by '\n'
			src = src[1:]
		case "(":
			tokens = append(tokens, Token{OpenParam, "(", l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++
		case ")":
			tokens = append(tokens, Token{CloseParam, ")", l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++
		case "{":
			tokens = append(tokens, Token{OpenBrace, "{", l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case "}":
			tokens = append(tokens, Token{CloseBrace, "}", l.lineNumber, l.columNumber})

			src = src[1:]
			l.columNumber++

		case "[":
			tokens = append(tokens, Token{OpenBracket, "[", l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case "]":
			tokens = append(tokens, Token{CloseBracket, "]", l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case "=":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{EqualsTo, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else if len(src) > 1 && src[1] == ">" {
				tokens = append(tokens, Token{FatArrow, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else if len(src) > 1 && src[1] == "+" {
				tokens = append(tokens, Token{Increment, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else if len(src) > 1 && src[1] == "-" {
				tokens = append(tokens, Token{Decrement, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{Equals, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++

			}
		case "+":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{PlusEquals, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{BinaryOperator, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++
			}
		case "-":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{MinusEquals, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{BinaryOperator, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++
			}
		case "*":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{MultiplyEquals, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{BinaryOperator, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++
			}
		case "/":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{DivideEquals, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{BinaryOperator, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++
			}
		case "%":
			tokens = append(tokens, Token{BinaryOperator, src[0], l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case ".":
			tokens = append(tokens, Token{Dot, src[0], l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case ";":
			tokens = append(tokens, Token{SemiColon, src[0], l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case ",":
			tokens = append(tokens, Token{Comma, src[0], l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case ":":
			tokens = append(tokens, Token{Colon, src[0], l.lineNumber, l.columNumber})
			src = src[1:]
			l.columNumber++

		case ">":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{GreaterThanOrEqual, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{GreaterThan, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++

			}
		case "<":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{LessThanOrEqual, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{LessThan, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++

			}
		case "!":
			if len(src) > 1 && src[1] == "=" {
				tokens = append(tokens, Token{NotEqual, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				tokens = append(tokens, Token{Not, src[0], l.lineNumber, l.columNumber})
				src = src[1:]
				l.columNumber++

			}
		case "&":
			if len(src) > 1 && src[1] == "&" {
				tokens = append(tokens, Token{And, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				return nil, errors.New("Invalid character: " + src[0])
			}
		case "|":
			if len(src) > 1 && src[1] == "|" {
				tokens = append(tokens, Token{Or, src[0] + src[1], l.lineNumber, l.columNumber})
				src = src[2:]
				l.columNumber += 2
			} else {
				return nil, errors.New("Invalid character: " + src[0])
			}
		case "#":
			for len(src) > 0 && src[0] != "\n" {
				src = src[1:]
				l.columNumber++

			}

		case "\"":
			startCol := l.columNumber
			var stringLiteral strings.Builder
			src = src[1:]
			l.columNumber++ // opening quote
			for len(src) > 0 && src[0] != "\"" {
				stringLiteral.WriteString(src[0])
				src = src[1:]
				l.columNumber++
			}
			if len(src) == 0 {
				return nil, errors.New("unterminated string literal")
			}
			src = src[1:]
			l.columNumber++ // closing quote
			tokens = append(tokens, Token{StringLiteralToken, stringLiteral.String(), l.lineNumber, startCol})
		default:
			if unicode.IsDigit(rune(src[0][0])) {
				startCol := l.columNumber
				var number strings.Builder
				isFloat := false

				for len(src) > 0 {
					char := rune(src[0][0])

					if unicode.IsDigit(char) {
						number.WriteRune(char)
					} else if char == '.' {
						if isFloat { // If there's already a dot, it's an error
							return nil, errors.New("invalid floating point number: " + number.String() + ".")
						}
						isFloat = true
						number.WriteRune(char)
					} else {
						break
					}
					src = src[1:]
					l.columNumber++
				}

				tokens = append(tokens, Token{Type: Number, Value: number.String(), LineNumber: l.lineNumber, Column: startCol})

			} else if unicode.IsLetter(rune(src[0][0])) {
				startCol := l.columNumber
				var identifier strings.Builder
				for len(src) > 0 && (unicode.IsLetter(rune(src[0][0])) || unicode.IsDigit(rune(src[0][0]))) {
					identifier.WriteString(src[0])
					src = src[1:]
					l.columNumber++
				}
				tokenType, exists := l.keyword[identifier.String()]

				if !exists {

					tokenType = IdentifierToken
				}
				tokens = append(tokens, Token{tokenType, identifier.String(), l.lineNumber, startCol})
			} else if src[0] == " " || src[0] == "\t" {
				ch := src[0]
				src = src[1:]
				if ch == "\t" {
					l.columNumber += 4
				} else {
					l.columNumber++
				}
			} else {
				return nil, errors.New("Invalid character: " + src[0])
			}
		}
	}
	tokens = append(tokens, Token{EOF, "EOF", l.lineNumber, l.columNumber})
	return tokens, nil
}
