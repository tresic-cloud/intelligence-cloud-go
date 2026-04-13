// Package transport implements the HTTP RoundTripper that composes auth,
// retry, tracing, logging, redaction, and error mapping for the SDK client.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	ic "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/auth"
	"github.com/tresic-cloud/intelligence-cloud-go/internal/version"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tokenSafetyMargin is subtracted from a token's expiry time when deciding
// whether the cached token is still valid.
const tokenSafetyMargin = 30 * time.Second

// Transport is an http.RoundTripper that injects authentication, performs
// retries with backoff, emits OTel spans, logs with slog, redacts sensitive
// data, maps HTTP statuses to typed errors, replays request bodies, sets the
// User-Agent header, and captures X-Request-Id from responses.
type Transport struct {
	// Base is the wrapped transport. If nil, http.DefaultTransport is used.
	Base http.RoundTripper
	// Provider is the credential provider for bearer-token injection. Required.
	Provider auth.CredentialProvider
	// Policy controls retry behaviour. Use intelligencecloud.DefaultRetryPolicy()
	// or intelligencecloud.NoRetry().
	Policy ic.RetryPolicy
	// TracerProv is the OTel tracer provider. If nil, a no-op tracer is used.
	TracerProv trace.TracerProvider
	// Propagator is the OTel context propagator. If nil, a composite W3C
	// trace-context + baggage propagator is used.
	Propagator propagation.TextMapPropagator
	// Logger is the slog logger. If nil, logging is disabled.
	Logger *slog.Logger
	// UserAgent overrides the User-Agent header. If empty, no override is applied.
	UserAgent string

	// mu guards cached token state.
	mu        sync.Mutex
	cached    auth.Token
	cachedExp time.Time
}

// base returns the underlying transport, defaulting to http.DefaultTransport.
func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// tracer returns the OTel tracer for the SDK.
func (t *Transport) tracer() trace.Tracer {
	tp := t.TracerProv
	if tp == nil {
		tp = trace.NewNoopTracerProvider()
	}
	return tp.Tracer("github.com/tresic-cloud/intelligence-cloud-go")
}

// propagator returns the configured propagator, defaulting to W3C.
func (t *Transport) propagator() propagation.TextMapPropagator {
	if t.Propagator != nil {
		return t.Propagator
	}
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// logger returns the configured logger or a no-op logger.
func (t *Transport) logger() *slog.Logger {
	if t.Logger != nil {
		return t.Logger
	}
	return slog.New(discardHandler{})
}

// discardHandler is a no-op slog.Handler.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h discardHandler) WithGroup(string) slog.Handler            { return h }

// getToken returns a valid token, using the cache when possible. It applies
// a 30-second safety margin to the token expiry. If forceRefresh is true,
// the cache is bypassed.
func (t *Transport) getToken(ctx context.Context, forceRefresh bool) (auth.Token, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !forceRefresh && t.cached.AccessToken != "" {
		// A zero ExpiresAt means the token never expires.
		if t.cachedExp.IsZero() || time.Now().Before(t.cachedExp) {
			return t.cached, nil
		}
	}

	tok, err := t.Provider.Token(ctx)
	if err != nil {
		return auth.Token{}, err
	}

	t.cached = tok
	if tok.ExpiresAt.IsZero() {
		t.cachedExp = time.Time{} // never expires
	} else {
		t.cachedExp = tok.ExpiresAt.Add(-tokenSafetyMargin)
	}

	return tok, nil
}

// invalidateToken clears the cached token.
func (t *Transport) invalidateToken() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cached = auth.Token{}
	t.cachedExp = time.Time{}
}

