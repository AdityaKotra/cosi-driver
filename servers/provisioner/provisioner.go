// © Copyright 2024 Hewlett Packard Enterprise Development LP

// Package provisioner
package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"hpe-cosi-osp/iam"
	"hpe-cosi-osp/servers/provisioner/utils"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	bucketclientset "sigs.k8s.io/container-object-storage-interface/client/clientset/versioned"
	cosi "sigs.k8s.io/container-object-storage-interface/proto"
)

const (
	FeatureEnabled  = "Enabled"
	FeatureDisabled = "Disabled"
)

// Server implements cosi.ProvisionerServer interface.
type Server struct {
	log             logr.Logger
	K8sClientset    *kubernetes.Clientset
	BucketClientset bucketclientset.Interface
	cosi.UnimplementedProvisionerServer
}

// Interface guards.
var _ cosi.ProvisionerServer = (*Server)(nil)

var (
	errEmptySecretParameters = errors.New("cosiUserSecretName and cosiUserSecretNamespace are required in the parameters of the bucket class")
	errEmptySecretData       = errors.New("accessKey, secretKey and endpoint data are required in the secret")
)

// New returns provisioner.Server with default values.
func New(logger logr.Logger) (*Server, error) {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	k8sClientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	bucketClientset, err := bucketclientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return &Server{
		log:             logger,
		K8sClientset:    k8sClientset,
		BucketClientset: bucketClientset,
	}, nil
}

// DriverCreateBucket call is made to create the bucket in the backend.
//
// NOTE: this call needs to be idempotent.
//  1. If a bucket that matches both name and parameters already exists, then OK (success) must be returned.
//  2. If a bucket by same name, but different parameters is provided, then the appropriate error code ALREADY_EXISTS must be returned.
func (s *Server) DriverCreateBucket(ctx context.Context, req *cosi.DriverCreateBucketRequest) (*cosi.DriverCreateBucketResponse, error) {
	s.log.Info("Received request to create bucket.", "bucketName", req.Name)
	ctx = context.WithValue(ctx, utils.LoggerKey, &s.log)

	// Get the bucket name from the request
	bucketName := req.Name

	// Get the parameters from the request
	param := req.Parameters

	// Create the S3 client
	s3c, err := createS3Client(ctx, param, s.K8sClientset)
	if err != nil {
		return nil, err
	}

	// --- Modular parameter parsing ---
	versioning := parseVersioning(param)
	compression := parseCompression(param)
	locking, retentionMode, objectLockDays, objectLockYears := parseObjectLock(param)
	tags := parseBucketTags(param)

	// Enforce: object locking only allowed if versioning is enabled
	if locking == FeatureEnabled && versioning != FeatureEnabled {
		return nil, status.Error(codes.InvalidArgument, "Object locking requires versioning to be enabled.")
	}

	bucketReq := BucketRequest{
		Compression:     compression,
		Versioning:      versioning,
		Locking:         locking,
		RetentionMode:   retentionMode,
		ObjectLockDays:  objectLockDays,
		ObjectLockYears: objectLockYears,
	}

	// Attempt bucket creation
	if err = s3c.CreateBucket(ctx, bucketName, bucketReq); err != nil {
		return nil, status.Error(codes.Internal, "failed to create bucket due to an internal error")
	}

	// If locking is enabled, set object lock configuration
	if locking == FeatureEnabled {
		err = s3c.SetObjectLockConfiguration(ctx, bucketName, locking, retentionMode, objectLockDays, objectLockYears)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to set object lock configuration: %v", err))
		}
	}

	// If tags are present, set bucket tags
	if len(tags) > 0 {
		err = s3c.PutBucketTagging(ctx, bucketName, tags)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to set bucket tags: %v", err))
		}
	}

	return &cosi.DriverCreateBucketResponse{
		BucketId: bucketName,
	}, nil
}

