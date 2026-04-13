package intelligencecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	genapi "github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// LocationConnector is a connector scoped to a single location.
type LocationConnector = genapi.LocationConnectorItem

// CompanyConnector is a connector scoped to a company, including its location
// association.
type CompanyConnector = genapi.CompanyConnectorItem

// ConnectorErrorSummaryItem is an aggregate error summary for connectors in
// error state across a company.
type ConnectorErrorSummaryItem = genapi.ConnectorErrorSummaryItem

// CompanyConnectorList holds both the paginated connector iterator and the
// aggregate error summary returned by the company-scoped connectors endpoint.
// The ErrorSummary reflects all connectors in error state across the entire
// company, regardless of pagination.
type CompanyConnectorList struct {
	// Iterator provides paginated access to company connectors.
	Iterator *Iterator[CompanyConnector]

	// ErrorSummary lists all connectors in error state across the company.
	// This is populated from the first page response and represents the
	// complete set, independent of pagination.
	ErrorSummary []ConnectorErrorSummaryItem
}

// ListByLocation returns a paginated iterator over connectors for the
// specified location. Filters can be applied using WithFilter with keys:
// "search", "status". Use WithPageSize to control page size.
func (s *ConnectorService) ListByLocation(ctx context.Context, locationID string, opts ...ListOption) *Iterator[LocationConnector] {
	var cfg listConfig
	for _, opt := range opts {
		opt.applyList(&cfg)
	}

	fetch := func(fetchCtx context.Context, pageToken string) ([]LocationConnector, string, error) {
		params := &genapi.ListLocationConnectorsParams{}

		if pageToken != "" {
			params.Cursor = &pageToken
		}
		if cfg.pageSize > 0 {
			params.Limit = &cfg.pageSize
		}
		if v, ok := cfg.filters["search"]; ok {
			params.Search = &v
		}
		if v, ok := cfg.filters["status"]; ok {
			params.Status = &v
		}

		httpReq, err := genapi.NewListLocationConnectorsRequest(s.client.baseURL.String(), locationID, params)
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to build location connector list request: %w", err)
		}
		httpReq = httpReq.WithContext(fetchCtx)

		resp, err := s.client.httpClient.Do(httpReq)
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: location connector list request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to read location connector list response: %w", err)
		}

		requestID := resp.Header.Get("X-Request-Id")

		if resp.StatusCode != http.StatusOK {
			return nil, "", mapStandardError(resp.StatusCode, body, requestID, "ListLocationConnectors")
		}

		var listResp genapi.LocationConnectorListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, "", fmt.Errorf("intelligencecloud: failed to decode location connector list response: %w", err)
		}

		nextToken := ""
		if listResp.Pagination.HasMore && listResp.Pagination.NextCursor != nil {
			nextToken = *listResp.Pagination.NextCursor
		}

		return listResp.Data, nextToken, nil
	}

	return newIterator(fetch, cfg.pageSize, cfg.maxItems)
}

// ListByCompany returns a CompanyConnectorList containing both a paginated
// iterator over company-scoped connectors and an aggregate error summary.
// The company connectors endpoint uses nested pagination
// (pagination.next_cursor + pagination.has_more) and carries a top-level
// errors array with connector-scoped aggregate diagnostics.
//
// The first page is fetched eagerly so that ErrorSummary is available
// immediately without calling Iterator.Next(). If the first page fetch fails,
// ErrorSummary will be empty and the error will surface when Iterator.Next()
// is called.
//
// Filters can be applied using WithFilter with keys: "search", "status",
// "type". Use WithPageSize to control page size.
func (s *ConnectorService) ListByCompany(ctx context.Context, companyID string, opts ...ListOption) *CompanyConnectorList {
	var cfg listConfig
	for _, opt := range opts {
		opt.applyList(&cfg)
	}

	result := &CompanyConnectorList{}

	// Eagerly fetch the first page to populate ErrorSummary.
	firstPage, firstNextToken, firstErr := fetchCompanyConnectorPage(
		ctx, s.client, companyID, "", &cfg,
	)

	if firstErr == nil {
		result.ErrorSummary = firstPage.errors
	}

	// Build a fetcher that returns the cached first page on the first call,
	// then fetches subsequent pages from the API.
	firstPageConsumed := false
	fetch := func(fetchCtx context.Context, pageToken string) ([]CompanyConnector, string, error) {
		if !firstPageConsumed {
			firstPageConsumed = true
			if firstErr != nil {
				return nil, "", firstErr
			}
			return firstPage.data, firstNextToken, nil
		}

		page, nextToken, err := fetchCompanyConnectorPage(
			fetchCtx, s.client, companyID, pageToken, &cfg,
		)
		if err != nil {
			return nil, "", err
		}
		return page.data, nextToken, nil
	}

	result.Iterator = newIterator(fetch, cfg.pageSize, cfg.maxItems)

	return result
}

// companyConnectorPage holds the parsed response from one page of company
// connectors, including the error summary.
type companyConnectorPage struct {
	data   []CompanyConnector
	errors []ConnectorErrorSummaryItem
}

// fetchCompanyConnectorPage fetches a single page of company connectors.
func fetchCompanyConnectorPage(
	ctx context.Context,
	c *Client,
	companyID string,
	pageToken string,
	cfg *listConfig,
) (companyConnectorPage, string, error) {
	params := &genapi.ListCompanyConnectorsParams{}

	if pageToken != "" {
		params.Cursor = &pageToken
	}
	if cfg.pageSize > 0 {
		params.Limit = &cfg.pageSize
	}
	if v, ok := cfg.filters["search"]; ok {
		params.Search = &v
	}
	if v, ok := cfg.filters["status"]; ok {
		params.Status = &v
	}
	if v, ok := cfg.filters["type"]; ok {
		params.Type = &v
	}

	httpReq, err := genapi.NewListCompanyConnectorsRequest(c.baseURL.String(), companyID, params)
	if err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: failed to build company connector list request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: company connector list request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: failed to read company connector list response: %w", err)
	}

	requestID := resp.Header.Get("X-Request-Id")

	if resp.StatusCode != http.StatusOK {
		return companyConnectorPage{}, "", mapStandardError(resp.StatusCode, body, requestID, "ListCompanyConnectors")
	}

	var listResp genapi.CompanyConnectorListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return companyConnectorPage{}, "", fmt.Errorf("intelligencecloud: failed to decode company connector list response: %w", err)
	}

	nextToken := ""
	if listResp.Pagination.HasMore && listResp.Pagination.NextCursor != nil {
		nextToken = *listResp.Pagination.NextCursor
	}

	return companyConnectorPage{
		data:   listResp.Data,
		errors: listResp.Errors,
	}, nextToken, nil
}
