/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"errors"
	"net/http"
	"testing"

	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/stretchr/testify/require"
)

func TestBasicUsage_SysError(t *testing.T) {
	require := require.New(t)

	t.Run("basic", func(t *testing.T) {
		testError := errors.New("test")
		err := WrapSysError(testError, http.StatusInternalServerError)
		var sysError SysError
		require.ErrorAs(err, &sysError)
		require.Equal("test", sysError.Message)
		require.Equal(http.StatusInternalServerError, sysError.HTTPStatus)
		require.Empty(sysError.Data)
		require.Empty(sysError.QName)
		require.Equal("test", sysError.Error())
		require.Equal(`{"sys.Error":{"HTTPStatus":500,"Message":"test"}}`, sysError.ToJSON())
	})

	t.Run("nil on nil", func(t *testing.T) {
		require.NoError(WrapSysError(nil, http.StatusInternalServerError))
	})

	t.Run("wrap error that is SysError already", func(t *testing.T) {
		err := SysError{
			HTTPStatus: http.StatusOK,
			QName:      istructs.NewQName("my", "test"),
			Message:    "test",
			Data:       "data",
		}
		wrapped := WrapSysError(err, http.StatusInternalServerError)
		require.Equal(err, wrapped)
	})

	t.Run("ToJSON", func(t *testing.T) {
		err := SysError{
			HTTPStatus: http.StatusOK,
			QName:      istructs.NewQName("my", "test"),
			Message:    "test",
			Data:       "data",
		}
		require.Equal(`{"sys.Error":{"HTTPStatus":200,"Message":"test","QName":"my.test","Data":"data"}}`, err.ToJSON())
	})
}