// DriverDeleteBucket call is made to delete the bucket in the backend.
//
// NOTE: this call needs to be idempotent.
// If the bucket has already been deleted, then no error should be returned.
func (s *Server) DriverDeleteBucket(ctx context.Context, req *cosi.DriverDeleteBucketRequest) (*cosi.DriverDeleteBucketResponse, error) {
	s.log.Info("Received request to delete bucket.", "bucketName", req.BucketId)
	ctx = context.WithValue(ctx, utils.LoggerKey, &s.log)

	// Get the bucket name from the request
	bucketName := req.BucketId

	// Get the bucket object
	bucket, err := s.BucketClientset.ObjectstorageV1alpha1().Buckets().Get(ctx, bucketName, metav1.GetOptions{})
	if err != nil {
		s.log.Error(err, "failed to get bucket object", "bucketName", bucketName)
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get bucket: %s", bucketName))
	}
	// Get the parameters from the spec of the bucket
	param := bucket.Spec.Parameters

	// Create the S3 client
	s3c, err := createS3Client(ctx, param, s.K8sClientset)
	if err != nil {
		return nil, err
	}

	// Attempt bucket deletion
	if err = s3c.DeleteBucket(ctx, bucketName); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete bucket due to an internal error")
	}

	return &cosi.DriverDeleteBucketResponse{}, nil
}

// DriverGrantBucketAccess call grants access to an account.
// The account_name in the request shall be used as a unique identifier to create credentials.
//
// NOTE: this call needs to be idempotent.
// The account_id returned in the response will be used as the unique identifier for deleting this access when calling DriverRevokeBucketAccess.
// The returned secret does not need to be the same each call to achieve idempotency.
func (s *Server) DriverGrantBucketAccess(ctx context.Context, req *cosi.DriverGrantBucketAccessRequest) (*cosi.DriverGrantBucketAccessResponse, error) {
	s.log.Info("Received request to grant access to bucket.", "bucketName", req.BucketId, "AccountName", req.Name)
	ctx = context.WithValue(ctx, utils.LoggerKey, &s.log)
	var eMsg string

	// Get the bucket access name from the request
	bucketAccessName := req.Name

	if req.AuthenticationType != cosi.AuthenticationType_Key {
		err := fmt.Errorf("%s authenticationType alone is supported by COSI driver", cosi.AuthenticationType_Key)
		s.log.Error(err, "Unsupported authenticationType")
		return &cosi.DriverGrantBucketAccessResponse{
			AccountId:   bucketAccessName,
			Credentials: nil,
		}, status.Error(codes.InvalidArgument, "Unsupported authenticationType")
	}

	param := req.Parameters
	bucketName := req.BucketId
	userName := utils.USER_PREFIX + bucketAccessName
	policyName := utils.ACCESS_POLICY_PREFIX + bucketAccessName

	name, namespace, err := getSecretNameAndNamespace(ctx, param)

	if err != nil {
		eMsg = "error while retrieving details of secret used in bucket access class"
		s.log.Error(err, eMsg)
		return &cosi.DriverGrantBucketAccessResponse{
			AccountId:   bucketAccessName,
			Credentials: nil,
		}, status.Error(codes.NotFound, eMsg)
	}

	secret, err := getSecret(s.K8sClientset, ctx, name, namespace)
	if err != nil || secret == nil || secret.Data == nil || len(secret.Data) == 0 {
		eMsg = fmt.Sprintf("error while fetching secret %s used in bucket access class", name)
		s.log.Error(err, eMsg)
		return &cosi.DriverGrantBucketAccessResponse{
			AccountId:   bucketAccessName,
			Credentials: nil,
		}, status.Error(codes.NotFound, eMsg)
	}

	secretKey, endpoint, err := createBucketAccess(ctx, userName, policyName, bucketName, secret.Data)
	if err != nil {
		eMsg = fmt.Sprintf("error while creating access for bucket %s.", bucketName)
		s.log.Error(err, eMsg)
		return &cosi.DriverGrantBucketAccessResponse{
			AccountId:   bucketAccessName,
			Credentials: nil,
		}, status.Error(codes.Internal, eMsg)
	}

	userDetails := make(map[string]string)
	userDetails["accessKeyID"] = userName
	userDetails["accessSecretKey"] = secretKey
	userDetails["endpoint"] = endpoint
	userDetails["region"] = "us-east-1"

	s.log.Info("Preparing s3 - COSI CredentialDetails map")
	cred := &cosi.CredentialDetails{
		Secrets: userDetails,
	}

	credMap := make(map[string]*cosi.CredentialDetails)
	credMap["s3"] = cred

	return &cosi.DriverGrantBucketAccessResponse{
		AccountId:   bucketAccessName,
		Credentials: credMap,
	}, nil
}

