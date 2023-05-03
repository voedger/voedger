/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package signupin

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

func CheckAppWSID(login string, urlWSID istructs.WSID, appWSAmount istructs.AppWSAmount) error {
	crc16 := utils.CRC16([]byte(login))
	appWSID := istructs.WSID(crc16%uint16(appWSAmount)) + istructs.FirstBaseAppWSID
	expectedAppWSID := istructs.NewWSID(urlWSID.ClusterID(), appWSID)
	if expectedAppWSID != urlWSID {
		return coreutils.NewHTTPErrorf(http.StatusForbidden, "wrong url WSID: ", expectedAppWSID, " expected, ", urlWSID, " got")
	}
	return nil
}

func GetPasswordSaltedHash(pwd string) (pwdSaltedHash []byte, err error) {
	if pwdSaltedHash, err = bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost); err != nil {
		err = fmt.Errorf("password salting & hashing failed: %w", err)
	}
	return
}

// istructs.NullRecordID means not found
func GetCDocLoginID(st istructs.IState, appWSID istructs.WSID, appName string, login string) (cdocLoginID istructs.RecordID, err error) {
	kb, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewLoginIdx)
	if err != nil {
		return istructs.NullRecordID, err
	}
	loginHash := GetLoginHash(login)
	kb.PutInt64(field_AppWSID, int64(appWSID))
	kb.PutString(field_AppIDLoginHash, appName+"/"+loginHash)
	loginIdx, ok, err := st.CanExist(kb)
	if err != nil {
		return istructs.NullRecordID, err
	}
	if !ok {
		return istructs.NullRecordID, nil
	}
	return loginIdx.AsRecordID(field_CDocLoginID), nil

}

func GetLoginHash(login string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(login)))
}

func GetCDocLogin(login string, st istructs.IState, appWSID istructs.WSID, appName string) (cdocLogin istructs.IStateValue, err error) {
	cdocLoginID, err := GetCDocLoginID(st, appWSID, appName, login)
	if err != nil {
		return nil, err
	}
	if cdocLoginID == istructs.NullRecordID {
		return nil, coreutils.NewHTTPErrorf(http.StatusUnauthorized, "login ", login, " does not exist")
	}

	kb, err := st.KeyBuilder(state.RecordsStorage, authnz.QNameCDocLogin)
	if err != nil {
		return nil, err
	}
	kb.PutRecordID(state.Field_ID, cdocLoginID)
	return st.MustExist(kb)
}

func CheckPassword(cdocLogin istructs.IStateValue, pwd string) error {
	if err := bcrypt.CompareHashAndPassword(cdocLogin.AsBytes(field_PwdHash), []byte(pwd)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return coreutils.NewHTTPErrorf(http.StatusUnauthorized, "password is incorrect")
		}
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	return nil
}

func ChangePasswordCDocLogin(cdocLogin istructs.IStateValue, newPwd string, intents istructs.IIntents, st istructs.IState) error {
	kb, err := st.KeyBuilder(state.RecordsStorage, appdef.NullQName)
	if err != nil {
		return err
	}
	loginUpdater, err := intents.UpdateValue(kb, cdocLogin)
	if err != nil {
		return err
	}
	newPwdSaltedHash, err := GetPasswordSaltedHash(newPwd)
	if err != nil {
		return err
	}
	loginUpdater.PutBytes(field_PwdHash, newPwdSaltedHash)
	return nil
}

func ChangePassword(login string, st istructs.IState, intents istructs.IIntents, wsid istructs.WSID, appName string, newPwd string) error {
	cdocLogin, err := GetCDocLogin(login, st, wsid, appName)
	if err != nil {
		return err
	}
	return ChangePasswordCDocLogin(cdocLogin, newPwd, intents, st)
}
