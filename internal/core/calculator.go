// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Calculate parses finite float64 arithmetic only: + - * / and parentheses.
// No identifiers, function calls, code evaluation or platform effects exist.
// Input, recursion, numeric tokens and total token count are bounded.
func Calculate(expression string) (string, error) {
	if len(expression) > 512 {
		return "", fmt.Errorf("expression is too long")
	}
	expression = strings.TrimSpace(expression)
	expression = strings.TrimPrefix(expression, "=")
	expression = strings.NewReplacer("×", "*", "÷", "/", "−", "-", "（", "(", "）", ")").Replace(expression)
	p := arithmeticParser{source: expression}
	value := p.expression()
	p.space()
	if p.err != nil || p.position != len(p.source) {
		return "", fmt.Errorf("use bounded arithmetic and parentheses only")
	}
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return "", fmt.Errorf("result is not finite")
	}
	if value == 0 {
		value = 0
	}
	return strconv.FormatFloat(value, 'g', 12, 64), nil
}

type arithmeticParser struct {
	source                  string
	position, tokens, depth int
	err                     error
}

func (p *arithmeticParser) space() {
	for p.position < len(p.source) && (p.source[p.position] == ' ' || p.source[p.position] == '\t') {
		p.position++
	}
}
func (p *arithmeticParser) peek() byte {
	p.space()
	if p.position >= len(p.source) {
		return 0
	}
	return p.source[p.position]
}
func (p *arithmeticParser) take() {
	p.position++
	p.tokens++
	if p.tokens > 128 {
		p.err = fmt.Errorf("too many tokens")
	}
}
func (p *arithmeticParser) expression() float64 {
	value := p.term()
	for p.err == nil {
		op := p.peek()
		if op != '+' && op != '-' {
			break
		}
		p.take()
		other := p.term()
		if op == '+' {
			value += other
		} else {
			value -= other
		}
		p.finite(value)
	}
	return value
}
func (p *arithmeticParser) term() float64 {
	value := p.factor()
	for p.err == nil {
		op := p.peek()
		if op != '*' && op != '/' {
			break
		}
		p.take()
		other := p.factor()
		if op == '*' {
			value *= other
		} else if other == 0 {
			p.err = fmt.Errorf("division by zero")
		} else {
			value /= other
		}
		p.finite(value)
	}
	return value
}
func (p *arithmeticParser) finite(value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		p.err = fmt.Errorf("nonfinite result")
	}
}
func (p *arithmeticParser) factor() float64 {
	if p.err != nil {
		return 0
	}
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 32 {
		p.err = fmt.Errorf("expression is too deep")
		return 0
	}
	switch p.peek() {
	case '+':
		p.take()
		return p.factor()
	case '-':
		p.take()
		return -p.factor()
	case '(':
		p.take()
		value := p.expression()
		if p.peek() != ')' {
			p.err = fmt.Errorf("missing parenthesis")
			return 0
		}
		p.take()
		return value
	}
	start := p.position
	for p.position < len(p.source) {
		ch := p.source[p.position]
		if ch >= '0' && ch <= '9' || ch == '.' || ch == 'e' || ch == 'E' || (ch == '+' || ch == '-') && p.position > start && (p.source[p.position-1] == 'e' || p.source[p.position-1] == 'E') {
			p.position++
		} else {
			break
		}
	}
	if p.position == start || p.position-start > 64 {
		p.err = fmt.Errorf("invalid numeric token")
		return 0
	}
	p.tokens++
	if p.tokens > 128 {
		p.err = fmt.Errorf("too many tokens")
		return 0
	}
	value, err := strconv.ParseFloat(p.source[start:p.position], 64)
	if err != nil {
		p.err = err
		return 0
	}
	p.finite(value)
	return value
}
