/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package mock

import "github.com/stretchr/testify/mock"

func AnythingOfType(t string) mock.AnythingOfTypeArgument {
	return mock.AnythingOfTypeArgument(t)
}
