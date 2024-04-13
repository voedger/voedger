/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package internal

import safe "github.com/voedger/voedger/pkg/state/isafestateapi"

var SafeStateAPI safe.IStateSafeAPI = hostSafeStateAPI{}
