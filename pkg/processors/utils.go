/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package processors

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func CheckResponseIntent(st state.IHostState) error {
	kb, err := st.KeyBuilder(sys.Storage_Response, appdef.NullQName)
	if err != nil {
		// notest
		return err
	}
	respIntent := st.FindIntent(kb)
	if respIntent == nil {
		return nil
	}
	respIntentValue := respIntent.BuildValue()
	statusCode := respIntentValue.AsInt32(sys.Storage_Response_Field_StatusCode)
	if statusCode == http.StatusOK {
		return nil
	}
	return coreutils.NewHTTPErrorf(int(statusCode), respIntentValue.AsString(sys.Storage_Response_Field_ErrorMessage))
}
