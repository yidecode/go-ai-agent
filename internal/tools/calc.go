package tools

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Eval evaluates an arithmetic expression such as "(12.5*3 - 4) / 2".
// Supported: + - * / % ^, parentheses, unary minus, decimal numbers.
func Eval(expr string) (float64, error) {
	p := &parser{s: strings.ReplaceAll(expr, " ", ""), pos: 0}
	if p.s == "" {
		return 0, fmt.Errorf("empty expression")
	}
	v, err := p.parseSum()
	if err != nil {
		return 0, err
	}
	if p.pos != len(p.s) {
		return 0, fmt.Errorf("unexpected character %q at position %d", rune(p.s[p.pos]), p.pos)
	}
	return v, nil
}

type parser struct {
	s   string
	pos int
}

func (p *parser) parseSum() (float64, error) {
	v, err := p.parseProduct()
	if err != nil {
		return 0, err
	}
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case '+':
			p.pos++
			rhs, err := p.parseProduct()
			if err != nil {
				return 0, err
			}
			v += rhs
		case '-':
			p.pos++
			rhs, err := p.parseProduct()
			if err != nil {
				return 0, err
			}
			v -= rhs
		default:
			return v, nil
		}
	}
	return v, nil
}

func (p *parser) parseProduct() (float64, error) {
	v, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case '*':
			p.pos++
			rhs, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			v *= rhs
		case '/':
			p.pos++
			rhs, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			if rhs == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			v /= rhs
		case '%':
			p.pos++
			rhs, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			if rhs == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			v = float64(int64(v) % int64(rhs))
		default:
			return v, nil
		}
	}
	return v, nil
}

func (p *parser) parsePower() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	if p.pos < len(p.s) && p.s[p.pos] == '^' {
		p.pos++
		exp, err := p.parsePower() // right-associative
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

func (p *parser) parseUnary() (float64, error) {
	if p.pos < len(p.s) && p.s[p.pos] == '-' {
		p.pos++
		v, err := p.parseUnary()
		return -v, err
	}
	if p.pos < len(p.s) && p.s[p.pos] == '+' {
		p.pos++
		return p.parseUnary()
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (float64, error) {
	if p.pos < len(p.s) && p.s[p.pos] == '(' {
		p.pos++
		v, err := p.parseSum()
		if err != nil {
			return 0, err
		}
		if p.pos >= len(p.s) || p.s[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return v, nil
	}
	start := p.pos
	for p.pos < len(p.s) && (unicode.IsDigit(rune(p.s[p.pos])) || p.s[p.pos] == '.') {
		p.pos++
	}
	if start == p.pos {
		return 0, fmt.Errorf("expected number at position %d", p.pos)
	}
	return strconv.ParseFloat(p.s[start:p.pos], 64)
}
