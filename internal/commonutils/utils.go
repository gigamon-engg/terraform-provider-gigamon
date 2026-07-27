// Copyright (c) Gigamon, Inc.

package commonutils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"terraform-provider-gigamon/internal/fmclient"
)

// -----------------------------------------------------------------------------
// Types used for Monitoring Session updates
// -----------------------------------------------------------------------------

// UpdateReq is the generic request used to update a Monitoring Session.
type UpdateReq struct {
	Requests []UpdateObject `json:"requests"`
}

type UpdateObject struct {
	EntityType  string `json:"entityType"`
	Operation   string `json:"operation"`
	ReferenceId string `json:"referenceId,omitempty"`

	Link        any `json:"link,omitempty"`
	Tunnel      any `json:"tunnel,omitempty"`
	Raw         any `json:"raw,omitempty"`
	Application any `json:"application,omitempty"`
	Map         any `json:"map,omitempty"`
}

type UpdateResp struct {
	OperationResponses []ResponseObject `json:"operationResponses"`
}

type ResponseObject struct {
	EntityType string `json:"entityType"`
	Id         string `json:"id"`
	Alias      string `json:"alias"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorMsg   string `json:"errorMessage,omitempty"`
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isUpdateOperationFailure(status string) bool {
	// FM can return a 2xx transport status while rejecting a specific operation
	// in operationResponses via status semantics.
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "failed", "failure", "error", "rejected", "invalid", "unhealthy":
		return true
	default:
		return false
	}
}

func extractBackendMessage(raw []byte) string {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}

	extractString := func(v any) string {
		s, _ := v.(string)
		return strings.TrimSpace(s)
	}

	if msg := firstNonEmpty(
		extractString(body["message"]),
		extractString(body["errorMessage"]),
		extractString(body["error"]),
		extractString(body["reason"]),
	); msg != "" {
		return msg
	}

	if errs, ok := body["errors"].([]any); ok && len(errs) > 0 {
		if first, ok := errs[0].(map[string]any); ok {
			if msg := firstNonEmpty(
				extractString(first["message"]),
				extractString(first["errorMessage"]),
				extractString(first["error"]),
				extractString(first["reason"]),
			); msg != "" {
				return msg
			}
		}
	}

	return ""
}

func parseUpdateMonSessResponse(respData []byte) (string, error) {
	var fmResp UpdateResp
	if err := json.Unmarshal(respData, &fmResp); err != nil {
		return "", fmt.Errorf(
			"Unable to decode update response: %s , err: %w",
			string(respData),
			err,
		)
	}

	if len(fmResp.OperationResponses) == 0 {
		backendMsg := extractBackendMessage(respData)
		if backendMsg != "" {
			return "", fmt.Errorf(
				"update response has no OperationResponses: %s. backend message: %s",
				string(respData),
				backendMsg,
			)
		}
		return "", fmt.Errorf(
			"update response has no OperationResponses: %s",
			string(respData),
		)
	}

	for _, op := range fmResp.OperationResponses {
		if !isUpdateOperationFailure(op.Status) {
			continue
		}

		detail := firstNonEmpty(op.ErrorMsg, op.Message, op.Error, extractBackendMessage(respData))
		if detail == "" {
			detail = string(respData)
		}
		return "", fmt.Errorf(
			"monitoring session update operation failed (entityType=%q alias=%q id=%q status=%q): %s",
			op.EntityType,
			op.Alias,
			op.Id,
			op.Status,
			detail,
		)
	}

	return fmResp.OperationResponses[0].Id, nil
}

// UpdateMonSess posts an update request for a Monitoring Session (create/change/delete).
func UpdateMonSess(
	ctx context.Context,
	req *UpdateReq,
	monSessId string,
	fmClient *fmclient.FmClient,
) (string, error) {

	rawID, err := UUIDFromTypedID(monSessId)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("Unable to encode into Json: %v, error: %w", req, err)
	}

Loop:
	for {
		respData, err := fmClient.DoRequest(
			ctx,
			"POST",
			fmt.Sprintf("api/v1.3/cloud/monitoringSessions/%s/update", rawID),
			map[string]string{"deploymentMode": "AUTO"},
			nil,
			bytes.NewBuffer(jsonData),
			"application/json",
		)
		if err != nil {
			var fmErr *fmclient.FMErrors
			if errors.As(err, &fmErr) {
				if fmErr.ErrorCode() == fmclient.TooManyRequests {
					timer := time.NewTimer(30 * time.Second)
					select {
					case <-timer.C:
						continue
					case <-ctx.Done():
						break Loop
					}
				}
			}
			return "", err
		}
		return parseUpdateMonSessResponse(respData)
	}
	return "", fmt.Errorf("Monitoring Session Update for %s timed out", monSessId)
}
