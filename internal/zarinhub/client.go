package zarinhub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"card2sheba/internal/apperr"
)

type Result struct {
	BankName           string `json:"bankName"`
	Card               string `json:"card"`
	Deposit            string `json:"deposit"`
	DepositDescription string `json:"depositDescription"`
	DepositOwners      string `json:"depositOwners"`
	DepositStatus      string `json:"depositStatus"`
	EnglishBankName    string `json:"englishBankName"`
	IBAN               string `json:"iban"`
}

type CallOutcome struct {
	HTTPStatus  int
	RawBody     json.RawMessage
	Result      *Result
	Error       error
}

type Converter interface {
	CardToIBAN(ctx context.Context, card string) CallOutcome
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) CardToIBAN(ctx context.Context, card string) CallOutcome {
	if c.token == "" {
		return CallOutcome{
			Error: apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "ZarinHub credentials are not configured"),
		}
	}

	body, _ := json.Marshal(map[string]string{"card": card})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v5/Kyc/CardToIban", bytes.NewReader(body))
	if err != nil {
		return CallOutcome{Error: apperr.Wrap(http.StatusInternalServerError, apperr.CodeInternal, "unable to create request", err)}
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || isTimeout(err) {
			return CallOutcome{Error: apperr.Wrap(http.StatusGatewayTimeout, apperr.CodeTimeout, "ZarinHub request timed out", err)}
		}
		return CallOutcome{Error: apperr.Wrap(http.StatusBadGateway, apperr.CodeProviderError, "ZarinHub is unreachable", err)}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return CallOutcome{
			HTTPStatus: resp.StatusCode,
			Error:      apperr.Wrap(http.StatusBadGateway, apperr.CodeProviderError, "failed to read ZarinHub response", err),
		}
	}

	outcome := CallOutcome{HTTPStatus: resp.StatusCode, RawBody: json.RawMessage(raw)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		outcome.Error = mapHTTPError(resp.StatusCode, raw)
		return outcome
	}

	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		outcome.Error = apperr.New(http.StatusBadGateway, apperr.CodeProviderError, "ZarinHub returned an unexpected payload")
		return outcome
	}
	if result.IBAN == "" {
		outcome.Error = mapBusinessError(raw)
		return outcome
	}
	outcome.Result = &result
	return outcome
}

type errorEnvelope struct {
	Code    string `json:"code"`
	Type    string `json:"type"`
	EN      string `json:"en"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func mapHTTPError(status int, raw []byte) error {
	env := parseError(raw)
	code, message, httpStatus := translateZarinHub(env.Code, env.EN, status)
	if message == "" {
		message = "ZarinHub rejected the request"
	}
	return apperr.New(httpStatus, code, message)
}

func mapBusinessError(raw []byte) error {
	env := parseError(raw)
	code, message, httpStatus := translateZarinHub(env.Code, env.EN, http.StatusBadGateway)
	if message == "" {
		message = "ZarinHub did not return a SHEBA number"
	}
	return apperr.New(httpStatus, code, message)
}

func parseError(raw []byte) errorEnvelope {
	var env errorEnvelope
	_ = json.Unmarshal(raw, &env)
	return env
}

func translateZarinHub(code, en string, fallbackStatus int) (string, string, int) {
	key := strings.ToLower(strings.TrimSpace(code))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(en))
	}
	switch key {
	case "card-to-iban-v5-1", "servererror":
		return apperr.CodeServerError, "ZarinHub reported a server error", http.StatusBadGateway
	case "card-to-iban-v5-14", "requesttimeout":
		return apperr.CodeTimeout, "ZarinHub request timed out", http.StatusGatewayTimeout
	case "card-to-iban-v5-15", "providererror":
		return apperr.CodeProviderError, "ZarinHub provider error", http.StatusBadGateway
	case "card-to-iban-v5-2", "badrequest":
		return apperr.CodeBadRequest, "ZarinHub rejected the request parameters", http.StatusBadRequest
	case "card-to-iban-v5-3", "notfound":
		return apperr.CodeNotFound, "card information was not found", http.StatusNotFound
	case "card-to-iban-v5-5", "logicerror":
		return apperr.CodeLogicError, "ZarinHub could not process the request", http.StatusUnprocessableEntity
	case "card-to-iban-v5-6", "unauthorized":
		return apperr.CodeUnauthorized, "ZarinHub authentication failed", http.StatusUnauthorized
	case "card-to-iban-v5-7", "unavailable":
		return apperr.CodeUnavailable, "ZarinHub is currently unavailable", http.StatusServiceUnavailable
	}
	switch fallbackStatus {
	case http.StatusUnauthorized:
		return apperr.CodeUnauthorized, "ZarinHub authentication failed", http.StatusUnauthorized
	case http.StatusForbidden:
		return apperr.CodeAccessDenied, "access to ZarinHub was denied", http.StatusForbidden
	case http.StatusNotFound:
		return apperr.CodeNotFound, "card information was not found", http.StatusNotFound
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return apperr.CodeTimeout, "ZarinHub request timed out", http.StatusGatewayTimeout
	case http.StatusServiceUnavailable:
		return apperr.CodeUnavailable, "ZarinHub is currently unavailable", http.StatusServiceUnavailable
	default:
		return apperr.CodeProviderError, "ZarinHub rejected the request", http.StatusBadGateway
	}
}

func isTimeout(err error) bool {
	var nerr interface{ Timeout() bool }
	return errors.As(err, &nerr) && nerr.Timeout()
}
