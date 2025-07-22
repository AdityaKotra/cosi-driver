// © Copyright 2024 Hewlett Packard Enterprise Development LP

// Package utils
package utils

import (
	"encoding/xml"
)

const (
	ACCESS_POLICY_PREFIX = "acp_"
	USER_PREFIX          = "user_"
	//KEYS in glcp secret
	GLCP_USER_CLIENTID   = "glcpUserClientId"
	GLCP_USER_SECRET_KEY = "glcpUserSecretKey"
	GLCP_COMMON_CLOUD    = "GLCP_COMMON_CLOUD"
	DSCC_ZONE            = "dsccZone"
	ALLETRA_MP_X10K_SNO  = "clusterSerialNumber"
	ENDPOINT             = "endpoint"
	RETRY_ATTEMPT        = 3
	PROXY                = "PROXY"
)

// Defines credentials to access DSCC through GLCP API user
type IAMCredentials struct {
	GLCPUser          string
	GLCPUserSecretKey string
	GLCPCommonCloud   string
	DSCCZone          string
	SystemId          string
	Endpoint          string
	Proxy             string
}

// Defines BucketRequest structure with options (Versioning, Locking, Compression)
type BucketRequest struct {
	Compression     bool   `json:"Compression"`
	Versioning      string `json:"Versioning,omitempty"`
	Locking         string `json:"Locking,omitempty"`
	RetentionMode   string `json:"RetentionMode,omitempty"`
	ObjectLockDays  int    `json:"ObjectLockDays,omitempty"`
	ObjectLockYears int    `json:"ObjectLockYears,omitempty"`
}

// Add new structs for S3 bucket creation
type SpaceQuota struct {
	QuotaType     string `json:"QuotaType"`
	QuotaLimitMiB int    `json:"QuotaLimitMiB"`
}

type CreateBucketRequest struct {
	LocationConstraint string     `json:"LocationConstraint"`
	Compression        bool       `json:"Compression"`
	BucketPolicy       string     `json:"BucketPolicy"`
	Versioning         string     `json:"Versioning,omitempty"`
	ObjectLockEnabled  string     `json:"ObjectLockEnabled,omitempty"`
	ObjectLockMode     string     `json:"ObjectLockMode,omitempty"`
	ObjectLockDays     int        `json:"ObjectLockDays,omitempty"`
	ObjectLockYears    int        `json:"ObjectLockYears,omitempty"`
	SpaceQuota         SpaceQuota `json:"SpaceQuota"`
}

// ObjectLockConfiguration represents the XML structure for object lock configuration
type ObjectLockConfiguration struct {
	XMLName           xml.Name        `xml:"ObjectLockConfiguration"`
	Xmlns             string          `xml:"xmlns,attr"`
	ObjectLockEnabled string          `xml:"ObjectLockEnabled"`
	Rule              *ObjectLockRule `xml:"Rule,omitempty"`
}

type ObjectLockRule struct {
	DefaultRetention ObjectLockDefaultRetention `xml:"DefaultRetention"`
}

type ObjectLockDefaultRetention struct {
	Mode  string `xml:"Mode,omitempty"`
	Days  int    `xml:"Days,omitempty"`
	Years int    `xml:"Years,omitempty"`
}

// Feature represents a feature toggle.
type Feature string

const (
	FeatureEnabled  Feature = "Enabled"
	FeatureDisabled Feature = "Disabled"
)
