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

	// Valid base64-encoded self-signed CA certificate for testing
	validCACert := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lVRk1wN01rN3NnZWJVQW52ZFFvejczaC9OM1NNd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0VqRVFNQTRHQTFVRUF3d0hkR1Z6ZEMxallUQWVGdzB5TmpBME1Ea3dOakEyTXpGYUZ3MHlOekEwTURrdwpOakEyTXpGYU1CSXhFREFPQmdOVkJBTU1CM1JsYzNRdFkyRXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCCkR3QXdnZ0VLQW9JQkFRQzk4SXVpOE80NUFiQmFVaHkyWjdTelRKK1lBTGxJMFRySUg5KzkvNFpDZXpydXQ1bkEKZEE0RUJ3Vml1aWpWdXphQXFLM1QvN2Y1R1c2MUt5NVZLOUdGTjhORGNmeHlqQjVHNjR1WUQrSGNGSTJCRjg0SQpLaVBBVVNuaUdYN2NRUnUyZndUd20rWU9xRU5TaGl4aUhwVzkvQkY1Ry84cU5vVjJLUnh6djMvaWtySVp1ZUovCkVhOVZMVlVvMG1NVHRyTkZlcFVaVXNoZnl3OVpQZW5MRUdvaGNRejd2WHJ5VThaNVp3eDBDODhBa3hsYmx1aFQKS3g4RmZ0NzNVVkpTSmZjSXRUME0xbUNrZEVNWDNhMDh1ZkdTU3lWWDZZRTA0RlFSMC96ZEVUazh0d3RkcGpjbgpWZ3lNN2JtY2ZWYmlMazRCSCtvZmRRbDJEVHlDUzVINkVBb1ZBZ01CQUFHalV6QlJNQjBHQTFVZERnUVdCQlRXCjRVSW54cUJMUEcwNnpCNkNMaGxNb0lXVDVEQWZCZ05WSFNNRUdEQVdnQlRXNFVJbnhxQkxQRzA2ekI2Q0xobE0Kb0lXVDVEQVBCZ05WSFJNQkFmOEVCVEFEQVFIL01BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQzBBczkxSnVwbAp2UUVDTkwxanJKcUVuc0lXc3BUajlPL2MzVUo3S0hZOHpGS2F6V21OL1hsV0RoZzRyVlg4UVFETk5rbTBxWGdRCjNSekN4RGZDSCtDQzAyemRoVkVYWnErMDNnWGkwSzZIaTJPOWNza0hOVFgyZ3VDZHF1dEpNY1l1eDRTSElFTzMKUHlBb3pyYmFENHJpK1NiNHkva0xNUUJ5aXdMcERWeG1JNm1Ubm9Sb2FOTEwvY3pvNnQ3ZEtGRFl2c0FJQVZuSwpSL243Y1ZFSTRRRGNyR1pPVUt5Y1lOcjlxazRtRXBzdUNCM29ZT0pPMmNKelo0Vk1qSjVOZnROdi9aVlphTmpKCmVaZ2xNWSswamc3NW9hLzJsaHJQTmRzaGVNaWo0RmxDUHpiNGlmNEVrZDFIUHIrdWlISmRUNnFzSTRsdEYvelAKNTUzZ2I5bVUvVGpUCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

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
