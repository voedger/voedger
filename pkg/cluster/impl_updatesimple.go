/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

func updateTable(update update, federation federation.IFederation, itokens itokens.ITokens) error {
	jsonFields, err := json.Marshal(update.setFields)
	if err != nil {
		// notest
		return err
	}
	cudBody := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":%s}]}`, update.id, jsonFields)
	sysToken, err := payloads.GetSystemPrincipalToken(itokens, update.appQName)
	if err != nil {
		// notest
		return err
	}
	_, err = federation.Func(fmt.Sprintf("api/%s/%d/c.sys.CUD", update.appQName, update.wsid), cudBody,
		coreutils.WithAuthorizeBy(sysToken),
		coreutils.WithDiscardResponse(),
	)
	return err
}

func validateQuery_Table(update update) error {
	if update.id == 0 {
		return errors.New("record ID is not provided")
	}
	if len(update.key) > 0 {
		return errors.New("conditions are not allowed on update")
	}
	if len(update.setFields) == 0 {
		return errors.New("no fields to update")
	}
	return nil
}
