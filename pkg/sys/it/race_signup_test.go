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

// note: Test_Race_SUCreateLogin is eliminated because chain Test_Race_SUCreateLogin(t) + Test_Race_SUsignUpIn(t) hangs for 30 seconds
// Do SignUps in Test_Race_SUCreateLogin, do not wait for accomplish, go to the next test and tokens became expired. Then workspace are continue to init in async projectors and
// async projectors are failed due of expired tokens -> wait for 30 second before error (projectros/actualizerErrorDelay)

// Test_Race_SUsignUpIn: sign up,sign in with existing logins & sign in with un-existing logins
func Test_Race_SUsignUpIn(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
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
