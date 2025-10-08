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
	host        string
	token       string
	proxy       string
	glcpCloudCA string // Base64 encoded CA certificate for on-premise DSCC
}

// Returns an instance of the api_client struct
func NewAPIClient(host, token, proxy, glcpCloudCA string) *api_client {
	return &api_client{
		host:        host,
		token:       token,
		proxy:       proxy,
		glcpCloudCA: glcpCloudCA,
	}
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

	// Configure TLS transport with custom CA support
	transport, err := a.configureTLSTransport()
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

	// Configure TLS transport with custom CA support
	transport, err := a.configureTLSTransport()
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

// configureTLSTransport configures HTTP transport with custom CA certificate if provided
func (a *api_client) configureTLSTransport() (*http.Transport, error) {
	// Create TLS configuration
	tlsConfig := &tls.Config{}

	// Configure custom CA certificate for on-premise DSCC if provided
	if len(a.glcpCloudCA) > 0 {
		// Decode base64 encoded CA certificate
		caCertData, err := base64.StdEncoding.DecodeString(a.glcpCloudCA)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 CA certificate: %v", err)
		}

		// Create certificate pool and add custom CA
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertData) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
	} else {
		// For corporate environments without custom CA, skip certificate verification
		tlsConfig.InsecureSkipVerify = true
	}

	// Create transport with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Configure proxy if provided
	if len(a.proxy) != 0 {
		proxy, err := url.Parse(a.proxy)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxy)
	}

	return transport, nil
}
