/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storages

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestFederationCommandRetryOn(t *testing.T) {
	require := require.New(t)
	const body = `{"args":{"Int":42}}`
	type request struct {
		method, path, body, authorization string
		readError, writeError             error
	}
	for _, test := range []struct {
		name             string
		retryOn          *string
		expectedCodes    string
		statuses         []int
		expectedRequests int
		expectedStatus   int32
		expectedError    string
	}{
		{name: "Selected statuses retry until success", retryOn: new("503,502"), statuses: []int{503, 502, 200}, expectedRequests: 3, expectedStatus: 200},
		{name: "Empty list entries are skipped", retryOn: new(",503,,502,"), statuses: []int{503, 502, 200}, expectedRequests: 3, expectedStatus: 200},
		{name: "Omitted retry list preserves one attempt", expectedCodes: "503", statuses: []int{503}, expectedRequests: 1, expectedStatus: 503},
		{name: "Empty retry list preserves one attempt", retryOn: new(""), expectedCodes: "503", statuses: []int{503}, expectedRequests: 1, expectedStatus: 503},
		{name: "Unselected status returns without retry", retryOn: new("502"), statuses: []int{503}, expectedRequests: 1, expectedError: "status 503"},
		{name: "Expected status still retries when selected", retryOn: new("503"), expectedCodes: "200,503", statuses: []int{503, 200}, expectedRequests: 2, expectedStatus: 200},
		{name: "Final status must still be expected", retryOn: new("503"), expectedCodes: "201", statuses: []int{503, 200}, expectedRequests: 2, expectedError: "status 200, expected [201]"},
		{name: "Malformed retry list sends no request", retryOn: new("503,invalid"), expectedError: "invalid syntax"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requestsMu sync.Mutex
			var actualRequests []request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestsMu.Lock()
				defer requestsMu.Unlock()
				requestBody, readError := io.ReadAll(r.Body)
				status := http.StatusOK
				if len(actualRequests) < len(test.statuses) {
					status = test.statuses[len(actualRequests)]
				}
				w.WriteHeader(status)
				_, writeError := io.WriteString(w, `{"newIDs":{"1":42},"result":{"Int":42}}`)
				actualRequests = append(actualRequests, request{r.Method, r.URL.Path, string(requestBody), r.Header.Get("Authorization"), readError, writeError})
			}))
			defer server.Close()
			serverURL, err := url.Parse(server.URL)
			require.NoError(err)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			client, cleanup := federation.New(ctx, func() *url.URL { return serverURL }, nil, nil)
			defer cleanup()
			appStructs, _ := mockedStructs(t)
			storage := NewFederationCommandStorage(func() istructs.IAppStructs { return appStructs }, func() istructs.WSID { return 123 }, client, nil, nil)
			key := storage.NewKeyBuilder(appdef.NullQName, nil)
			key.PutQName(sys.Storage_FederationCommand_Field_Command, appdef.NewQName("test", "Command"))
			key.PutString(sys.Storage_FederationCommand_Field_Token, "token")
			key.PutString(sys.Storage_FederationCommand_Field_Body, body)
			key.PutString(sys.Storage_FederationCommand_Field_ExpectedCodes, test.expectedCodes)
			if test.retryOn != nil {
				key.PutString(sys.Storage_FederationCommand_Field_RetryOn, *test.retryOn)
			}

			actual, actualError := storage.(state.IWithGet).Get(key)

			// Request counts distinguish selected retries, unchanged defaults, and parsing failures.
			requestsMu.Lock()
			requests := slices.Clone(actualRequests)
			requestsMu.Unlock()
			require.Len(requests, test.expectedRequests)
			for _, actualRequest := range requests {
				require.NoError(actualRequest.readError)
				require.NoError(actualRequest.writeError)
				require.Equal(http.MethodPost, actualRequest.method)
				require.Equal("/api/"+testAppQName.String()+"/123/c.test.Command", actualRequest.path)
				require.JSONEq(body, actualRequest.body)
				require.Equal("Bearer token", actualRequest.authorization)
			}
			if test.expectedError != "" {
				require.ErrorContains(actualError, test.expectedError)
				require.Nil(actual)
				return
			}
			require.NoError(actualError)
			// The final response must survive the storage's command-result decoding.
			require.Equal(test.expectedStatus, actual.AsInt32(sys.Storage_FederationCommand_Field_StatusCode))
			require.Equal(int32(42), actual.AsValue(sys.Storage_FederationCommand_Field_Result).AsInt32("Int"))
			require.Equal(int64(42), actual.AsValue(sys.Storage_FederationCommand_Field_NewIDs).AsInt64("1"))
		})
	}
}

func TestFederationCommandKeyEqualityIncludesRetryOn(t *testing.T) {
	require := require.New(t)
	storage := NewFederationCommandStorage(nil, nil, nil, nil, nil)
	actual := storage.NewKeyBuilder(appdef.NullQName, nil)
	expected := storage.NewKeyBuilder(appdef.NullQName, nil)
	actual.PutString(sys.Storage_FederationCommand_Field_RetryOn, "503,502")
	expected.PutString(sys.Storage_FederationCommand_Field_RetryOn, "503,502")
	// Identical keys remain reusable; changing retry behavior must distinguish keys.
	require.True(actual.Equals(expected))
	expected.PutString(sys.Storage_FederationCommand_Field_RetryOn, "503")
	require.False(actual.Equals(expected))
	require.False(expected.Equals(actual))
}