// DriverRevokeBucketAccess call revokes all access to a particular bucket from a principal.
//
// NOTE: this call needs to be idempotent.
func (s *Server) DriverRevokeBucketAccess(ctx context.Context, req *cosi.DriverRevokeBucketAccessRequest) (*cosi.DriverRevokeBucketAccessResponse, error) {
	s.log.Info("Received request to revoke access to bucket.", "bucketName", req.BucketId, "AccountName", req.AccountId)
	ctx = context.WithValue(ctx, utils.LoggerKey, &s.log)

	bucketName := req.BucketId
	accessName := req.AccountId
	userName := utils.USER_PREFIX + accessName
	policyName := utils.ACCESS_POLICY_PREFIX + accessName
	var eMsg string

	s.log.Info(fmt.Sprintf("BucketName %s, AccountId %s", bucketName, accessName))
	// Get the bucket object
	bucket, err := s.BucketClientset.ObjectstorageV1alpha1().Buckets().Get(ctx, bucketName, metav1.GetOptions{})
	if err != nil {
		s.log.Error(err, "failed to get bucket object", "bucketName", bucketName)
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get details from bucket: %s", bucketName))
	}

	// TODO: This is for time being, until side-car & driver is patched to receive parameters along with the access revoke request
	// Get the parameters from the spec of the bucket
	param := bucket.Spec.Parameters

	name, namespace, err := getSecretNameAndNamespace(ctx, param)

	if err != nil {
		eMsg = "error while retrieving details of secret used in bucket access class"
		s.log.Error(err, eMsg)
		return &cosi.DriverRevokeBucketAccessResponse{}, status.Error(codes.NotFound, eMsg)
	}

	secret, err := getSecret(s.K8sClientset, ctx, name, namespace)
	if err != nil || secret == nil || secret.Data == nil || len(secret.Data) == 0 {
		eMsg = fmt.Sprintf("error while fetching secret %s used in bucket access class", name)
		return &cosi.DriverRevokeBucketAccessResponse{}, status.Error(codes.NotFound, eMsg)
	}

	if err = deleteBucketAccess(ctx, userName, policyName, bucketName, secret.Data); err != nil {
		eMsg = fmt.Sprintf("error while deleting access for bucket %s", bucketName)
		s.log.Error(err, eMsg)
		return &cosi.DriverRevokeBucketAccessResponse{}, status.Error(codes.Internal, eMsg)
	}

	return &cosi.DriverRevokeBucketAccessResponse{}, nil

}

// BucketRequest is the payload for creating a bucket with versioning and other options.
type BucketRequest struct {
	Compression     bool   `json:"Compression"`
	Versioning      string `json:"Versioning,omitempty"`
	Locking         string `json:"Locking,omitempty"`
	RetentionMode   string `json:"RetentionMode,omitempty"`
	ObjectLockDays  int    `json:"ObjectLockDays,omitempty"`
	ObjectLockYears int    `json:"ObjectLockYears,omitempty"`
}

// --- Add new structs for S3 bucket creation ---
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

// S3Client is a client for direct REST API calls to the S3 backend for S3 bucket operations.
type S3Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AccessKey  string
	SecretKey  string
}

