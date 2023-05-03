/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package workspace

import (
	"encoding/json"
	"io/fs"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONs(t *testing.T) {
	require := require.New(t)
	// just walk over all postinit jsons and check its validity
	require.NoError(fs.WalkDir(Postinit, "postinit", func(filePath string, d fs.DirEntry, _ error) error {
		if d.IsDir() || path.Ext(d.Name()) != ".json" {
			return nil
		}
		content, err := fs.ReadFile(Postinit, filePath)
		require.NoError(err, filePath)
		require.NoError(json.Unmarshal(content, &[]map[string]interface{}{}), filePath)
		return nil
	}))
}
