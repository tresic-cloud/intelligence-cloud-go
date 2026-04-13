package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
)

func TestJSONFormatter_Entity(t *testing.T) {
	entity := map[string]interface{}{
		"id":     "rsl_01HW3XYZ",
		"name":   "Acme Bakery",
		"active": true,
	}

	var buf bytes.Buffer
	f := output.NewJSONFormatter(&buf)

	if err := f.WriteEntity(entity); err != nil {
		t.Fatalf("WriteEntity: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if got["id"] != "rsl_01HW3XYZ" {
		t.Errorf("id = %v; want %v", got["id"], "rsl_01HW3XYZ")
	}
}

func TestJSONFormatter_List(t *testing.T) {
	items := []interface{}{
		map[string]interface{}{"id": "1", "name": "A"},
		map[string]interface{}{"id": "2", "name": "B"},
	}
	pageInfo := output.PageInfo{
		ItemsFetched: 2,
		PagesFetched: 1,
		HasNextPage:  false,
		NextPageToken: "",
	}

	var buf bytes.Buffer
	f := output.NewJSONFormatter(&buf)

	if err := f.WriteList(items, pageInfo); err != nil {
		t.Fatalf("WriteList: %v", err)
	}

	var got struct {
		Items    []json.RawMessage `json:"items"`
		PageInfo output.PageInfo   `json:"page_info"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 2 {
		t.Errorf("items count = %d; want 2", len(got.Items))
	}
	if got.PageInfo.ItemsFetched != 2 {
		t.Errorf("items_fetched = %d; want 2", got.PageInfo.ItemsFetched)
	}
	if got.PageInfo.HasNextPage {
		t.Error("has_next_page should be false")
	}
}

func TestJSONFormatter_Error(t *testing.T) {
	var stderr bytes.Buffer
	f := output.NewJSONFormatter(nil) // stdout unused for errors

	errInfo := output.ErrorInfo{
		Kind:      "authentication",
		Status:    401,
		Code:      "token_expired",
		Message:   "Bearer token expired",
		RequestID: "req_01HW",
		Operation: "GetMe",
	}

	if err := f.WriteError(&stderr, errInfo); err != nil {
		t.Fatalf("WriteError: %v", err)
	}

	var got struct {
		Error output.ErrorInfo `json:"error"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal stderr: %v", err)
	}
	if got.Error.Kind != "authentication" {
		t.Errorf("kind = %q; want %q", got.Error.Kind, "authentication")
	}
	if got.Error.Status != 401 {
		t.Errorf("status = %d; want 401", got.Error.Status)
	}
	if got.Error.Code != "token_expired" {
		t.Errorf("code = %q; want %q", got.Error.Code, "token_expired")
	}
	if got.Error.Message != "Bearer token expired" {
		t.Errorf("message = %q; want %q", got.Error.Message, "Bearer token expired")
	}
	if got.Error.RequestID != "req_01HW" {
		t.Errorf("request_id = %q; want %q", got.Error.RequestID, "req_01HW")
	}
	if got.Error.Operation != "GetMe" {
		t.Errorf("operation = %q; want %q", got.Error.Operation, "GetMe")
	}
}

func TestJSONFormatter_EmptyList(t *testing.T) {
	var buf bytes.Buffer
	f := output.NewJSONFormatter(&buf)

	if err := f.WriteList(nil, output.PageInfo{}); err != nil {
		t.Fatalf("WriteList: %v", err)
	}

	var got struct {
		Items    []json.RawMessage `json:"items"`
		PageInfo output.PageInfo   `json:"page_info"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Items == nil {
		t.Error("items should be an empty array, not null")
	}
	if len(got.Items) != 0 {
		t.Errorf("items len = %d; want 0", len(got.Items))
	}
}

func TestJSONFormatter_ListWithPagination(t *testing.T) {
	items := []interface{}{
		map[string]interface{}{"id": "1"},
	}
	pageInfo := output.PageInfo{
		ItemsFetched:  50,
		PagesFetched:  3,
		HasNextPage:   true,
		NextPageToken: "tok_next",
	}

	var buf bytes.Buffer
	f := output.NewJSONFormatter(&buf)

	if err := f.WriteList(items, pageInfo); err != nil {
		t.Fatalf("WriteList: %v", err)
	}

	var got struct {
		PageInfo output.PageInfo `json:"page_info"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !got.PageInfo.HasNextPage {
		t.Error("HasNextPage should be true")
	}
	if got.PageInfo.NextPageToken != "tok_next" {
		t.Errorf("NextPageToken = %q; want %q", got.PageInfo.NextPageToken, "tok_next")
	}
}
