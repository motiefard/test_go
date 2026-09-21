package inquiry

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"card2sheba/internal/apperr"
	"card2sheba/internal/repository"
	"card2sheba/internal/validation"
	"card2sheba/internal/zarinhub"
)

type Service struct {
	client zarinhub.Converter
	repo   repository.AuditRepository
	log    *slog.Logger
}

func NewService(client zarinhub.Converter, repo repository.AuditRepository, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{client: client, repo: repo, log: log}
}

type Result struct {
	HTTPStatus         int
	ExternalHTTPStatus *int
	Error              *apperr.Error
	Payload            *zarinhub.Result
	RawResponse        json.RawMessage
	MaskedCard         string
}

func (s *Service) Convert(ctx context.Context, requestID, clientIP, rawCard string) Result {
	started := time.Now()
	card, err := validation.NormalizeCard(rawCard)
	masked := validation.Mask(card)
	if card == "" {
		masked = "****"
	}

	if err != nil {
		appErr, _ := apperr.As(err)
		res := Result{HTTPStatus: appErr.HTTPStatus, Error: appErr, MaskedCard: masked}
		s.persist(ctx, requestID, clientIP, started, res)
		s.log.Info("inquiry_rejected", "request_id", requestID, "reason", appErr.Code, "masked_card", masked)
		return res
	}

	outcome := s.client.CardToIBAN(ctx, card)
	res := Result{
		ExternalHTTPStatus: statusPtr(outcome.HTTPStatus),
		RawResponse:        outcome.RawBody,
		MaskedCard:         masked,
		Payload:            outcome.Result,
	}
	if outcome.Error != nil {
		if appErr, ok := apperr.As(outcome.Error); ok {
			res.Error = appErr
			res.HTTPStatus = appErr.HTTPStatus
		} else {
			res.Error = apperr.Wrap(http.StatusInternalServerError, apperr.CodeInternal, "internal error", outcome.Error)
			res.HTTPStatus = http.StatusInternalServerError
		}
		s.persist(ctx, requestID, clientIP, started, res)
		s.log.Warn("inquiry_failed",
			"request_id", requestID,
			"masked_card", masked,
			"error_code", res.Error.Code,
			"external_status", outcome.HTTPStatus,
			"duration_ms", time.Since(started).Milliseconds(),
		)
		return res
	}

	res.HTTPStatus = http.StatusOK
	s.persist(ctx, requestID, clientIP, started, res)
	s.log.Info("inquiry_succeeded",
		"request_id", requestID,
		"masked_card", masked,
		"external_status", outcome.HTTPStatus,
		"duration_ms", time.Since(started).Milliseconds(),
	)
	return res
}

func (s *Service) persist(ctx context.Context, requestID, clientIP string, started time.Time, res Result) {
	errCode := ""
	if res.Error != nil {
		errCode = res.Error.Code
	}
	raw := res.RawResponse
	if res.Error != nil {
		raw = nil
	}
	err := s.repo.Save(context.WithoutCancel(ctx), repository.AuditRecord{
		RequestID:          requestID,
		InquiredAt:         started,
		Duration:           time.Since(started),
		ClientIP:           clientIP,
		HTTPStatus:         res.HTTPStatus,
		ExternalHTTPStatus: res.ExternalHTTPStatus,
		MaskedCard:         res.MaskedCard,
		ErrorCode:          errCode,
		RawResponse:        raw,
	})
	if err != nil {
		s.log.Error("audit_persist_failed", "request_id", requestID, "error", "persistence_error")
	}
}

func statusPtr(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}
