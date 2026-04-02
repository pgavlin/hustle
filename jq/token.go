package jq

// TokenKind represents the type of a lexer token.
type TokenKind int

const (
	TokenEOF       TokenKind = iota
	TokenDot                 // .
	TokenDotField            // .foo (dot + identifier)
	TokenDotDot              // ..
	TokenIdent               // bare identifier
	TokenVariable            // $foo
	TokenNumber              // 42, 3.14, 1e10
	TokenString              // "hello"
	TokenPipe                // |
	TokenComma               // ,
	TokenPlus                // +
	TokenMinus               // -
	TokenStar                // *
	TokenSlash               // /
	TokenPercent             // %
	TokenEq                  // ==
	TokenNe                  // !=
	TokenLt                  // <
	TokenGt                  // >
	TokenLe                  // <=
	TokenGe                  // >=
	TokenAlt                 // //
	TokenUpdateAlt           // //=
	TokenUpdatePipe          // |=
	TokenUpdateAdd           // +=
	TokenUpdateSub           // -=
	TokenUpdateMul           // *=
	TokenUpdateDiv           // /=
	TokenUpdateMod           // %=
	TokenAssign              // =
	TokenLParen              // (
	TokenRParen              // )
	TokenLBracket            // [
	TokenRBracket            // ]
	TokenLBrace              // {
	TokenRBrace              // }
	TokenColon               // :
	TokenSemicolon           // ;
	TokenQuestion            // ?
)

// Token is a lexer token with position information.
type Token struct {
	Kind  TokenKind
	Value string
	Pos   int // byte offset in source
	End   int // byte offset past end
}

func (t Token) String() string {
	if t.Value != "" {
		return t.Value
	}
	return tokenKindNames[t.Kind]
}

var tokenKindNames = map[TokenKind]string{
	TokenEOF: "EOF", TokenDot: ".", TokenDotDot: "..",
	TokenPipe: "|", TokenComma: ",", TokenPlus: "+", TokenMinus: "-",
	TokenStar: "*", TokenSlash: "/", TokenPercent: "%",
	TokenEq: "==", TokenNe: "!=", TokenLt: "<", TokenGt: ">",
	TokenLe: "<=", TokenGe: ">=", TokenAlt: "//",
	TokenAssign: "=", TokenLParen: "(", TokenRParen: ")",
	TokenLBracket: "[", TokenRBracket: "]",
	TokenLBrace: "{", TokenRBrace: "}", TokenColon: ":",
	TokenSemicolon: ";", TokenQuestion: "?",
	TokenUpdatePipe: "|=", TokenUpdateAdd: "+=", TokenUpdateSub: "-=",
	TokenUpdateMul: "*=", TokenUpdateDiv: "/=", TokenUpdateMod: "%=",
	TokenUpdateAlt: "//=",
}
