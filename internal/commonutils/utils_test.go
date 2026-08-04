// Copyright (c) Gigamon, Inc.

package commonutils

import (
	"strings"
	"testing"
)

func TestParseUpdateMonSessResponseSuccess(t *testing.T) {
	resp := []byte(`{"operationResponses":[{"entityType":"trafficMap","id":"abc-123","alias":"tm1","status":"success"}]}`)

	id, err := parseUpdateMonSessResponse(resp)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if id != "abc-123" {
		t.Fatalf("expected id abc-123, got %q", id)
	}
}

func TestParseUpdateMonSessResponseFailureIncludesBackendMessage(t *testing.T) {
	resp := []byte(`{
		"operationResponses":[
			{
				"entityType":"trafficMap",
				"id":"",
				"alias":"tm1",
				"status":"failed",
				"errorMessage":"subset/valueMax mismatch in ruleset"
			}
		]
	}`)

	_, err := parseUpdateMonSessResponse(resp)
	if err == nil {
		t.Fatalf("expected failure error, got nil")
	}

	errText := err.Error()
	if !strings.Contains(errText, "subset/valueMax mismatch in ruleset") {
		t.Fatalf("expected backend message in error, got: %s", errText)
	}
	if !strings.Contains(errText, "status=\"failed\"") {
		t.Fatalf("expected operation status in error, got: %s", errText)
	}
}

func TestParseUpdateMonSessResponseNoOperationResponsesIncludesTopLevelMessage(t *testing.T) {
	resp := []byte(`{"message":"backend validation failed: invalid traffic map encoding"}`)

	_, err := parseUpdateMonSessResponse(resp)
	if err == nil {
		t.Fatalf("expected failure error, got nil")
	}

	errText := err.Error()
	if !strings.Contains(errText, "backend validation failed: invalid traffic map encoding") {
		t.Fatalf("expected top-level backend message in error, got: %s", errText)
	}
}
