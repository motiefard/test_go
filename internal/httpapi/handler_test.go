package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"card2sheba/internal/httpapi"
	"card2sheba/internal/inquiry"
	"card2sheba/internal/repository"
	"card2sheba/internal/zarinhub"
)

type stubClient struct{}

func (stubClient) CardToIBAN(_ context.Context, card string) zarinhub.CallOutcome {
	body, _ := json.Marshal(zarinhub.Result{
		BankName:        "ملت",
		Card:            card,
		Deposit:         "0010608837001",
		EnglishBankName: "SAMAN",
		IBAN:            "IR120570028080010608837001",
	})
	var result zarinhub.Result
	_ = json.Unmarshal(body, &result)
	return zarinhub.CallOutcome{HTTPStatus: http.StatusOK, RawBody: body, Result: &result}
}

type nopRepo struct{}

func (nopRepo) Save(context.Context, repository.AuditRecord) error { return nil }

func TestCardToShebaHTTPSuccessAndValidation(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := inquiry.NewService(stubClient{}, nopRepo{}, log)
	h := httpapi.NewHandler(svc, log)
	srv := httptest.NewServer(httpapi.NewRouter(h, "http://localhost:3000"))
	defer srv.Close()

	t.Run("success", func(t *testing.T) {
		body := []byte(`{"card":"6037997599939999"}`)
		resp, err := http.Post(srv.URL+"/api/v1/card-to-sheba", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		var got zarinhub.Result
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.IBAN != "IR120570028080010608837001" {
			t.Fatalf("unexpected payload: %+v", got)
		}
	})

	t.Run("validation", func(t *testing.T) {
		body := []byte(`{"card":"123"}`)
		resp, err := http.Post(srv.URL+"/api/v1/card-to-sheba", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status %d", resp.StatusCode)
		}
		var got map[string]map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got["error"]["code"] != "VALIDATION_ERROR" {
			t.Fatalf("unexpected error: %+v", got)
		}
	})
}
