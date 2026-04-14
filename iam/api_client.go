// © Copyright 2024 Hewlett Packard Enterprise Development LP

package iam

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	sdk "hpe-cosi-osp/alletraMPX10000_sdk"
	tsdk "hpe-cosi-osp/glcp_tasks_sdk"
	"net/http"
	"net/url"
	"strings"
)

// Defines attributes to create an API client for the DSCC
type api_client struct {
	host          string
	token         string
	proxy         string
	onPremCloudCA string // Base64 encoded CA certificate for on-premise DSCC
}

// Returns an instance of the api_client struct
func NewAPIClient(host, token, proxy, onPremCloudCA string) *api_client {
	return &api_client{
		host:          host,
		token:         token,
		proxy:         proxy,
		onPremCloudCA: onPremCloudCA,
	}
}

// buildHTTPTransport creates an http.Transport configured with proxy and/or custom CA certificate.
// This avoids duplicating proxy and CA configuration logic across multiple API clients.
func buildHTTPTransport(proxy, onPremCloudCA string) (*http.Transport, error) {
	transport := &http.Transport{}

	// Configure proxy if provided
	if len(proxy) != 0 {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy: %v", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	// Configure custom CA certificate if provided
	if len(onPremCloudCA) > 0 {
		caCertData, err := base64.StdEncoding.DecodeString(onPremCloudCA)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 CA certificate: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertData) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		transport.TLSClientConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	}

	return transport, nil
}

// Creates an API client for communicating with the DSCC
func (a *api_client) GetAPIClient() (*sdk.APIClient, error) {
	configuration := sdk.NewConfiguration()
	configuration.Host = a.host
	if _, host, prefix := strings.Cut(a.host, "://"); prefix {
		configuration.Host = host
	}
	configuration.DefaultHeader = constructBearerToken(a.token)
	uri := a.host
	if !strings.HasPrefix(uri, "https://") {
		uri = "https://" + uri
	}
	sConfig := sdk.ServerConfiguration{
		URL: uri,
	}
	serverConfigs := []sdk.ServerConfiguration{sConfig}
	configuration.Servers = serverConfigs
	configuration.OperationServers = map[string]sdk.ServerConfigurations{
		"ObjectstoreIdentitiesAPI.HPE_Alletra_Storage_MP_X10000": serverConfigs}
	api := sdk.NewAPIClient(configuration)

	transport, err := buildHTTPTransport(a.proxy, a.onPremCloudCA)
	if err != nil {
		return nil, err
	}
	api.GetConfig().HTTPClient.Transport = transport

	return api, nil
}

// TODO: Would be merged with above method, once we have a single SDK
// Creates a task API client for communicating with the task module of the DSCC
func (a *api_client) GetTaskAPIClient() (*tsdk.APIClient, error) {
	configuration := tsdk.NewConfiguration()
	configuration.Host = a.host
	if _, host, prefix := strings.Cut(a.host, "://"); prefix {
		configuration.Host = host
	}
	configuration.DefaultHeader = constructBearerToken(a.token)
	uri := a.host
	if !strings.HasPrefix(uri, "https://") {
		uri = "https://" + uri
	}
	sConfig := tsdk.ServerConfiguration{
		URL: uri,
	}
	serverConfigs := []tsdk.ServerConfiguration{sConfig}
	configuration.Servers = serverConfigs
	//TODO: Need to check, if this can be removed
	configuration.OperationServers = map[string]tsdk.ServerConfigurations{
		"TasksAPIService.GetTaskStatus": serverConfigs}
	api := tsdk.NewAPIClient(configuration)

	transport, err := buildHTTPTransport(a.proxy, a.onPremCloudCA)
	if err != nil {
		return nil, err
	}
	api.GetConfig().HTTPClient.Transport = transport

	return api, nil
}

// Constructs an header for the passing the oauth2 bearer token
func constructBearerToken(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}
