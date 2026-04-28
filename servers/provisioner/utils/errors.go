// © Copyright Hewlett Packard Enterprise Development LP

// Package utils
package utils

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClassifyError inspects the error message and returns a gRPC status error with
// the appropriate error code. This centralises error-code mapping so that all
// Driver* methods (and consequently Kubernetes events) carry a meaningful code.
//
// Mapping rules (evaluated in order):
//
//	"already exists"          → codes.AlreadyExists
//	"permission denied",
//	"forbidden", "403"        → codes.PermissionDenied
//	"invalid", "required",
//	"unsupported"             → codes.InvalidArgument
//	"not found", "404"        → codes.NotFound
//	everything else           → codes.Internal
func ToGRPCStatusError(msg string, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	combined := strings.ToLower(msg + " " + errMsg)

	var code codes.Code
	switch {
	case strings.Contains(combined, "already exists"):
		code = codes.AlreadyExists
	case strings.Contains(combined, "permission denied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "403"):
		code = codes.PermissionDenied
	case strings.Contains(combined, "invalid") ||
		strings.Contains(combined, "required") ||
		strings.Contains(combined, "unsupported"):
		code = codes.InvalidArgument
	case strings.Contains(combined, "not found") ||
		strings.Contains(combined, "404"):
		code = codes.NotFound
	default:
		code = codes.Internal
	}

	// Use the human-readable msg as the status message; append the
	// underlying error when available.
	detail := msg
	if err != nil {
		detail = msg + ": " + err.Error()
	}
	return status.Error(code, detail)
}
