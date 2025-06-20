// © Copyright 2024 Hewlett Packard Enterprise Development LP

// Package utils
package utils

import (
	"context"
	stdlog "log"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const (
	// Bucket Class parameters' keys
	bucketTagsKey              string = "bucketTags"
	CosiUserSecretNameKey      string = "cosiUserSecretName"
	CosiUserSecretNamespaceKey string = "cosiUserSecretNamespace"

	// Provisioner pod's env variable for the COSI driver container
	PodNamespaceEnv string = "POD_NAMESPACE"
)

var (
// Removed unused variables httpTimeout, enablePathStyle, disableSSL
)

// LoggerKeyType is a custom type for context keys to avoid collisions
// See: https://staticcheck.io/docs/checks/#SA1029
type LoggerKeyType string

// Logger key to extract logger from context
const LoggerKey LoggerKeyType = "loggerKey"

func GetLoggerFromContext(ctx context.Context) *logr.Logger {
	logger := ctx.Value(LoggerKey)

	// If the context has no logger, create a new logger and return
	if logger == nil {
		log := stdr.New(stdlog.New(os.Stdout, "", stdlog.LstdFlags))
		return &log
	}

	return logger.(*logr.Logger)
}
