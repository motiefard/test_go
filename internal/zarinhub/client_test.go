package zarinhub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"card2sheba/internal/apperr"
)

func TestClientCardToIBANWrappedSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {"bankName":"bankName_value","card":"6037997512345678","deposit":"0101234567001","depositDescription":"depositDescription_value","depositOwners":"depositOwners_value","depositStatus":"depositStatus_value","englishBankName":"englishBankName_value","iban":"IR820540102680020817909002"},
			"meta": {"code":0,"errorMessage":null,"errorType":null,"errors":[],"isSuccess":true,"message":"ok","status":"success","trackId":"track-id"}
		}`))
	}))
	defer server.Close()

	outcome := NewClient(server.URL, "token", time.Second).CardToIBAN(context.Background(), "6037997512345678")
	if outcome.Error != nil {
		t.Fatalf("unexpected error: %v", outcome.Error)
	}
	if outcome.Result == nil || outcome.Result.IBAN != "IR820540102680020817909002" {
		t.Fatalf("unexpected result: %+v", outcome.Result)
	}
}

func TestClientCardToIBANWrappedFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {},
			"meta": {"trackId":null,"status":"error","isSuccess":false,"code":-1,"message":"failed","errorMessage":"invalid parameters","errorType":"validation_error","errors":[]}
		}`))
	}))
	defer server.Close()

	outcome := NewClient(server.URL, "token", time.Second).CardToIBAN(context.Background(), "6037997512345678")
	if outcome.Error == nil {
		t.Fatal("expected error")
	}
	appErr, ok := outcome.Error.(*apperr.Error)
	if !ok {
		t.Fatalf("error type = %T, want *apperr.Error", outcome.Error)
	}
	if appErr.HTTPStatus != http.StatusBadRequest || appErr.Code != apperr.CodeBadRequest {
		t.Fatalf("unexpected error: %+v", appErr)
	}
}
