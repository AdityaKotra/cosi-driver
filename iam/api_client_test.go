// © Copyright 2024 Hewlett Packard Enterprise Development LP

package iam

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

var host = "us1.xxxx.xxxxx.hpe.com"
var token = "bearerdummyoxyzxxzzz12xxxx341111zzzzyyyyyyQQQQQHHHHH"
var proxy = "http://dummy_proxy:8080"

// generateTestCACert generates a self-signed CA certificate at runtime for unit testing.
// Visible for UT alone — avoids hardcoding certificate data.
func generateTestCACert(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return base64.StdEncoding.EncodeToString(certPEM)
}

func Test_GetAPIClient(t *testing.T) {
	validCACert := generateTestCACert(t)

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
	validCACert := generateTestCACert(t)

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
	validCACert := generateTestCACert(t)

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
