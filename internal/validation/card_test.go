package validation_test

import (
	"testing"

	"card2sheba/internal/apperr"
	"card2sheba/internal/validation"
)

func TestNormalizeCard(t *testing.T) {
	t.Parallel()
	valid := "6037997599939999"
	if !luhnLocal(valid) {
		t.Fatalf("fixture must pass luhn")
	}

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr string
	}{
		{name: "empty", in: "  ", wantErr: apperr.CodeValidation},
		{name: "letters", in: "60379975abcd9999", wantErr: apperr.CodeValidation},
		{name: "short", in: "603799759999", wantErr: apperr.CodeValidation},
		{name: "luhn fail", in: "6037997599999999", wantErr: apperr.CodeValidation},
		{name: "spaces and dashes", in: "6037-9975-9993-9999", want: "6037997599939999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validation.NormalizeCard(tt.in)
			if tt.wantErr != "" {
				app, ok := apperr.As(err)
				if !ok || app.Code != tt.wantErr {
					t.Fatalf("expected %s, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %s want %s", got, tt.want)
			}
		})
	}
}

func TestMask(t *testing.T) {
	t.Parallel()
	got := validation.Mask("6037997599939999")
	want := "603799******9999"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func luhnLocal(s string) bool {
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