// RoundTrip implements http.RoundTripper. It orchestrates the full
// per-attempt flow: auth injection, retry, OTel spans, slog logging,
// redaction, error mapping, body replay, UA header, and RequestID capture.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Resolve operation context for span naming and attributes.
	oc, hasOC := OperationContextFromContext(ctx)
	operationID := "unknown"
	urlTemplate := req.URL.Path
	resource := ""
	if hasOC {
		if oc.OperationID != "" {
			operationID = oc.OperationID
		}
		if oc.URLTemplate != "" {
			urlTemplate = oc.URLTemplate
		}
		resource = oc.Resource
	}

	// Determine span name.
	var spanName string
	if operationID != "unknown" {
		spanName = "intelligencecloud." + operationID
	} else {
		spanName = "intelligencecloud.http." + strings.ToLower(req.Method)
	}

	// Start OTel span (covers all retry attempts).
	tr := t.tracer()
	ctx, span := tr.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	log := t.logger()

	// Buffer body for replay on retries.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("intelligencecloud: failed to buffer request body: %w", err)
		}
	}

	bodySize := -1
	if bodyBytes != nil {
		bodySize = len(bodyBytes)
	}

	// Compute server attributes from request URL.
	host := req.URL.Hostname()
	port := 0
	if p := req.URL.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}

	// Set initial span attributes.
	redactedURL := RedactURL(req.URL)
	ua := t.UserAgent
	if ua == "" {
		ua = req.Header.Get("User-Agent")
	}

	span.SetAttributes(ic.HTTPRequestAttrs(req.Method, urlTemplate, redactedURL, ua, bodySize)...)
	span.SetAttributes(ic.ServerAttrs(host, port)...)
	span.SetAttributes(ic.SDKAttrs(operationID, resource, "", version.Version, 0, "")...)

	policy := t.Policy
	maxAttempts := policy.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if !policy.Enabled {
		maxAttempts = 1
	}

	var (
		lastResp *http.Response
		lastErr  error
		reqID    string
	)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check context before each attempt.
		if ctx.Err() != nil {
			wrapErr := fmt.Errorf("intelligencecloud: %w", ctx.Err())
			span.RecordError(wrapErr)
			setSpanStatusFromCtx(span, ctx)
			log.ErrorContext(ctx, "transport error",
				"operation_id", operationID,
				"error", wrapErr.Error(),
				"attempts", attempt,
			)
			return nil, wrapErr
		}

		// Sleep backoff on retry attempts (not the first attempt).
		if attempt > 0 {
			backoff := computeBackoff(policy, attempt-1)

			// Check if Retry-After should override.
			if raDelay := retryAfterDelay(policy, lastResp); raDelay > 0 {
				backoff = raDelay
			}

			cause := retryCause(lastResp, lastErr)

			log.WarnContext(ctx, "retry sleeping",
				"operation_id", operationID,
				"attempt", attempt,
				"backoff_ms", backoff.Milliseconds(),
				"cause", cause,
			)

			// Interruptible sleep.
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				wrapErr := fmt.Errorf("intelligencecloud: %w", ctx.Err())
				span.RecordError(wrapErr)
				setSpanStatusFromCtx(span, ctx)
				log.ErrorContext(ctx, "transport error",
					"operation_id", operationID,
					"error", wrapErr.Error(),
					"attempts", attempt,
				)
				return nil, wrapErr
			case <-timer.C:
			}
		}

		// Get token.
		tok, err := t.getToken(ctx, false)
		if err != nil {
			authErr := ic.NewAuthenticationError(0, "", err.Error(), reqID, operationID)
			span.RecordError(authErr)
			span.SetStatus(codes.Error, authErr.Error())
			log.ErrorContext(ctx, "transport error",
				"operation_id", operationID,
				"error", authErr.Error(),
				"attempts", attempt+1,
			)
			return nil, authErr
		}

		// Clone request for this attempt.
		clone := req.Clone(ctx)
		clone.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		if t.UserAgent != "" {
			clone.Header.Set("User-Agent", t.UserAgent)
		}

		// Replay body.
		if bodyBytes != nil {
			clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			clone.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
			clone.ContentLength = int64(len(bodyBytes))
		}

		// Inject trace-context headers.
		t.propagator().Inject(ctx, propagation.HeaderCarrier(clone.Header))

		log.DebugContext(ctx, "request start",
			"operation_id", operationID,
			"method", req.Method,
			"url.template", urlTemplate,
			"attempt", attempt,
		)

		start := time.Now()
		resp, roundTripErr := t.base().RoundTrip(clone)
		elapsed := time.Since(start)

		// Capture X-Request-Id.
		if resp != nil {
			if id := resp.Header.Get("X-Request-Id"); id != "" {
				reqID = id
			}
		}

		// Update span attributes with response info.
		if resp != nil {
			protoVer := resp.Proto
			// Normalize: "HTTP/1.1" -> "1.1", "HTTP/2.0" -> "2"
			protoVer = strings.TrimPrefix(protoVer, "HTTP/")
			if protoVer == "2.0" {
				protoVer = "2"
			}

			respBodySize := -1
			if resp.ContentLength >= 0 {
				respBodySize = int(resp.ContentLength)
			}

			span.SetAttributes(ic.HTTPResponseAttrs(resp.StatusCode, respBodySize, protoVer)...)
			if reqID != "" {
				span.SetAttributes(attribute.String("intelligencecloud.request_id", reqID))
			}
			span.SetAttributes(attribute.Int("intelligencecloud.retry_attempt", attempt))
			if attempt > 0 {
				span.SetAttributes(attribute.String("intelligencecloud.retry_cause", retryCause(lastResp, lastErr)))
			}

			log.DebugContext(ctx, "response received",
				"operation_id", operationID,
				"status", resp.StatusCode,
				"request_id", reqID,
				"duration_ms", elapsed.Milliseconds(),
			)
		}

		// Handle transport error.
		if roundTripErr != nil {
			lastResp = nil
			lastErr = roundTripErr

			if attempt+1 < maxAttempts && shouldRetry(policy, nil, roundTripErr) {
				// Retryable transport error.
				span.AddEvent("retry.attempt_failed", trace.WithAttributes(
					attribute.Int("retry_attempt", attempt),
					attribute.Int("backoff_ms", int(computeBackoff(policy, attempt).Milliseconds())),
					attribute.String("cause", "transport_error"),
				))
				continue
			}

			// Terminal transport error.
			span.RecordError(roundTripErr)
			span.SetStatus(codes.Error, roundTripErr.Error())
			if attempt > 0 {
				span.AddEvent("retry.giving_up", trace.WithAttributes(
					attribute.Int("total_attempts", attempt+1),
					attribute.String("last_cause", "transport_error"),
				))
			}
			log.ErrorContext(ctx, "transport error",
				"operation_id", operationID,
				"error", roundTripErr.Error(),
				"attempts", attempt+1,
			)
			return nil, fmt.Errorf("intelligencecloud: transport error: %w", roundTripErr)
		}

		// Handle 2xx/3xx success.
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			// Span status: Unset (default).
			return resp, nil
		}

		// Handle 401 — special re-fetch path (separate from retry budget).
		if resp.StatusCode == 401 {
			authResp, authErr := t.handle401(ctx, req, resp, bodyBytes, tok, attempt, span, log, operationID, urlTemplate)
			if authResp != nil || authErr != nil {
				return authResp, authErr
			}
			// handle401 returns nil, nil if it could not resolve — fall through
			// to error mapping (should not happen in practice).
		}

		// Handle retryable statuses (429, 5xx).
		if shouldRetry(policy, resp, nil) {
			lastResp = resp
			lastErr = nil

			if attempt+1 < maxAttempts {
				cause := retryCause(resp, nil)
				raDelay := retryAfterDelay(policy, resp)
				backoffMs := int(computeBackoff(policy, attempt).Milliseconds())

				evAttrs := []attribute.KeyValue{
					attribute.Int("retry_attempt", attempt),
					attribute.Int("backoff_ms", backoffMs),
					attribute.String("cause", cause),
				}
				if raDelay > 0 {
					evAttrs = append(evAttrs, attribute.Int("retry_after_ms", int(raDelay.Milliseconds())))
				}
				span.AddEvent("retry.attempt_failed", trace.WithAttributes(evAttrs...))

				log.WarnContext(ctx, "backend returned error",
					"operation_id", operationID,
					"status", resp.StatusCode,
					"code", "",
					"request_id", reqID,
				)

				// Drain and close response body so connection can be reused.
				drainBody(resp)
				continue
			}

			// Retries exhausted.
			cause := retryCause(resp, nil)
			span.AddEvent("retry.giving_up", trace.WithAttributes(
				attribute.Int("total_attempts", attempt+1),
				attribute.String("last_cause", cause),
			))
		}

		// Terminal response: map to typed error.
		typedErr := t.mapError(resp, reqID, operationID)

		// Set span status.
		if resp.StatusCode >= 500 {
			span.RecordError(typedErr)
			span.SetStatus(codes.Error, typedErr.Error())
			span.SetAttributes(attribute.String("error.type", errorTypeName(typedErr)))
		}
		// 4xx: Unset per semconv.

		log.ErrorContext(ctx, "transport error",
			"operation_id", operationID,
			"error", typedErr.Error(),
			"attempts", attempt+1,
		)

		return nil, typedErr
	}

	// Should not reach here, but handle edge case: retries exhausted for
	// transport error captured by lastErr.
	if lastErr != nil {
		return nil, fmt.Errorf("intelligencecloud: transport error after retries: %w", lastErr)
	}

	// Retries exhausted for retryable HTTP status.
	if lastResp != nil {
		typedErr := t.mapError(lastResp, reqID, operationID)
		if lastResp.StatusCode >= 500 {
			span.RecordError(typedErr)
			span.SetStatus(codes.Error, typedErr.Error())
			span.SetAttributes(attribute.String("error.type", errorTypeName(typedErr)))
		}
		log.ErrorContext(ctx, "transport error",
			"operation_id", operationID,
			"error", typedErr.Error(),
			"attempts", maxAttempts,
		)
		return nil, typedErr
	}

	return nil, fmt.Errorf("intelligencecloud: unexpected end of retry loop")
}