// CreateBucket creates a bucket with versioning enabled via direct REST API call.
func (c *S3Client) CreateBucket(ctx context.Context, bucketName string, req BucketRequest) error {
	var logger logr.Logger
	if l, ok := ctx.Value(utils.LoggerKey).(*logr.Logger); ok && l != nil {
		logger = *l
	}

	url := fmt.Sprintf("%s/%s", c.BaseURL, bucketName)

	// Build the S3 JSON payload (minimal for COSI)
	payloadStruct := struct {
		Compression     bool   `json:"Compression"`
		Versioning      string `json:"Versioning,omitempty"`
		Locking         string `json:"Locking,omitempty"`
		RetentionMode   string `json:"RetentionMode,omitempty"`
		ObjectLockDays  int    `json:"ObjectLockDays,omitempty"`
		ObjectLockYears int    `json:"ObjectLockYears,omitempty"`
	}{
		Compression:     req.Compression,
		Versioning:      req.Versioning,
		Locking:         req.Locking,
		RetentionMode:   req.RetentionMode,
		ObjectLockDays:  req.ObjectLockDays,
		ObjectLockYears: req.ObjectLockYears,
	}
	payload, err := json.Marshal(payloadStruct)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to marshal bucket request payload", "bucketName", bucketName, "payload", payloadStruct)
		}
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to create HTTP request", "bucketName", bucketName)
		}
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Sign the request using AWS Signature Version 4
	service := "s3"
	err = utils.SignAWSV4(httpReq, c.AccessKey, c.SecretKey, "", service, payload)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to sign request with AWS4-HMAC-SHA256", "bucketName", bucketName)
		}
		return err
	}
	httpReq.Body = io.NopCloser(bytes.NewReader(payload))
	httpReq.ContentLength = int64(len(payload))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "HTTP request to create bucket failed", "bucketName", bucketName, "url", url)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		if logger.GetSink() != nil {
			logger.Error(fmt.Errorf("unexpected status code: %d", resp.StatusCode), "Failed to create bucket", "bucketName", bucketName, "status", resp.StatusCode, "response", string(body), "payload", string(payload), "url", url)
		}
		return fmt.Errorf("failed to create bucket: %s", string(body))
	}
	return nil
}

// DeleteBucket deletes a bucket via direct REST API call.
func (c *S3Client) DeleteBucket(ctx context.Context, bucketName string) error {
	url := fmt.Sprintf("%s/%s", c.BaseURL, bucketName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	httpReq.SetBasicAuth(c.AccessKey, c.SecretKey)
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete bucket: %s", string(body))
	}
	return nil
}

// SetObjectLockConfiguration sends a PUT request to set object lock config on a bucket.
func (c *S3Client) SetObjectLockConfiguration(ctx context.Context, bucketName, enabled, mode string, days, years int) error {
	var logger logr.Logger
	if l, ok := ctx.Value(utils.LoggerKey).(*logr.Logger); ok && l != nil {
		logger = *l
	}

	url := fmt.Sprintf("%s/%s?object-lock", c.BaseURL, bucketName)
	config := ObjectLockConfiguration{
		Xmlns:             "http://s3.amazonaws.com/doc/2006-03-01/",
		ObjectLockEnabled: enabled,
	}
	if mode != "" || days > 0 || years > 0 {
		config.Rule = &ObjectLockRule{
			DefaultRetention: ObjectLockDefaultRetention{
				Mode:  mode,
				Days:  days,
				Years: years,
			},
		}
	}
	body, err := xml.Marshal(config)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to marshal object lock XML", "bucketName", bucketName)
		}
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to create object lock HTTP request", "bucketName", bucketName)
		}
		return err
	}
	httpReq.Header.Set("Content-Type", "application/xml")

	// Sign the request using AWS Signature Version 4
	service := "s3"
	err = utils.SignAWSV4(httpReq, c.AccessKey, c.SecretKey, "", service, body)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to sign object lock request", "bucketName", bucketName)
		}
		return err
	}
	httpReq.Body = io.NopCloser(bytes.NewReader(body))
	httpReq.ContentLength = int64(len(body))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "HTTP request to set object lock failed", "bucketName", bucketName, "url", url)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		if logger.GetSink() != nil {
			logger.Error(fmt.Errorf("unexpected status code: %d", resp.StatusCode), "Failed to set object lock", "bucketName", bucketName, "status", resp.StatusCode, "response", string(respBody), "url", url)
		}
		return fmt.Errorf("failed to set object lock: %s", string(respBody))
	}
	return nil
}

