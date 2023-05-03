/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package heeus_it

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
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	wg := &sync.WaitGroup{}
	for i := 0; i < loginCnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			login := fmt.Sprintf("login%s", strconv.Itoa(hit.NextNumber()))
			// note: "token expired" messages here because we do not wait for user profile init. Next test -> next day -> "token expired" on updateOwner in logs (tests not failed)
			hit.SignUp(login, "1", istructs.AppQName_test1_app1)
		}()
	}
	wg.Wait()
}

// Test_Race_SUsignUpIn: sign up,sign in with existing logins & sign in with un-existing logins
func Test_Race_SUsignUpIn(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	wg := &sync.WaitGroup{}
	logins := make(chan it.Login, loginCnt)
	for i := 0; i < loginCnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			login := hit.SignUp(fmt.Sprintf("login%s", strconv.Itoa(hit.NextNumber())), "1", istructs.AppQName_test1_app1)
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
			hit.SignUp(fmt.Sprintf("login%s", strconv.Itoa(hit.NextNumber())), "1", istructs.AppQName_test1_app1)
		}()
		wgin.Add(1)
		go func(login it.Login) {
			defer wgin.Done()
			hit.SignIn(login, it.DoNotFailOnTimeout())
		}(login)
	}
	wgin.Wait()
}