// handle401 implements the 401 re-fetch path. It is separate from the
// general retry budget. Returns (resp, nil) on success, (nil, error) on
// terminal failure, or (nil, nil) if caller should fall through.
func (t *Transport) handle401(
	ctx context.Context,
	origReq *http.Request,
	resp *http.Response,
	bodyBytes []byte,
	prevToken auth.Token,
	attempt int,
	span trace.Span,
	log *slog.Logger,
	operationID, urlTemplate string,
) (*http.Response, error) {
	reqID := ""
	if id := resp.Header.Get("X-Request-Id"); id != "" {
		reqID = id
	}

	// Drain the 401 response body.
	drainBody(resp)

	// Invalidate cache and re-fetch token.
	t.invalidateToken()
	newTok, err := t.Provider.Token(ctx)
	if err != nil {
		authErr := ic.NewAuthenticationError(401, "", err.Error(), reqID, operationID)
		span.RecordError(authErr)
		span.SetStatus(codes.Error, authErr.Error())
		log.ErrorContext(ctx, "transport error",
			"operation_id", operationID,
			"error", authErr.Error(),
			"attempts", attempt+1,
		)
		return nil, authErr
	}

	// If provider returns the same token, give up immediately.
	if newTok.AccessToken == prevToken.AccessToken {
		authErr := ic.NewAuthenticationError(401, "token_unchanged",
			"provider returned the same token after 401", reqID, operationID)
		span.RecordError(authErr)
		span.SetStatus(codes.Error, authErr.Error())
		log.ErrorContext(ctx, "transport error",
			"operation_id", operationID,
			"error", authErr.Error(),
			"attempts", attempt+1,
		)
		return nil, authErr
	}

	// Cache the new token.
	t.mu.Lock()
	t.cached = newTok
	if newTok.ExpiresAt.IsZero() {
		t.cachedExp = time.Time{}
	} else {
		t.cachedExp = newTok.ExpiresAt.Add(-tokenSafetyMargin)
	}
	t.mu.Unlock()

	// Retry once with the new token.
	clone := origReq.Clone(ctx)
	clone.Header.Set("Authorization", "Bearer "+newTok.AccessToken)
	if t.UserAgent != "" {
		clone.Header.Set("User-Agent", t.UserAgent)
	}
	if bodyBytes != nil {
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		clone.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		clone.ContentLength = int64(len(bodyBytes))
	}
	t.propagator().Inject(ctx, propagation.HeaderCarrier(clone.Header))

	log.DebugContext(ctx, "request start",
		"operation_id", operationID,
		"method", origReq.Method,
		"url.template", urlTemplate,
		"attempt", attempt+1,
	)

	retryResp, retryErr := t.base().RoundTrip(clone)

	if retryResp != nil {
		if id := retryResp.Header.Get("X-Request-Id"); id != "" {
			reqID = id
		}
	}

	if retryErr != nil {
		authErr := ic.NewAuthenticationError(0, "", retryErr.Error(), reqID, operationID)
		span.RecordError(authErr)
		span.SetStatus(codes.Error, authErr.Error())
		return nil, authErr
	}

	// If still 401, give up.
	if retryResp.StatusCode == 401 {
		drainBody(retryResp)
		authErr := ic.NewAuthenticationError(401, "token_rejected",
			"fresh token also returned 401", reqID, operationID)
		span.RecordError(authErr)
		span.SetStatus(codes.Error, authErr.Error())
		log.ErrorContext(ctx, "transport error",
			"operation_id", operationID,
			"error", authErr.Error(),
			"attempts", attempt+2,
		)
		return nil, authErr
	}

	// Success or non-401 error.
	if retryResp.StatusCode >= 200 && retryResp.StatusCode < 400 {
		return retryResp, nil
	}

	// Non-401 error: map to typed error.
	typedErr := t.mapError(retryResp, reqID, operationID)
	if retryResp.StatusCode >= 500 {
		span.RecordError(typedErr)
		span.SetStatus(codes.Error, typedErr.Error())
		span.SetAttributes(attribute.String("error.type", errorTypeName(typedErr)))
	}
	return nil, typedErr
}