// --- Modular parameter parsing helpers ---
func parseVersioning(param map[string]string) string {
	for k, v := range param {
		if strings.EqualFold(k, "versioning") && strings.EqualFold(v, FeatureEnabled) {
			return FeatureEnabled
		}
	}
	return FeatureDisabled
}

func parseCompression(param map[string]string) bool {
	for k, v := range param {
		if strings.EqualFold(k, "compression") {
			return strings.EqualFold(v, FeatureEnabled)
		}
	}
	return false // default to false if not specified
}

func parseObjectLock(param map[string]string) (locking, retentionMode string, days, years int) {
	for k, v := range param {
		switch strings.ToLower(k) {
		case "locking":
			if strings.EqualFold(v, FeatureEnabled) {
				locking = FeatureEnabled
			}
		case "retentionmode":
			retentionMode = v
		case "objectlockdays":
			fmt.Sscanf(v, "%d", &days)
		case "objectlockyears":
			fmt.Sscanf(v, "%d", &years)
		}
	}
	return
}

func parseBucketTags(param map[string]string) map[string]string {
	tags := make(map[string]string)
	// Support legacy tag:key and new bucketTags parameter
	for k, v := range param {
		if strings.HasPrefix(strings.ToLower(k), "tag:") {
			tagKey := strings.TrimPrefix(k, "tag:")
			tags[tagKey] = v
		}
	}
	if tagString, ok := param["bucketTags"]; ok {
		pairs := strings.Split(tagString, ",")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			if strings.Contains(pair, "=") {
				kv := strings.SplitN(pair, "=", 2)
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])
				if key != "" {
					tags[key] = val
				}
			} else {
				// key only, value is empty
				tags[pair] = ""
			}
		}
	}
	return tags
}

// --- Add PutBucketTagging to S3Client ---
func (c *S3Client) PutBucketTagging(ctx context.Context, bucketName string, tags map[string]string) error {
	var logger logr.Logger
	if l, ok := ctx.Value(utils.LoggerKey).(*logr.Logger); ok && l != nil {
		logger = *l
	}

	type Tag struct {
		Key   string `xml:"Key"`
		Value string `xml:"Value"`
	}
	type TagSet struct {
		Tags []Tag `xml:"Tag"`
	}
	type Tagging struct {
		XMLName xml.Name `xml:"Tagging"`
		TagSet  TagSet   `xml:"TagSet"`
	}

	tagList := make([]Tag, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, Tag{Key: k, Value: v})
	}
	tagging := Tagging{TagSet: TagSet{Tags: tagList}}
	body, err := xml.Marshal(tagging)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to marshal bucket tags XML", "bucketName", bucketName)
		}
		return err
	}

	url := fmt.Sprintf("%s/%s?tagging", c.BaseURL, bucketName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to create bucket tagging HTTP request", "bucketName", bucketName)
		}
		return err
	}
	httpReq.Header.Set("Content-Type", "application/xml")

	// Sign the request using AWS Signature Version 4
	service := "s3"
	err = utils.SignAWSV4(httpReq, c.AccessKey, c.SecretKey, "", service, body)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "Failed to sign bucket tagging request", "bucketName", bucketName)
		}
		return err
	}
	httpReq.Body = io.NopCloser(bytes.NewReader(body))
	httpReq.ContentLength = int64(len(body))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		if logger.GetSink() != nil {
			logger.Error(err, "HTTP request to set bucket tags failed", "bucketName", bucketName, "url", url)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		if logger.GetSink() != nil {
			logger.Error(fmt.Errorf("unexpected status code: %d", resp.StatusCode), "Failed to set bucket tags", "bucketName", bucketName, "status", resp.StatusCode, "response", string(respBody), "url", url)
		}
		return fmt.Errorf("failed to set bucket tags: %s", string(respBody))
	}
	return nil
}

