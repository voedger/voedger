/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package in10nmemv3

import (
	"time"

	"github.com/voedger/voedger/pkg/in10n"
)

func ProvideEx2(quotas in10n.Quotas, now func() time.Time) (nb in10n.IN10nBroker, cleanup func()) {
	return NewN10nBroker(quotas, now)
}
