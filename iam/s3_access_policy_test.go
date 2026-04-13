// © Copyright 2024 Hewlett Packard Enterprise Development LP

package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	sdk "hpe-cosi-osp/alletraMPX10000_sdk"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"gotest.tools/v3/assert"
)

var systemId = "DUMMY_SETUP"

func Test_GetS3AccessPolicy(t *testing.T) {
	policy := getNewS3Policy()
	policyName := policy.name
	systemId := policy.systemId
	t.Run("GetS3AccessPolicy successful", func(t *testing.T) {
		gId := int32(1725948764)
		ap := sdk.AccessPolicy{
			Generation: &gId,
			Id:         &policyName,
			Name:       &policyName,
			SystemUid:  &systemId,
		}
		r := http.Response{StatusCode: http.StatusOK}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7GetAccessPolicyByIdExecute", &ap, &r, nil)
		defer patch.Reset()
		policy, err := policy.GetS3AccessPolicy()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.DeepEqual(t, policy, &ap)
	})
	t.Run("GetS3AccessPolicy failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7GetAccessPolicyByIdExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := policy.GetS3AccessPolicy(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_PolicyExists(t *testing.T) {
	policy := getNewS3Policy()
	policyName := policy.name
	systemId := policy.systemId
	t.Run("PolicyExists successful", func(t *testing.T) {
		gId := int32(1725948764)
		ap := sdk.AccessPolicy{
			Generation: &gId,
			Id:         &policyName,
			Name:       &policyName,
			SystemUid:  &systemId,
		}
		r := http.Response{StatusCode: http.StatusOK}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7GetAccessPolicyByIdExecute", &ap, &r, nil)
		defer patch.Reset()
		isExist, err := policy.PolicyExists()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, isExist, true)
	})
	t.Run("PolicyExists false", func(t *testing.T) {
		body, _ := json.Marshal(resourceNotFoundResponse)
		r := http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBuffer(body))}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7GetAccessPolicyByIdExecute", nil, &r, errors.New("404 Not Found"))
		defer patch.Reset()
		isExist, err := policy.PolicyExists()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, isExist, false)
	})

	t.Run("InCorrect DSCC URL", func(t *testing.T) {
		body, _ := json.Marshal(badRoutingResponse)
		r := http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBuffer(body))}
		e := errors.New("404 Not Found")
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7GetAccessPolicyByIdExecute", nil, &r, e)
		defer patch.Reset()
		isExist, err := policy.PolicyExists()
		if err == nil {
			t.Errorf("FAILED: expected error not found")
		}
		assert.Equal(t, isExist, false)
		assert.Equal(t, err, e)
	})

	t.Run("Invalid URL with inline character", func(t *testing.T) {
		err := fmt.Errorf("parse \"https://%s\\n/api/v1/storage-systems/device-type7/XX0000000000XX/dummy-test-policies/acp_ba-zzzz-xxxxx-yyyy-access-policy-11\": net/url: invalid control character in URL", host)
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7GetAccessPolicyByIdExecute", nil, nil, err)
		defer patch.Reset()
		isExist, e := policy.PolicyExists()
		if e == nil {
			t.Errorf("FAILED: expected error not found")
		}
		assert.Equal(t, isExist, false)
		assert.Equal(t, err, e)
	})
}

func Test_CreateS3AccessPolicy(t *testing.T) {
	policy := getNewS3Policy()
	t.Run("CreateS3AccessPolicy successful", func(t *testing.T) {
		taskUi := GetMockTaskResponseUi()
		r := http.Response{StatusCode: http.StatusAccepted}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7CreateAccessPolicyExecute", taskUi, &r, nil)
		defer patch.Reset()
		taskResp, err := policy.CreateS3AccessPolicy()
		if err != nil {
			fmt.Println(err)
			t.Errorf("FAILED: unexpected error")
		}
		assert.DeepEqual(t, taskResp, taskUi)
	})
	t.Run("CreateS3AccessPolicy failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7CreateAccessPolicyExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := policy.CreateS3AccessPolicy(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_UpdateS3AccessPolicy(t *testing.T) {
	policy := getNewS3Policy()
	t.Run("UpdateS3AccessPolicy successful", func(t *testing.T) {
		taskUi := GetMockTaskResponseUi()
		r := http.Response{StatusCode: http.StatusAccepted}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7UpdateAccessPolicyByIdExecute", taskUi, &r, nil)
		defer patch.Reset()
		taskResp, err := policy.UpdateS3AccessPolicy()
		if err != nil {
			fmt.Println(err)
			t.Errorf("FAILED: unexpected error")
		}
		assert.DeepEqual(t, taskResp, taskUi)
	})
	t.Run("UpdateS3AccessPolicy failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7UpdateAccessPolicyByIdExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := policy.UpdateS3AccessPolicy(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_DeleteS3AccessPolicy(t *testing.T) {
	policy := getNewS3Policy()
	t.Run("DeleteS3AccessPolicy successful", func(t *testing.T) {
		taskUi := GetMockTaskResponseUi()
		r := http.Response{StatusCode: http.StatusAccepted}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7DeleteAccessPolicyByIdExecute", taskUi, &r, nil)
		defer patch.Reset()
		taskResp, err := policy.DeleteS3AccessPolicy()
		if err != nil {
			fmt.Println(err)
			t.Errorf("FAILED: unexpected error")
		}
		assert.DeepEqual(t, taskResp, taskUi)
	})
	t.Run("DeleteS3AccessPolicy failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(policy.client.ObjectstoreIdentitiesAPI, "DeviceType7DeleteAccessPolicyByIdExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := policy.DeleteS3AccessPolicy(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func getNewS3Policy() *s3policy {
	policyName := "bucket1_policy"
	bucket := "bucket1"
	apiClient, _ := NewAPIClient(host, token, proxy, "").GetAPIClient()
	return NewS3Policy(policyName, bucket, systemId, apiClient, context.Background())
}
