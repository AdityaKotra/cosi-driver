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

var badRoutingResponse = map[string]string{
	"httpStatusCode": "404",
	"errorCode":      "HPE_GL_ERROR_NOT_FOUND",
	"message":        "Routing details for the customer not found.",
	"debugId":        "CsggFMdSMOR55P3k99--gd9plOITW8d3hxmw5b67dCsw6ww615TNzw==",
}

var resourceNotFoundResponse = map[string]string{
	"errorCode": string(ERR_RESOURCE_NOT_FOUND),
	"error":     "Invalid resource or Resource not found",
}

func Test_GetS3User(t *testing.T) {
	user := getNewS3User()
	name := user.name
	systemId := user.systemId
	gId := int32(1725948764)
	t.Run("GetS3User successful", func(t *testing.T) {
		us := sdk.UserListDetails{
			Generation: *sdk.NewNullableInt32(&gId),
			Id:         &name,
			Name:       &name,
			SystemUid:  &systemId,
		}
		r := http.Response{StatusCode: http.StatusOK}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7GetSingleUserExecute", &us, &r, nil)
		defer patch.Reset()
		_, err := user.GetS3User()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
	})
	t.Run("GetS3User failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7GetSingleUserExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := user.GetS3User(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_UserExists(t *testing.T) {
	user := getNewS3User()
	name := user.name
	systemId := user.systemId
	gId := int32(1725948764)
	t.Run("UserExists successful", func(t *testing.T) {
		ap := sdk.UserListDetails{
			Generation: *sdk.NewNullableInt32(&gId),
			Id:         &name,
			Name:       &name,
			SystemUid:  &systemId,
		}
		r := http.Response{StatusCode: http.StatusOK}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7GetSingleUserExecute", &ap, &r, nil)
		defer patch.Reset()
		isExist, err := user.UserExists()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, isExist, true)
	})
	t.Run("UserExists false", func(t *testing.T) {
		body, _ := json.Marshal(resourceNotFoundResponse)
		r := http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBuffer(body))}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7GetSingleUserExecute", nil, &r, errors.New("404 Not Found"))
		defer patch.Reset()
		isExist, err := user.UserExists()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, isExist, false)
	})

	t.Run("InCorrect DSCC URL", func(t *testing.T) {
		body, _ := json.Marshal(badRoutingResponse)
		r := http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBuffer(body))}
		e := errors.New("404 Not Found")
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7GetSingleUserExecute", nil, &r, e)
		defer patch.Reset()
		isExist, err := user.UserExists()
		if err == nil {
			t.Errorf("FAILED: expected error not found")
		}
		assert.Equal(t, isExist, false)
		assert.Equal(t, err, e)
	})

	t.Run("Invalid URL with inline character", func(t *testing.T) {
		err := fmt.Errorf("parse \"https://%s\\n/api/v1/storage-systems/device-type7/XX0000000000XX/dummy-users/user_zzzz-xxxxx-yyyy-s3-user-11\": net/url: invalid control character in URL", host)
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7GetSingleUserExecute", nil, nil, err)
		defer patch.Reset()
		isExist, e := user.UserExists()
		if e == nil {
			t.Errorf("FAILED: expected error not found")
		}
		assert.Equal(t, isExist, false)
		assert.Equal(t, err, e)
	})

}

func Test_CreateS3User(t *testing.T) {
	user := getNewS3User()
	t.Run("CreateS3User successful", func(t *testing.T) {
		secretKey := "s3_user_key"
		resp := sdk.UserResponseDetails{
			SecretKey: &secretKey,
		}
		r := http.Response{StatusCode: http.StatusCreated}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7CreateUserExecute", &resp, &r, nil)
		defer patch.Reset()
		secret, err := user.CreateS3User()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, secret, secretKey)
	})
	t.Run("CreateS3User failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7CreateUserExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := user.CreateS3User(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_ResetPassword(t *testing.T) {
	user := getNewS3User()
	t.Run("ResetPassword successful", func(t *testing.T) {
		secretKey := "s3_user_key"
		resp := sdk.UserResponseDetails{
			SecretKey: &secretKey,
		}
		r := http.Response{StatusCode: http.StatusCreated}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7EditUserExecute", &resp, &r, nil)
		defer patch.Reset()
		secret, err := user.ResetPassword()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, secret, secretKey)
	})
	t.Run("ResetPassword failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7EditUserExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := user.ResetPassword(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_ApplyPolicy(t *testing.T) {
	user := getNewS3User()
	policyName := "bucket1_policy"
	t.Run("ApplyPolicy successful", func(t *testing.T) {
		taskUi := GetMockTaskResponseUi()
		r := http.Response{StatusCode: http.StatusAccepted}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "ApplyPolicyExecute", taskUi, &r, nil)
		defer patch.Reset()
		taskResp, err := user.ApplyPolicy([]string{policyName})
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.Equal(t, taskResp, taskUi)
	})
	t.Run("ApplyPolicy failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "ApplyPolicyExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := user.ApplyPolicy([]string{policyName}); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func Test_DeleteS3User(t *testing.T) {
	user := getNewS3User()
	t.Run("DeleteS3User successful", func(t *testing.T) {
		taskUi := GetMockTaskResponseUi()
		r := http.Response{StatusCode: http.StatusAccepted}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7DeleteUserByIdExecute", taskUi, &r, nil)
		defer patch.Reset()
		taskResp, err := user.DeleteS3User()
		if err != nil {
			t.Errorf("FAILED: unexpected error")
		}
		assert.DeepEqual(t, taskResp, taskUi)
	})
	t.Run("DeleteS3User failure", func(t *testing.T) {
		r := http.Response{StatusCode: http.StatusBadRequest}
		patch := gomonkey.ApplyMethodReturn(user.client.ObjectstoreIdentitiesAPI, "DeviceType7DeleteUserByIdExecute", nil, &r, nil)
		defer patch.Reset()
		if _, err := user.DeleteS3User(); err == nil {
			t.Errorf("FAILED: expected error not found")
		}
	})
}

func getNewS3User() *s3user {
	token := "bearerdummyoxyzxxzzz12xxxx341111zzzzyyyyyyQQQQQHHHHH"
	userName := "bucket1_user"
	apiClient, _ := NewAPIClient(host, token, proxy, "").GetAPIClient()
	return NewS3User(userName, systemId, apiClient, context.Background())
}
