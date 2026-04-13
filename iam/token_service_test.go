// © Copyright 2024 Hewlett Packard Enterprise Development LP
package iam

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"

	stdlog "log"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/stdr"
	"gotest.tools/v3/assert"
)

const testExpToken = "bearerdummyoxyzxxzzz12xxxx341111zzzzyyyyyyQQQQQHHHHHbF9fc2NGOGpT"

func Test_GetAccessToken(t *testing.T) {
	glcpUrl := "sso.cloud.example.com"
	glcpUser := "xxxxxxx-zzz-123-3456"
	glcpUserSecret := "zzzzzrandomxxxxxxzzzzz"
	proxy := "http://dummy_proxy:8080"
	log := stdr.New(stdlog.New(os.Stdout, "", stdlog.LstdFlags))

	ts := NewTokenService(glcpUrl, glcpUser, glcpUserSecret, proxy, "", &log)

	t.Run("GetAccessToken successful", func(t *testing.T) {
		expToken := testExpToken
		oauth2 := oauth2_token{
			AccessToken: expToken,
			TokenType:   "Bearer",
			ExpiresIn:   7199,
		}
		data, _ := json.Marshal(oauth2)
		reader := bytes.NewReader(data)
		response := http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(reader),
		}
		patch := gomonkey.ApplyMethodReturn(ts, "PostRequest", &response, nil)
		defer patch.Reset()

		token, err := ts.GetAccessToken()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, token, expToken)
	})

	t.Run("GetAccessToken failed", func(t *testing.T) {
		patch := gomonkey.ApplyMethodReturn(ts, "PostRequest", nil, errors.New("error while generating token"))
		defer patch.Reset()
		token, err := ts.GetAccessToken()
		if err == nil {
			t.Errorf("FAILED: expected errornot found")
		}
		assert.Equal(t, len(token), 0)
	})

	t.Run("GetAccessToken with non-OK status code", func(t *testing.T) {
		response := http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}
		patch := gomonkey.ApplyMethodReturn(ts, "PostRequest", &response, nil)
		defer patch.Reset()

		token, err := ts.GetAccessToken()
		if err == nil {
			t.Errorf("FAILED: expected error for non-OK status code")
		}
		assert.Equal(t, len(token), 0)
	})

	t.Run("GetAccessToken with empty token response", func(t *testing.T) {
		oauth2 := oauth2_token{
			AccessToken: "",
			TokenType:   "Bearer",
			ExpiresIn:   7199,
		}
		data, _ := json.Marshal(oauth2)
		response := http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(data)),
		}
		patch := gomonkey.ApplyMethodReturn(ts, "PostRequest", &response, nil)
		defer patch.Reset()

		token, err := ts.GetAccessToken()
		if err == nil {
			t.Errorf("FAILED: expected error for empty token")
		}
		assert.Equal(t, len(token), 0)
	})
}

func Test_GetAccessToken_WithCA(t *testing.T) {
	glcpUrl := "sso.on-prem.example.com"
	glcpUser := "xxxxxxx-zzz-123-3456"
	glcpUserSecret := "zzzzzrandomxxxxxxzzzzz"
	log := stdr.New(stdlog.New(os.Stdout, "", stdlog.LstdFlags))

	// Generate CA certificate at runtime to avoid hardcoded certificate patterns in source code
	validCACert := generateTestCACert(t)

	t.Run("GetAccessToken with CA certificate successful", func(t *testing.T) {
		ts := NewTokenService(glcpUrl, glcpUser, glcpUserSecret, "", validCACert, &log)
		expToken := testExpToken
		oauth2 := oauth2_token{
			AccessToken: expToken,
			TokenType:   "Bearer",
			ExpiresIn:   7199,
		}
		data, _ := json.Marshal(oauth2)
		response := http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(data)),
		}
		patch := gomonkey.ApplyMethodReturn(ts, "PostRequest", &response, nil)
		defer patch.Reset()

		token, err := ts.GetAccessToken()
		if err != nil {
			t.Errorf("FAILED: unexpected error: %v", err)
		}
		assert.Equal(t, token, expToken)
	})

	t.Run("GetAccessToken with CA certificate and proxy", func(t *testing.T) {
		ts := NewTokenService(glcpUrl, glcpUser, glcpUserSecret, "http://dummy_proxy:8080", validCACert, &log)
		expToken := testExpToken
		oauth2 := oauth2_token{
			AccessToken: expToken,
			TokenType:   "Bearer",
			ExpiresIn:   7199,
		}
		data, _ := json.Marshal(oauth2)
		response := http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(data)),
		}
		patch := gomonkey.ApplyMethodReturn(ts, "PostRequest", &response, nil)
		defer patch.Reset()

		token, err := ts.GetAccessToken()
		if err != nil {
			t.Errorf("FAILED: unexpected error: %v", err)
		}
		assert.Equal(t, token, expToken)
	})

	t.Run("GetAccessToken with invalid base64 CA certificate", func(t *testing.T) {
		ts := NewTokenService(glcpUrl, glcpUser, glcpUserSecret, "", "not-valid-base64!!!", &log)
		// PostRequest will fail because buildHTTPTransport will fail with invalid base64
		token, err := ts.GetAccessToken()
		if err == nil {
			t.Errorf("FAILED: expected error for invalid base64 CA certificate")
		}
		assert.Equal(t, len(token), 0)
	})

	t.Run("GetAccessToken with invalid PEM CA certificate", func(t *testing.T) {
		// Valid base64 but not a valid PEM certificate
		ts := NewTokenService(glcpUrl, glcpUser, glcpUserSecret, "", "dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==", &log)
		token, err := ts.GetAccessToken()
		if err == nil {
			t.Errorf("FAILED: expected error for invalid PEM CA certificate")
		}
		assert.Equal(t, len(token), 0)
	})
}
