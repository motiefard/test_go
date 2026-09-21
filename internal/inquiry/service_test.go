package inquiry_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"card2sheba/internal/apperr"
	"card2sheba/internal/inquiry"
	"card2sheba/internal/repository"
	"card2sheba/internal/zarinhub"
)

type fakeClient struct {
	outcome zarinhub.CallOutcome
	called  bool
	card    string
}

func (f *fakeClient) CardToIBAN(_ context.Context, card string) zarinhub.CallOutcome {
	f.called = true
	f.card = card
	return f.outcome
}

type memRepo struct {
	mu      sync.Mutex
	records []repository.AuditRecord
}

func (m *memRepo) Save(_ context.Context, rec repository.AuditRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, rec)
	return nil
}

func silentLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestConvertRejectsInvalidCardWithoutExternalCall(t *testing.T) {
	client := &fakeClient{}
	repo := &memRepo{}
	svc := inquiry.NewService(client, repo, silentLog())

	res := svc.Convert(context.Background(), "req-1", "127.0.0.1", "not-a-card")
	if client.called {
		t.Fatal("zarinhub must not be called for invalid cards")
	}
	if res.Error == nil || res.Error.Code != apperr.CodeValidation {
		t.Fatalf("expected validation error, got %+v", res.Error)
	}
	if len(repo.records) != 1 {
		t.Fatalf("expected audit row, got %d", len(repo.records))
	}
}

func TestConvertSuccessPersistsRawResponse(t *testing.T) {
	payload := &zarinhub.Result{
		BankName: "ملت",
		Card:     "6037997599939999",
		IBAN:     "IR120570028080010608837001",
	}
	raw, _ := json.Marshal(payload)
	status := 200
	client := &fakeClient{outcome: zarinhub.CallOutcome{
		HTTPStatus: status,
		RawBody:    raw,
		Result:     payload,
	}}
	repo := &memRepo{}
	svc := inquiry.NewService(client, repo, silentLog())

	res := svc.Convert(context.Background(), "req-2", "10.0.0.1", "6037-9975-9993-9999")
	if res.Error != nil {
		t.Fatalf("unexpected error: %+v", res.Error)
	}
	if !client.called || client.card != "6037997599939999" {
		t.Fatalf("client not called with normalized card: %+v", client)
	}
	if res.Payload.IBAN != payload.IBAN {
		t.Fatalf("iban mismatch")
	}
	if len(repo.records) != 1 || string(repo.records[0].RawResponse) != string(raw) {
		t.Fatalf("raw response not stored: %+v", repo.records)
	}
	if repo.records[0].MaskedCard != "603799******9999" {
		t.Fatalf("card not masked: %s", repo.records[0].MaskedCard)
	}
}

func TestConvertMapsProviderFailure(t *testing.T) {
	client := &fakeClient{outcome: zarinhub.CallOutcome{
		HTTPStatus: 401,
		Error:      apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "ZarinHub authentication failed"),
	}}
	repo := &memRepo{}
	svc := inquiry.NewService(client, repo, silentLog())

	res := svc.Convert(context.Background(), "req-3", "10.0.0.1", "6037997599939999")
	if res.Error == nil || res.Error.Code != apperr.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got %+v", res.Error)
	}
	if len(repo.records) != 1 || repo.records[0].RawResponse != nil {
		t.Fatalf("failed attempts must be audited without success payload")
	}
}