// apiErrorBody is the expected JSON error response from the backend.
type apiErrorBody struct {
	Code           string           `json:"code"`
	Message        string           `json:"message"`
	FieldErrors    []ic.FieldError  `json:"field_errors"`
	RequiredScopes []string         `json:"required_scopes"`
	ResourceType   string           `json:"resource_type"`
	ResourceID     string           `json:"resource_id"`
	RetryAfter     *float64         `json:"retry_after"`
}

// parseErrorBody attempts to parse the response body as a JSON error.
// It reads up to 4 KiB. Returns a zero struct and the raw body bytes
// if parsing fails.
func parseErrorBody(resp *http.Response) (apiErrorBody, []byte) {
	if resp.Body == nil {
		return apiErrorBody{}, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	if err != nil || len(raw) == 0 {
		return apiErrorBody{}, raw
	}
	var body apiErrorBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return apiErrorBody{}, raw
	}
	return body, raw
}

// mapError maps an HTTP response to the appropriate typed error.
func (t *Transport) mapError(resp *http.Response, requestID, operationID string) error {
	body, rawBody := parseErrorBody(resp)

	code := body.Code
	message := body.Message
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	switch {
	case resp.StatusCode == 400 || resp.StatusCode == 422:
		return ic.NewValidationError(resp.StatusCode, code, message, requestID, operationID, body.FieldErrors)

	case resp.StatusCode == 401:
		return ic.NewAuthenticationError(resp.StatusCode, code, message, requestID, operationID)

	case resp.StatusCode == 403:
		return ic.NewAuthorizationError(resp.StatusCode, code, message, requestID, operationID, body.RequiredScopes)

	case resp.StatusCode == 404:
		return ic.NewNotFoundError(resp.StatusCode, code, message, requestID, operationID, body.ResourceType, body.ResourceID)

	case resp.StatusCode == 409:
		return ic.NewConflictError(resp.StatusCode, code, message, requestID, operationID)

	case resp.StatusCode == 429:
		retryAfter := time.Duration(0)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfter = parseRetryAfter(ra)
		}
		return ic.NewRateLimitError(resp.StatusCode, code, message, requestID, operationID, retryAfter)

	case resp.StatusCode >= 500:
		return ic.NewServerError(resp.StatusCode, code, message, requestID, operationID)

	default:
		return ic.NewUnexpectedError(resp.StatusCode, code, message, requestID, operationID, rawBody)
	}
}