// Helper to create S3Client using secret data
func createS3Client(ctx context.Context, parameters map[string]string, kcs *kubernetes.Clientset) (*S3Client, error) {
	secretName, secretNamespace, err := getSecretNameAndNamespace(ctx, parameters)
	if err != nil {
		return nil, err
	}
	accessKey, secretKey, endpoint, err := getDataFromSecret(ctx, secretName, secretNamespace, kcs)
	if err != nil {
		return nil, err
	}
	return &S3Client{
		BaseURL:    endpoint,
		HTTPClient: &http.Client{},
		AccessKey:  accessKey,
		SecretKey:  secretKey,
	}, nil
}

// Get the name and namespace of the secret
func getSecretNameAndNamespace(ctx context.Context, parameters map[string]string) (secretName string, secretNamespace string, err error) {
	log := utils.GetLoggerFromContext(ctx)
	// Get the secret's namespace
	secretNamespace = parameters[utils.CosiUserSecretNamespaceKey]
	// If secret's namespace is not set, retrieve the pod's namespace
	if secretNamespace == "" {
		secretNamespace = os.Getenv(utils.PodNamespaceEnv)
	}

	// Get the secret's name
	secretName = parameters[utils.CosiUserSecretNameKey]

	// Return with error if any field is not set
	if secretName == "" ||
		secretNamespace == "" {
		log.Error(errEmptySecretParameters, "cosiUserSecretName and cosiUserSecretNamespace cannot be empty", "cosiUserSecretName", secretName, "cosiUserSecretNamespace", secretNamespace)
		return "", "", status.Error(codes.InvalidArgument, "cosiUserSecretName and cosiUserSecretNamespace are required parameters in the bucket class")
	}

	return secretName, secretNamespace, nil
}

// Get the access and secret keys, and the endpoint from the secret
func getDataFromSecret(ctx context.Context, secretName string, secretNamespace string, kcs *kubernetes.Clientset) (accessKey string, secretKey string, endpoint string, err error) {
	log := utils.GetLoggerFromContext(ctx)
	// Get secret details
	secret, err := getSecret(kcs, ctx, secretName, secretNamespace)
	if err != nil {
		log.Error(err, "failed to get secret", "secretName", secretName, "secretNamespace", secretNamespace)
		return "", "", "", status.Error(codes.Internal, fmt.Sprintf("failed to get secret: %s from namespace: %s", secretName, secretNamespace))
	}
	// Extract the credentials and endpoint from the secret
	accessKey = string(secret.Data["accessKey"])
	secretKey = string(secret.Data["secretKey"])
	endpoint = string(secret.Data["endpoint"])

	// Return error if any field is not set
	if accessKey == "" ||
		secretKey == "" ||
		endpoint == "" {
		log.Error(errEmptySecretData, "accessKey, secretKey and endpoint values cannot be empty", "accessKey", accessKey, "secretKey", secretKey, "endpoint", endpoint)
		return "", "", "", status.Error(codes.InvalidArgument, "accessKey, secretKey and endpoint are required data in the secret")
	}

	return accessKey, secretKey, endpoint, nil
}

