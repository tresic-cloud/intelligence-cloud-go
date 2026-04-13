package intelligencecloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// maxErrBodyBytes caps the amount of response body read for error mapping.
const maxErrBodyBytes = 4096

// apiErrEnvelope matches the generated ErrorResponse shape: { "error": { ... } }.
type apiErrEnvelope struct {
	Error struct {
		Code      string                  `json:"code"`
		Message   string                  `json:"message"`
		Details   []generated.ErrorDetail `json:"details"`
		RequestID *string                 `json:"request_id"`
	} `json:"error"`
}

// mapResponseError reads a non-2xx HTTP response body and returns the
// appropriate typed SDK error. It is used by all resource-service wrappers.
// The caller must NOT have already consumed resp.Body.
func mapResponseError(resp *http.Response, operationID string) error {
	if resp == nil {
		return fmt.Errorf("intelligencecloud: nil response for operation %s", operationID)
	}

	var raw []byte
	if resp.Body != nil {
		raw, _ = io.ReadAll(io.LimitReader(resp.Body, maxErrBodyBytes))
		resp.Body.Close()
	}

	var env apiErrEnvelope
	_ = json.Unmarshal(raw, &env)

	code := env.Error.Code
	message := env.Error.Message
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	requestID := ""
	if env.Error.RequestID != nil {
		requestID = *env.Error.RequestID
	}
	if rid := resp.Header.Get("X-Request-Id"); rid != "" {
		requestID = rid
	}

	switch {
	case resp.StatusCode == 400 || resp.StatusCode == 422:
		var fieldErrors []FieldError
		for _, d := range env.Error.Details {
			fe := FieldError{}
			if d.Field != nil {
				fe.Field = *d.Field
			}
			if d.Code != nil {
				fe.Code = *d.Code
			}
			if d.Message != nil {
				fe.Message = *d.Message
			}
			fieldErrors = append(fieldErrors, fe)
		}
		return NewValidationError(resp.StatusCode, code, message, requestID, operationID, fieldErrors)

	case resp.StatusCode == 401:
		return NewAuthenticationError(resp.StatusCode, code, message, requestID, operationID)

	case resp.StatusCode == 403:
		return NewAuthorizationError(resp.StatusCode, code, message, requestID, operationID, nil)

	case resp.StatusCode == 404:
		return NewNotFoundError(resp.StatusCode, code, message, requestID, operationID, "", "")

	case resp.StatusCode == 409:
		return NewConflictError(resp.StatusCode, code, message, requestID, operationID)

	case resp.StatusCode == 429:
		retryAfter := time.Duration(0)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				retryAfter = time.Duration(secs) * time.Second
			}
		}
		return NewRateLimitError(resp.StatusCode, code, message, requestID, operationID, retryAfter)

	case resp.StatusCode >= 500:
		return NewServerError(resp.StatusCode, code, message, requestID, operationID)

	default:
		return NewUnexpectedError(resp.StatusCode, code, message, requestID, operationID, raw)
	}
}

// genClient creates a generated.Client wired to the SDK Client's base URL
// and HTTP client. The generated client is lightweight and safe to create
// per-call; it holds only a server string and *http.Client reference.
func (c *Client) genClient() (*generated.Client, error) {
	return generated.NewClient(
		c.baseURL.String(),
		generated.WithHTTPClient(c.httpClient),
	)
}

// decodeResponse reads and JSON-decodes the response body into v, then
// closes the body. It returns an error if decoding fails.
func decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// handleResponse checks the HTTP status code and either decodes a
// successful response or maps the error. For responses with a status code
// in [200,299] the body is decoded into v. Otherwise a typed SDK error is
// returned.
func handleResponse(resp *http.Response, operationID string, v interface{}) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if v == nil {
			// No response body expected (e.g. 204 No Content).
			if resp.Body != nil {
				resp.Body.Close()
			}
			return nil
		}
		return decodeResponse(resp, v)
	}
	return mapResponseError(resp, operationID)
}