// errorTypeName returns the short type name for a typed error, suitable
// for the error.type span attribute.
func errorTypeName(err error) string {
	switch err.(type) {
	case *ic.AuthenticationError:
		return "AuthenticationError"
	case *ic.AuthorizationError:
		return "AuthorizationError"
	case *ic.ValidationError:
		return "ValidationError"
	case *ic.NotFoundError:
		return "NotFoundError"
	case *ic.ConflictError:
		return "ConflictError"
	case *ic.RateLimitError:
		return "RateLimitError"
	case *ic.ServerError:
		return "ServerError"
	case *ic.UnexpectedError:
		return "UnexpectedError"
	default:
		return "UnknownError"
	}
}

// setSpanStatusFromCtx sets the span status based on a context error.
func setSpanStatusFromCtx(span trace.Span, ctx context.Context) {
	err := ctx.Err()
	if err == nil {
		return
	}
	switch {
	case ctx.Err() == context.Canceled:
		span.SetStatus(codes.Error, "context cancelled")
	case ctx.Err() == context.DeadlineExceeded:
		span.SetStatus(codes.Error, "context deadline exceeded")
	default:
		span.SetStatus(codes.Error, err.Error())
	}
}

// drainBody reads and closes the response body so the connection can be reused.
func drainBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}
}
