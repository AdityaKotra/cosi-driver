// © Copyright 2024 Hewlett Packard Enterprise Development LP

// Package utils
package utils

import (
	"context"
	stdlog "log"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

func TestGetLoggerFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	log := stdr.New(stdlog.New(os.Stdout, "", stdlog.LstdFlags))
	tests := []struct {
		name   string
		args   args
		ctxVal *logr.Logger
	}{
		{
			name:   "Logger in context",
			args:   args{ctx: context.WithValue(context.Background(), LoggerKey, &log)},
			ctxVal: &log,
		},
		{
			name:   "Logger not in context",
			args:   args{ctx: context.Background()},
			ctxVal: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLoggerFromContext(tt.args.ctx)
			if tt.ctxVal != nil {
				if got != tt.ctxVal {
					t.Errorf("GetLoggerFromContext() = %v, want %v", got, tt.ctxVal)
				}
			} else {
				if got == nil {
					t.Errorf("GetLoggerFromContext() = nil, want non-nil logger")
				}
			}
		})
	}
}
