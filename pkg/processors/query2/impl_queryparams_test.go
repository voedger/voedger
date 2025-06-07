/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"encoding/json"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_BasicUsage(t *testing.T) {
	require := require.New(t)

	t.Run("Test_BasicUsage", func(t *testing.T) {
		params := map[string]string{
			"order":   "id,created_at",
			"limit":   "10",
			"skip":    "5",
			"include": "name,email",
			"keys":    "id,name",
			"where":   `{"id_department":123456,"number":{"$gte":100,"$lte":200}}`,
			"args":    `{"key":"value"}`,
		}
		parsedParams, err := ParseQueryParams(params)
		require.NoError(err)
		require.NotNil(parsedParams)
		require.NotNil(parsedParams.Constraints)
		require.NotNil(parsedParams.Argument)
		require.Equal([]string{"id", "created_at"}, parsedParams.Constraints.Order)
		require.Equal(10, parsedParams.Constraints.Limit)
		require.Equal(5, parsedParams.Constraints.Skip)
		require.Equal([]string{"name", "email"}, parsedParams.Constraints.Include)
		require.Equal([]string{"id", "name"}, parsedParams.Constraints.Keys)
		require.Equal(Where{
			"id_department": json.Number("123456"),
			"number": map[string]interface{}{
				"$gte": json.Number("100"),
				"$lte": json.Number("200"),
			},
		}, parsedParams.Constraints.Where)
		require.Equal(map[string]interface{}{
			"key": "value",
		}, parsedParams.Argument)
	})

	t.Run("error", func(t *testing.T) {
		params := map[string]string{
			"order": "id,created_at",
			"limit": "ten",
		}
		parsedParams, err := ParseQueryParams(params)
		require.ErrorContains(err, "invalid 'limit' parameter")
		require.Nil(parsedParams)
	})

	t.Run("empty", func(t *testing.T) {
		params := map[string]string{}
		parsedParams, err := ParseQueryParams(params)
		require.NoError(err)
		require.NotNil(parsedParams)
		require.Nil(parsedParams.Constraints)
		require.Nil(parsedParams.Argument)
	})

}
