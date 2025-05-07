/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

func TestWrongTypes(t *testing.T) {
	require := require.New(t)
	serviceChannel := make(iprocbus.ServiceChannel)
	done := make(chan struct{})
	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)

	appParts, cleanAppParts, appTokens, statelessResources := deployTestAppWithSecretToken(require, nil)

	defer cleanAppParts()

	queryProcessor := ProvideServiceFactory()(
		serviceChannel,
		appParts,
		3, // maxPrepareQueries

		imetrics.Provide(), "vvm", authn, itokensjwt.TestTokensJWT(), nil, statelessResources, isecretsimpl.TestSecretReader)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		queryProcessor.Run(ctx)
		close(done)
	}()

	tests := []struct {
		name string
		body string
		err  string
	}{
		{
			name: "Elements must be an array",
			body: `{"elements":{}}`,
			err:  `elements: field "elements" must be an array of objects: field type mismatch`,
		},
		{
			name: "Element must be an object",
			body: `{"elements":[45]}`,
			err:  "elements: each member must be an object: wrong type",
		},
		{
			name: "Element path is wrong",
			body: `{"elements":[{"path":45}]}`,
			err:  `elements: element: field "path" must be a string: field type mismatch`,
		},
		{
			name: "Element fields must be an array",
			body: `{"elements":[{"fields":{}}]}`,
			err:  `elements: element: field "fields" must be an array of objects: field type mismatch`,
		},
		{
			name: "Element fields must be a string or an array of strings",
			body: `{"elements":[{"fields":[4]}]}`,
			err:  "elements: element: must be a sting or an array of strings: wrong type",
		},
		{
			name: "Element refs must be an array",
			body: `{"elements":[{"refs":{}}]}`,
			err:  `elements: element: field "refs" must be an array of objects: field type mismatch`,
		},
		{
			name: "Ref field parameters length must be 2",
			body: `{"elements":[{"refs":[[]]}]}`,
			err:  `elements: element: field 'ref' parameters length must be 2 but got 0`,
		},
		{
			name: "Ref field parameters type must be a string",
			body: `{"elements":[{"refs":[["id",1]]}]}`,
			err:  `elements: element: field 'ref' parameter must a string: wrong type`,
		},
		{
			name: "Ref field parameters type must be a string",
			body: `{"elements":[{"refs":[[1,1]]}]}`,
			err:  `elements: element: field 'ref' parameter must a string`,
		},
		{
			name: "Filters must be an array",
			body: `{"filters":{}}`,
			err:  `filters: field "filters" must be an array of objects: field type mismatch`,
		},
		{
			name: "Filter args not found",
			body: `{"filters":[{}]}`,
			err:  `filters: filter: field 'args' must be present: not found`,
		},
		{
			name: "Filter expr not found",
			body: `{"filters":[{"args":{}}]}`,
			err:  `filters: filter: field "expr" missing: fields are missed`,
		},
		{
			name: "Filter expr is unknown",
			body: `{"filters":[{"args":{},"expr":"<<<"}]}`,
			err:  "filters: filter: expr: filter '<<<' is unknown: wrong type",
		},
		{
			name: "Filter field must be present",
			body: `{"filters":[{"expr":"eq","args":{}}]}`,
			err:  `filters: 'eq' filter: field "field" missing: fields are missed`,
		},
		{
			name: "Filter value must be present",
			body: `{"filters":[{"expr":"eq","args":{"field":"id"}}]}`,
			err:  "filters: 'eq' filter: field 'value' must be present: not found",
		},
		{
			name: "Equals filter wrong args",
			body: `{"filters":[{"expr":"eq","args":""}]}`,
			err:  "filters: 'eq' filter: field 'args' must be an object: wrong type",
		},
		{
			name: "Equals filter options must be an object",
			body: `{"filters":[{"expr":"eq","args":{"field":"id","value":1,"options":[]}}]}`,
			err:  `filters: 'eq' filter: field "options" must be an object: field type mismatch`,
		},
		{
			name: "Equals filter epsilon must be a float64",
			body: `{"filters":[{"expr":"eq","args":{"field":"id","value":1,"options":{"epsilon":"wrong"}}}]}`,
			err:  `filters: 'eq' filter: field "epsilon" must be json.Number: field type mismatch`,
		},
		{
			name: "Not equals filter wrong args",
			body: `{"filters":[{"expr":"notEq","args":""}]}`,
			err:  "filters: 'notEq' filter: field 'args' must be an object: wrong type",
		},
		{
			name: "Not equals filter epsilon must be a float64",
			body: `{"filters":[{"expr":"notEq","args":{"field":"id","value":1,"options":{"epsilon":"wrong"}}}]}`,
			err:  `filters: 'notEq' filter: field "epsilon" must be json.Number: field type mismatch`,
		},
		{
			name: "Greater filter wrong args",
			body: `{"filters":[{"expr":"gt","args":""}]}`,
			err:  "filters: 'gt' filter: field 'args' must be an object: wrong type",
		},
		{
			name: "Less filter wrong args",
			body: `{"filters":[{"expr":"lt","args":""}]}`,
			err:  "filters: 'lt' filter: field 'args' must be an object: wrong type",
		},
		{
			name: "And filter wrong args",
			body: `{"filters":[{"expr":"and","args":""}]}`,
			err:  "filters: 'and' filter: field 'args' must be an array of objects: wrong type",
		},
		{
			name: "And filter wrong member",
			body: `{"filters":[{"expr":"and","args":[""]}]}`,
			err:  "filters: 'and' filter: each 'args' member must be an object: wrong type",
		},
		{
			name: "And filter error in member",
			body: `{"filters":[{"expr":"and","args":[{}]}]}`,
			err:  "filters: 'and' filter: filter: field 'args' must be present: not found",
		},
		{
			name: "Or filter wrong args",
			body: `{"filters":[{"expr":"or","args":""}]}`,
			err:  "filters: 'or' filter: field 'args' must be an array of objects: wrong type",
		},
		{
			name: "Or filter wrong member",
			body: `{"filters":[{"expr":"or","args":[""]}]}`,
			err:  "filters: 'or' filter: each 'args' member must be an object: wrong type",
		},
		{
			name: "Or filter error in member",
			body: `{"filters":[{"expr":"or","args":[{}]}]}`,
			err:  "filters: 'or' filter: filter: field 'args' must be present: not found",
		},
		{
			name: "And filter error in equals filter",
			body: `{"filters":[{"expr":"and","args":[{"expr":"eq","args":{"field":"wrong","value":"wrong"}}]}]}`,
			err:  "filters: 'and' filter: 'eq' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "Or filter error in equals filter",
			body: `{"filters":[{"expr":"or","args":[{"expr":"eq","args":{"field":"wrong","value":"wrong"}}]}]}`,
			err:  "filters: 'or' filter: 'eq' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "And filter error in first equals filter",
			body: `{"elements":[{"fields":["sys.ID"]}],"filters":[{"expr":"and","args":[{"expr":"eq","args":{"field":"wrong","value":"wrong"}},{"expr":"eq","args":{"field":"sys.ID","value":1}}]}]}`,
			err:  "filters: 'and' filter: 'eq' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "OrderBy must be an array",
			body: `{"orderBy":{}}`,
			err:  `orderBy: field "orderBy" must be an array of objects: field type mismatch`,
		},
		{
			name: "OrderBy field not found",
			body: `{"orderBy":[{}]}`,
			err:  `orderBy: orderBy: field "field" missing: fields are missed`,
		},
		{
			name: "OrderBy desc must be a boolean",
			body: `{"orderBy":[{"field":"","desc":"wrong"}]}`,
			err:  `orderBy: orderBy: field "desc" must be a boolean: field type mismatch`,
		},
		{
			name: "Count must be a int64",
			body: `{"count":{}}`,
			err:  `field "count" must be json.Number: field type mismatch`,
		},
		{
			name: "StartFrom must be a int64",
			body: `{"startFrom":{}}`,
			err:  `field "startFrom" must be json.Number: field type mismatch`,
		},
		{
			name: "Root element fields must be present in result fields",
			body: `{"elements":[{"path":"not/root"},{"fields":["wrong"]}]}`,
			err:  "elements: unknown nested table not: unexpected",
		},
		{
			name: "Equals filter field must be present in root element fields/refs",
			body: `{"elements":[{"fields":["sys.ID"]}],"filters":[{"expr":"eq","args":{"field":"wrong","value":1}}]}`,
			err:  "filters: 'eq' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "Not equals filter field must be present in root element fields/refs",
			body: `{"elements":[{"fields":["sys.ID"]}],"filters":[{"expr":"notEq","args":{"field":"wrong","value":1}}]}`,
			err:  "filters: 'notEq' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "Greater filter field must be present in root element fields/refs",
			body: `{"elements":[{"fields":["sys.ID"]}],"filters":[{"expr":"gt","args":{"field":"wrong","value":1}}]}`,
			err:  "filters: 'gt' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "Less filter field must be present in root element fields/refs",
			body: `{"elements":[{"fields":["sys.ID"]}],"filters":[{"expr":"lt","args":{"field":"wrong","value":1}}]}`,
			err:  "filters: 'lt' filter has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "Order by field must be present in root element fields/refs",
			body: `{"elements":[{"fields":["sys.ID"]}],"orderBy":[{"field":"wrong"}]}`,
			err:  "orderBy has field 'wrong' that is absent in root element fields/refs, please add or change it: unexpected",
		},
		{
			name: "Each element must have unique path",
			body: `{"elements":[{"fields":["sys.ID"],"path":"article"},{"fields":["sys.ID"],"path":"article"}]}`,
			err:  "elements: path 'article' must be unique",
		},
	}

	sysToken := getSystemToken(appTokens)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestSender := bus.NewIRequestSender(testingu.MockTime, bus.GetTestSendTimeout(), func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
				serviceChannel <- NewQueryMessage(context.Background(), appName, partID, wsID, responder, []byte(test.body), qNameFunction, "", sysToken)
			})
			respCh, respMeta, respErr, err := requestSender.SendRequest(context.Background(), bus.Request{})
			require.NoError(err)
			require.Equal(coreutils.ContentType_ApplicationJSON, respMeta.ContentType)
			require.Equal(http.StatusBadRequest, respMeta.StatusCode)
			for range respCh {
			}
			err = *respErr
			require.Contains(err.Error(), test.err, test.name)
		})
	}
	cancel()
	<-done
}
