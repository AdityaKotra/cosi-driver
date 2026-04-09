// © Copyright 2024 Hewlett Packard Enterprise Development LP

package iam

import (
	"testing"
)

var host = "us1.xxxx.xxxxx.hpe.com"
var token = "bearerdummyoxyzxxzzz12xxxx341111zzzzyyyyyyQQQQQHHHHH"
var proxy = "http://dummy_proxy:8080"

// Valid base64-encoded self-signed CA certificate for testing
var validCACert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lVRk1wN01rN3NnZWJVQW52ZFFvejczaC9OM1NNd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0VqRVFNQTRHQTFVRUF3d0hkR1Z6ZEMxallUQWVGdzB5TmpBME1Ea3dOakEyTXpGYUZ3MHlOekEwTURrdwpOakEyTXpGYU1CSXhFREFPQmdOVkJBTU1CM1JsYzNRdFkyRXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCCkR3QXdnZ0VLQW9JQkFRQzk4SXVpOE80NUFiQmFVaHkyWjdTelRKK1lBTGxJMFRySUg5KzkvNFpDZXpydXQ1bkEKZEE0RUJ3Vml1aWpWdXphQXFLM1QvN2Y1R1c2MUt5NVZLOUdGTjhORGNmeHlqQjVHNjR1WUQrSGNGSTJCRjg0SQpLaVBBVVNuaUdYN2NRUnUyZndUd20rWU9xRU5TaGl4aUhwVzkvQkY1Ry84cU5vVjJLUnh6djMvaWtySVp1ZUovCkVhOVZMVlVvMG1NVHRyTkZlcFVaVXNoZnl3OVpQZW5MRUdvaGNRejd2WHJ5VThaNVp3eDBDODhBa3hsYmx1aFQKS3g4RmZ0NzNVVkpTSmZjSXRUME0xbUNrZEVNWDNhMDh1ZkdTU3lWWDZZRTA0RlFSMC96ZEVUazh0d3RkcGpjbgpWZ3lNN2JtY2ZWYmlMazRCSCtvZmRRbDJEVHlDUzVINkVBb1ZBZ01CQUFHalV6QlJNQjBHQTFVZERnUVdCQlRXCjRVSW54cUJMUEcwNnpCNkNMaGxNb0lXVDVEQWZCZ05WSFNNRUdEQVdnQlRXNFVJbnhxQkxQRzA2ekI2Q0xobE0Kb0lXVDVEQVBCZ05WSFJNQkFmOEVCVEFEQVFIL01BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQzBBczkxSnVwbAp2UUVDTkwxanJKcUVuc0lXc3BUajlPL2MzVUo3S0hZOHpGS2F6V21OL1hsV0RoZzRyVlg4UVFETk5rbTBxWGdRCjNSekN4RGZDSCtDQzAyemRoVkVYWnErMDNnWGkwSzZIaTJPOWNza0hOVFgyZ3VDZHF1dEpNY1l1eDRTSElFTzMKUHlBb3pyYmFENHJpK1NiNHkva0xNUUJ5aXdMcERWeG1JNm1Ubm9Sb2FOTEwvY3pvNnQ3ZEtGRFl2c0FJQVZuSwpSL243Y1ZFSTRRRGNyR1pPVUt5Y1lOcjlxazRtRXBzdUNCM29ZT0pPMmNKelo0Vk1qSjVOZnROdi9aVlphTmpKCmVaZ2xNWSswamc3NW9hLzJsaHJQTmRzaGVNaWo0RmxDUHpiNGlmNEVrZDFIUHIrdWlISmRUNnFzSTRsdEYvelAKNTUzZ2I5bVUvVGpUCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

func Test_GetAPIClient(t *testing.T) {
	t.Run("GetAPIClient successful", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, proxy, "")
		client, err := apiClient.GetAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error")
		}
	})
	t.Run("GetAPIClient with proxy & password", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, proxy, "")
		client, err := apiClient.GetAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error")
		}
	})
	t.Run("GetAPIClient with valid CA certificate", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, "", validCACert)
		client, err := apiClient.GetAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error when providing valid CA certificate")
		}
	})
	t.Run("GetAPIClient with CA certificate and proxy", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, proxy, validCACert)
		client, err := apiClient.GetAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error when providing CA certificate and proxy")
		}
	})
	t.Run("GetAPIClient with invalid base64 CA certificate", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, "", "not-valid-base64!!!")
		client, err := apiClient.GetAPIClient()
		if err == nil {
			t.Errorf("FAILED: expected error for invalid base64 CA certificate")
		}
		if client != nil {
			t.Errorf("FAILED: expected nil client for invalid base64 CA certificate")
		}
	})
	t.Run("GetAPIClient with invalid PEM CA certificate", func(t *testing.T) {
		// Valid base64, but not a valid PEM certificate
		apiClient := NewAPIClient(host, token, "", "dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==")
		client, err := apiClient.GetAPIClient()
		if err == nil {
			t.Errorf("FAILED: expected error for invalid PEM CA certificate")
		}
		if client != nil {
			t.Errorf("FAILED: expected nil client for invalid PEM CA certificate")
		}
	})
}