func getSecret(client *kubernetes.Clientset, ctx context.Context, name, namespace string) (*v1.Secret, error) {
	return client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// Retrieves the GLCP credentials(required to communicate with DSCC) from the passed secret data
// returns error, if any of the necessary credentials are missing in the passed secret data
func getGLCPCDetails(data map[string][]byte) (*utils.IAMCredentials, error) {
	getData := func(key string) (str string, err error) {
		if val, ok := data[key]; ok {
			str = string(val)[:]
		} else {
			err = fmt.Errorf("unable to find key %s from mounted secret", key)
		}
		return
	}

	proxy := os.Getenv(utils.PROXY)
	glcpCommonCloud := os.Getenv(utils.GLCP_COMMON_CLOUD)

	if len(glcpCommonCloud) == 0 {
		return nil, fmt.Errorf("unable to fetch GLCP cloud URL form environment")
	}
	glcpUserClientId, err := getData(utils.GLCP_USER_CLIENTID)
	if err != nil {
		return nil, err
	}
	glcpUserSecretKey, err := getData(utils.GLCP_USER_SECRET_KEY)
	if err != nil {
		return nil, err
	}
	dsccZone, err := getData(utils.DSCC_ZONE)
	if err != nil {
		return nil, err
	}
	clusterSerialNumber, err := getData(utils.ALLETRA_MP_X10K_SNO)
	if err != nil {
		return nil, err
	}
	endpoint, err := getData(utils.ENDPOINT)
	if err != nil {
		return nil, err
	}
	return &utils.IAMCredentials{
		GLCPUser:          glcpUserClientId,
		GLCPUserSecretKey: glcpUserSecretKey,
		GLCPCommonCloud:   glcpCommonCloud,
		DSCCZone:          dsccZone,
		SystemId:          clusterSerialNumber,
		Endpoint:          endpoint,
		Proxy:             proxy,
	}, nil
}

// Create a S3 user with privileges to access a bucket
// returns an error when S3 user,policy creation fails in DSCC
func createBucketAccess(ctx context.Context, userName, policyName, bucketName string, data map[string][]byte) (string, string, error) {
	log := utils.GetLoggerFromContext(ctx)
	credentials, err := getGLCPCDetails(data)
	if err != nil {
		return "", "", err
	}
	token, err := getAccessToken(credentials, log)
	if err != nil || len(token) == 0 {
		return "", "", err
	}

	api_client := iam.NewAPIClient(credentials.DSCCZone, token, credentials.Proxy)
	client, _ := api_client.GetAPIClient()
	tclient, err := api_client.GetTaskAPIClient()
	if err != nil {
		log.Error(err, fmt.Sprintf("error creating DSCC API %s, client for IAM operations.", credentials.DSCCZone))
		return "", "", err
	}
	// Create S3 User
	u := iam.NewS3User(userName, credentials.SystemId, client, ctx)

	log.Info(fmt.Sprintf("Checking if s3 user %s, exists in DSCC", userName))
	exist, err := u.UserExists()
	if err != nil {
		log.Error(err, "error fetching user details from DSCC. Please verify the connectivity with DSCC, ", credentials.DSCCZone)
		return "", "", err
	}

	var secretKey string
	if !exist {
		log.Info(fmt.Sprintf("Creating s3 user %s, in DSCC", userName))
		secretKey, err = u.CreateS3User()
	} else {
		log.Info(fmt.Sprintf("S3 user %s, is seen to be existing in DSCC, resetting password & reusing it for bucket access", userName))
		secretKey, err = u.ResetPassword()
	}
	if err != nil || len(secretKey) == 0 {
		log.Error(err, "error while creating s3 user in DSCC.")
		return "", "", err
	}
	// Create S3 Access policy
	p := iam.NewS3Policy(policyName, bucketName, credentials.SystemId, client, ctx)

	log.Info(fmt.Sprintf("Checking if access policy %s, exists in DSCC", policyName))
	exist, err = p.PolicyExists()
	if err != nil {
		log.Error(err, "error fetching access policy details from DSCC. Please verify the connectivity with DSCC, ", credentials.DSCCZone)
		return "", "", err
	}

	if !exist {
		log.Info(fmt.Sprintf("Creating access policy %s, in DSCC", policyName))
		task, err := p.CreateS3AccessPolicy()
		if task == nil && err != nil {
			log.Error(err, "error while creating s3 access policy in DSCC.")
			return "", "", err
		}
		if taskSuccessful, err := iam.TaskStatus(task.TaskUri, credentials.SystemId, tclient, ctx); !taskSuccessful {
			msg := "failed to create s3 access policy in DSCC."
			log.Error(err, msg)
			if err == nil {
				err = fmt.Errorf("%s", msg)
			}
			return "", "", err

		}
	} else {
		log.Info(fmt.Sprintf("Access policy %s, is seen to be existing in DSCC, reusing it for bucket access", policyName))
	}

	// Applying access policy to S3 user
	log.Info(fmt.Sprintf("Applying access policy %s to s3 user %s", policyName, userName))
	task, err := u.ApplyPolicy([]string{policyName})
	if task == nil && err != nil {
		log.Error(err, "error while applying policy to user.")
		return "", "", err
	}
	if taskSuccessful, err := iam.TaskStatus(task.TaskUri, credentials.SystemId, tclient, ctx); !taskSuccessful {
		msg := "failure seen with applying policy to s3 user in DSCC."
		log.Error(err, msg)
		if err == nil {
			err = fmt.Errorf("%s", msg)
		}
		return "", "", err
	}

	return secretKey, credentials.Endpoint, nil
}

// Deletes the S3 user which has privileges to access a bucket
// returns an error when S3 user,policy deletion fails in DSCC
func deleteBucketAccess(ctx context.Context, userName, policyName, bucketName string, data map[string][]byte) error {
	log := utils.GetLoggerFromContext(ctx)

	credentials, err := getGLCPCDetails(data)
	if err != nil {
		return err
	}

	token, err := getAccessToken(credentials, log)
	if err != nil {
		return err
	}

	//Get SDK CLients
	api_client := iam.NewAPIClient(credentials.DSCCZone, token, credentials.Proxy)
	client, _ := api_client.GetAPIClient()
	tclient, err := api_client.GetTaskAPIClient()
	if err != nil {
		log.Error(err, fmt.Sprintf("error creating DSCC API %s, client for IAM operations.", credentials.DSCCZone))
		return err
	}

	u := iam.NewS3User(userName, credentials.SystemId, client, ctx)

	log.Info(fmt.Sprintf("Checking if s3 user %s, exists in DSCC", userName))
	exist, err := u.UserExists()
	if err != nil {
		log.Error(err, fmt.Sprintf("error fetching user details from DSCC %s. Please verify the connectivity with DSCC.", credentials.DSCCZone))
		return err
	}

	if exist {
		log.Info(fmt.Sprintf("Deleting S3 user %s", userName))
		task, err := u.DeleteS3User()
		if task == nil && err != nil {
			log.Error(err, "error while deleting s3 user.")
			return err
		}
		if taskSuccessful, err := iam.TaskStatus(task.TaskUri, credentials.SystemId, tclient, ctx); !taskSuccessful {
			msg := "failed to delete s3 user in DSCC."
			log.Error(err, msg)
			if err == nil {
				err = fmt.Errorf("%s", msg)
			}
			return err

		}

	} else {
		log.Info(fmt.Sprintf("S3 user %s, doesn't exist in DSCC", userName))
	}

	p := iam.NewS3Policy(policyName, bucketName, credentials.SystemId, client, ctx)

	log.Info(fmt.Sprintf("Checking if s3 access policy %s, exists in DSCC", policyName))
	exist, err = p.PolicyExists()
	if err != nil {
		log.Error(err, fmt.Sprintf("error fetching policy details from DSCC %s. Please verify the connectivity with DSCC.", credentials.DSCCZone))
		return err
	}

	if exist {
		log.Info(fmt.Sprintf("Deleting S3 access policy %s", userName))
		task, err := p.DeleteS3AccessPolicy()
		if task == nil && err != nil {
			log.Error(err, "error while deleting s3 access policy.")
			return err
		}
		if taskSuccessful, err := iam.TaskStatus(task.TaskUri, credentials.SystemId, tclient, ctx); !taskSuccessful {
			msg := "failed to delete s3 access policy in DSCC."
			log.Error(err, msg)
			if err == nil {
				err = fmt.Errorf("%s", msg)
			}
			return err

		}
	} else {
		log.Info(fmt.Sprintf("S3 access policy %s, doesn't exist in DSCC", userName))
	}
	return nil

}

// Fetches a fresh bearer token to access DSCC, through GLCP
func getAccessToken(credentials *utils.IAMCredentials, log *logr.Logger) (string, error) {
	ts := iam.NewTokenService(credentials.GLCPCommonCloud, credentials.GLCPUser, credentials.GLCPUserSecretKey, credentials.Proxy, log)
	token, err := ts.GetAccessToken()
	return token, err
}
