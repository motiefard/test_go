package validation

import (
	"net/http"
	"strings"
	"unicode"

	"card2sheba/internal/apperr"
)

func NormalizeCard(raw string) (string, error) {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if unicode.IsSpace(r) || r == '-' {
			continue
		}
		if r < '0' || r > '9' {
			return "", apperr.New(http.StatusBadRequest, apperr.CodeValidation, "card number must contain only digits")
		}
		b.WriteRune(r)
	}
	card := b.String()
	if card == "" {
		return "", apperr.New(http.StatusBadRequest, apperr.CodeValidation, "card number is required")
	}
	if len(card) != 16 {
		return "", apperr.New(http.StatusBadRequest, apperr.CodeValidation, "card number must be 16 digits")
	}
	if !luhn(card) {
		return "", apperr.New(http.StatusBadRequest, apperr.CodeValidation, "card number is structurally invalid")
	}
	return card, nil
}

func Mask(card string) string {
	if len(card) < 10 {
		return "****"
	}
	return card[:6] + strings.Repeat("*", len(card)-10) + card[len(card)-4:]
}

func luhn(s string) bool {
	sum := 0
	alt := false
	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