func Test_GetTaskAPIClient(t *testing.T) {
	t.Run("GetTaskAPIClient successful", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, proxy, "")
		client, err := apiClient.GetTaskAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error")
		}
	})
	t.Run("GetTaskAPIClient with proxy & password", func(t *testing.T) {
		proxy := "http://dummy_user:dummy_password@dummy_proxy:8080"
		apiClient := NewAPIClient(host, token, proxy, "")
		client, err := apiClient.GetTaskAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error")
		}
	})
	t.Run("GetTaskAPIClient with valid CA certificate", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, "", validCACert)
		client, err := apiClient.GetTaskAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error when providing valid CA certificate")
		}
	})
	t.Run("GetTaskAPIClient with CA certificate and proxy", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, proxy, validCACert)
		client, err := apiClient.GetTaskAPIClient()
		if err != nil || client == nil {
			t.Errorf("FAILED: unexpected error when providing CA certificate and proxy")
		}
	})
	t.Run("GetTaskAPIClient with invalid base64 CA certificate", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, "", "not-valid-base64!!!")
		client, err := apiClient.GetTaskAPIClient()
		if err == nil {
			t.Errorf("FAILED: expected error for invalid base64 CA certificate")
		}
		if client != nil {
			t.Errorf("FAILED: expected nil client for invalid base64 CA certificate")
		}
	})
	t.Run("GetTaskAPIClient with invalid PEM CA certificate", func(t *testing.T) {
		apiClient := NewAPIClient(host, token, "", "dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==")
		client, err := apiClient.GetTaskAPIClient()
		if err == nil {
			t.Errorf("FAILED: expected error for invalid PEM CA certificate")
		}
		if client != nil {
			t.Errorf("FAILED: expected nil client for invalid PEM CA certificate")
		}
	})
}

func Test_buildHTTPTransport(t *testing.T) {
	t.Run("buildHTTPTransport with no proxy and no CA", func(t *testing.T) {
		transport, err := buildHTTPTransport("", "")
		if err != nil {
			t.Errorf("FAILED: unexpected error: %v", err)
		}
		if transport == nil {
			t.Errorf("FAILED: expected non-nil transport")
		}
		if transport.Proxy != nil {
			t.Errorf("FAILED: expected nil proxy")
		}
		if transport.TLSClientConfig != nil {
			t.Errorf("FAILED: expected nil TLS config")
		}
	})
	t.Run("buildHTTPTransport with proxy only", func(t *testing.T) {
		transport, err := buildHTTPTransport(proxy, "")
		if err != nil {
			t.Errorf("FAILED: unexpected error: %v", err)
		}
		if transport == nil {
			t.Errorf("FAILED: expected non-nil transport")
		}
		if transport.Proxy == nil {
			t.Errorf("FAILED: expected proxy to be configured")
		}
		if transport.TLSClientConfig != nil {
			t.Errorf("FAILED: expected nil TLS config when no CA provided")
		}
	})
	t.Run("buildHTTPTransport with CA only", func(t *testing.T) {
		transport, err := buildHTTPTransport("", validCACert)
		if err != nil {
			t.Errorf("FAILED: unexpected error: %v", err)
		}
		if transport == nil {
			t.Errorf("FAILED: expected non-nil transport")
		}
		if transport.TLSClientConfig == nil {
			t.Errorf("FAILED: expected TLS config to be set")
		}
		if transport.TLSClientConfig.RootCAs == nil {
			t.Errorf("FAILED: expected RootCAs to be configured")
		}
	})
	t.Run("buildHTTPTransport with proxy and CA", func(t *testing.T) {
		transport, err := buildHTTPTransport(proxy, validCACert)
		if err != nil {
			t.Errorf("FAILED: unexpected error: %v", err)
		}
		if transport == nil {
			t.Errorf("FAILED: expected non-nil transport")
		}
		if transport.Proxy == nil {
			t.Errorf("FAILED: expected proxy to be configured")
		}
		if transport.TLSClientConfig == nil {
			t.Errorf("FAILED: expected TLS config to be set")
		}
		if transport.TLSClientConfig.RootCAs == nil {
			t.Errorf("FAILED: expected RootCAs to be configured")
		}
	})
	t.Run("buildHTTPTransport with invalid base64 CA", func(t *testing.T) {
		transport, err := buildHTTPTransport("", "not-valid-base64!!!")
		if err == nil {
			t.Errorf("FAILED: expected error for invalid base64")
		}
		if transport != nil {
			t.Errorf("FAILED: expected nil transport for invalid base64")
		}
	})
	t.Run("buildHTTPTransport with invalid PEM CA", func(t *testing.T) {
		transport, err := buildHTTPTransport("", "dGhpcyBpcyBub3QgYSB2YWxpZCBjZXJ0aWZpY2F0ZQ==")
		if err == nil {
			t.Errorf("FAILED: expected error for invalid PEM")
		}
		if transport != nil {
			t.Errorf("FAILED: expected nil transport for invalid PEM")
		}
	})
	t.Run("buildHTTPTransport with invalid proxy URL", func(t *testing.T) {
		transport, err := buildHTTPTransport("://invalid-proxy", "")
		if err == nil {
			t.Errorf("FAILED: expected error for invalid proxy URL")
		}
		if transport != nil {
			t.Errorf("FAILED: expected nil transport for invalid proxy URL")
		}
	})
}
