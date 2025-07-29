// © Copyright 2025 Hewlett Packard Enterprise Development LP

package utils

import (
	"bytes"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

// SignAWSV4 signs an HTTP request using AWS Signature Version 4.
func SignAWSV4(req *http.Request, accessKey, secretKey, region, service string, payload []byte) error {
	creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
	signer := v4.NewSigner(creds)
	body := bytes.NewReader(payload)
	_, err := signer.Sign(req, body, service, region, time.Now())
	return err
}
