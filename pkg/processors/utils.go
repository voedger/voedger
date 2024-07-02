/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package processors

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func CheckResponseIntent(st state.IHostState) error {
	kb, err := st.KeyBuilder(state.Response, appdef.NullQName)
	if err != nil {
		// notest
		return err
	}
	respIntent := st.FindIntent(kb)
	if respIntent == nil {
		return nil
	}
	respIntentValue := respIntent.BuildValue()
	statusCode := respIntentValue.AsInt32(state.Field_StatusCode)
	if statusCode == http.StatusOK {
		return nil
	}
	return coreutils.NewHTTPErrorf(int(statusCode), respIntentValue.AsString(state.Field_ErrorMessage))
}
