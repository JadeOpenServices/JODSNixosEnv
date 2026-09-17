package broker

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

type Action string

const (
	ActionStatus            Action = "status"
	ActionAudit             Action = "audit"
	ActionAllowOnce         Action = "allow-once"
	ActionTrustPermanent    Action = "trust-permanent"
	ActionForget            Action = "forget"
	ActionAcceptReplacement Action = "accept-replacement"
	ActionEnrollInternal    Action = "enroll-internal"
)

type Request struct {
	Action    Action `json:"action"`
	RuntimeID string `json:"runtimeId,omitempty"`
	TrustedID string `json:"trustedId,omitempty"`
	Role      string `json:"role,omitempty"`

	// Portable is explicit for permanent external trust.
	// nil means the caller did not make a portability decision.
	Portable *bool `json:"portable,omitempty"`
}

type Status struct {
	StatePresent    bool   `json:"statePresent"`
	Revision        uint64 `json:"revision,omitempty"`
	TrustedDevices  int    `json:"trustedDevices"`
	ExpectedDevices int    `json:"expectedDevices"`
	ObservedDevices int    `json:"observedDevices"`
}

type Response struct {
	OK       bool                  `json:"ok"`
	Message  string                `json:"message,omitempty"`
	Error    string                `json:"error,omitempty"`
	Revision uint64                `json:"revision,omitempty"`
	Status   *Status               `json:"status,omitempty"`
	Audit    *usbtrust.AuditResult `json:"audit,omitempty"`
}

func (a Action) Valid() bool {
	switch a {
	case ActionStatus,
		ActionAudit,
		ActionAllowOnce,
		ActionTrustPermanent,
		ActionForget,
		ActionAcceptReplacement,
		ActionEnrollInternal:
		return true
	default:
		return false
	}
}

func (a Action) Mutation() bool {
	switch a {
	case ActionAllowOnce,
		ActionTrustPermanent,
		ActionForget,
		ActionAcceptReplacement,
		ActionEnrollInternal:
		return true
	default:
		return false
	}
}

func ValidateRequest(request Request) error {
	if !request.Action.Valid() {
		return fmt.Errorf(
			"unsupported USB trust action %q",
			request.Action,
		)
	}

	switch request.Action {
	case ActionStatus, ActionAudit:
		return noMutationArguments(request)

	case ActionAllowOnce:
		if strings.TrimSpace(request.RuntimeID) == "" {
			return fmt.Errorf(
				"%s requires runtimeId",
				request.Action,
			)
		}

	case ActionTrustPermanent:
		if strings.TrimSpace(request.RuntimeID) == "" {
			return fmt.Errorf(
				"%s requires runtimeId",
				request.Action,
			)
		}

		if request.Portable == nil {
			return fmt.Errorf(
				"%s requires an explicit portable decision",
				request.Action,
			)
		}

	case ActionForget:
		if strings.TrimSpace(request.TrustedID) == "" {
			return fmt.Errorf(
				"%s requires trustedId",
				request.Action,
			)
		}

	case ActionAcceptReplacement:
		if strings.TrimSpace(request.RuntimeID) == "" ||
			strings.TrimSpace(request.TrustedID) == "" {
			return fmt.Errorf(
				"%s requires runtimeId and trustedId",
				request.Action,
			)
		}

	case ActionEnrollInternal:
		if strings.TrimSpace(request.RuntimeID) == "" ||
			strings.TrimSpace(request.Role) == "" {
			return fmt.Errorf(
				"%s requires runtimeId and role",
				request.Action,
			)
		}
	}

	return nil
}

func noMutationArguments(request Request) error {
	if request.RuntimeID != "" ||
		request.TrustedID != "" ||
		request.Role != "" ||
		request.Portable != nil {
		return fmt.Errorf(
			"%s does not accept mutation arguments",
			request.Action,
		)
	}

	return nil
}

func DecodeRequest(reader io.Reader) (Request, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	var request Request

	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf(
			"decode USB trust request: %w",
			err,
		)
	}

	var trailing any

	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Request{}, fmt.Errorf(
				"USB trust request contains trailing JSON",
			)
		}

		return Request{}, fmt.Errorf(
			"decode trailing USB trust request data: %w",
			err,
		)
	}

	if err := ValidateRequest(request); err != nil {
		return Request{}, err
	}

	return request, nil
}

func EncodeResponse(
	writer io.Writer,
	response Response,
) error {
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf(
			"encode USB trust response: %w",
			err,
		)
	}

	return nil
}
