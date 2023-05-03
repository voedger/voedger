/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

const (
	loginCnt = 20
)

// Test_Race_SUCreateLogin: Create login & sign up
func Test_Race_SUCreateLogin(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	wg := &sync.WaitGroup{}
	for i := 0; i < loginCnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			login := fmt.Sprintf("login%s", strconv.Itoa(vit.NextNumber()))
			// note: "token expired" messages here because we do not wait for user profile init. Next test -> next day -> "token expired" on updateOwner in logs (tests not failed)
			vit.SignUp(login, "1", istructs.AppQName_test1_app1)
		}()
	}
	wg.Wait()
}

// Test_Race_SUsignUpIn: sign up,sign in with existing logins & sign in with un-existing logins
func Test_Race_SUsignUpIn(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	wg := &sync.WaitGroup{}
	logins := make(chan it.Login, loginCnt)
	for i := 0; i < loginCnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			login := vit.SignUp(fmt.Sprintf("login%s", strconv.Itoa(vit.NextNumber())), "1", istructs.AppQName_test1_app1)
			logins <- login
		}()
	}
	wg.Wait()
	close(logins)

	wgin := &sync.WaitGroup{}
	for login := range logins {
		wgin.Add(1)
		go func() {
			defer wgin.Done()
			vit.SignUp(fmt.Sprintf("login%s", strconv.Itoa(vit.NextNumber())), "1", istructs.AppQName_test1_app1)
		}()
		wgin.Add(1)
		go func(login it.Login) {
			defer wgin.Done()
			vit.SignIn(login, it.DoNotFailOnTimeout())
		}(login)
	}
	wgin.Wait()
}
