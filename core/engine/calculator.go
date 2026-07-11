package engine

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxCalculatorInputRunes = 80

func CalculatorInputShouldCompose(input string) bool {
	normalized, ok := normalizeCalculatorInput(input)
	if !ok {
		return false
	}
	hasDigit := false
	hasOperator := false
	for _, r := range normalized {
		if r >= '0' && r <= '9' {
			hasDigit = true
		}
		switch r {
		case '+', '-', '*', '/', '%', '^', '=':
			hasOperator = true
		}
	}
	return hasDigit && hasOperator
}

func calculatorEntriesForInput(input string) []Entry {
	normalized, ok := normalizeCalculatorInput(input)
	if !ok || !CalculatorInputShouldCompose(normalized) {
		return nil
	}
	expression := strings.TrimSuffix(normalized, "=")
	if strings.TrimSpace(expression) == "" {
		return nil
	}
	value, err := parseCalculatorExpression(expression)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	result := formatCalculatorResult(value)
	if result == "" {
		return nil
	}
	return []Entry{{
		Reading: normalized,
		Text:    result,
		Kind:    "dynamic",
		Source:  "builtin-calculator",
		Comment: "计算",
		Weight:  dynamicCandidateWeightBase,
	}}
}

func normalizeCalculatorInput(input string) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" || utf8.RuneCountInString(input) > maxCalculatorInputRunes {
		return "", false
	}
	var builder strings.Builder
	for _, r := range input {
		switch {
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			continue
		case r == '+', r == '-', r == '*', r == '/', r == '%', r == '^', r == '.', r == '(', r == ')', r == '=':
			builder.WriteRune(r)
		case r == '×':
			builder.WriteRune('*')
		case r == '÷':
			builder.WriteRune('/')
		default:
			return "", false
		}
	}
	normalized := builder.String()
	if normalized == "" {
		return "", false
	}
	return normalized, true
}

func parseCalculatorExpression(input string) (float64, error) {
	parser := calculatorParser{input: input}
	value, err := parser.parseExpression()
	if err != nil {
		return 0, err
	}
	if parser.position != len(parser.input) {
		return 0, errors.New("unexpected trailing input")
	}
	return value, nil
}

type calculatorParser struct {
	input    string
	position int
}

func (p *calculatorParser) parseExpression() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for p.match('+') || p.match('-') {
		operator := p.input[p.position-1]
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if operator == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *calculatorParser) parseTerm() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for p.match('*') || p.match('/') || p.match('%') {
		operator := p.input[p.position-1]
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		switch operator {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, errors.New("division by zero")
			}
			left /= right
		case '%':
			if right == 0 {
				return 0, errors.New("modulo by zero")
			}
			left = math.Mod(left, right)
		}
	}
	return left, nil
}

func (p *calculatorParser) parsePower() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	if p.match('^') {
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		left = math.Pow(left, right)
	}
	return left, nil
}

func (p *calculatorParser) parseUnary() (float64, error) {
	if p.match('+') {
		return p.parseUnary()
	}
	if p.match('-') {
		value, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return -value, nil
	}
	return p.parsePrimary()
}

func (p *calculatorParser) parsePrimary() (float64, error) {
	if p.match('(') {
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		if !p.match(')') {
			return 0, errors.New("missing closing parenthesis")
		}
		return value, nil
	}
	start := p.position
	seenDigit := false
	seenDot := false
	for p.position < len(p.input) {
		ch := p.input[p.position]
		if ch >= '0' && ch <= '9' {
			seenDigit = true
			p.position++
			continue
		}
		if ch == '.' && !seenDot {
			seenDot = true
			p.position++
			continue
		}
		break
	}
	if !seenDigit {
		return 0, errors.New("expected number")
	}
	return strconv.ParseFloat(p.input[start:p.position], 64)
}

func (p *calculatorParser) match(ch byte) bool {
	if p.position >= len(p.input) || p.input[p.position] != ch {
		return false
	}
	p.position++
	return true
}

func formatCalculatorResult(value float64) string {
	if math.Abs(value) < 0.000000000001 {
		value = 0
	}
	rounded := math.Round(value)
	if math.Abs(value-rounded) < 0.000000000001 {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
