/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package verifier

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

var translationsCatalog = coreutils.GetCatalogFromTranslations(translations)

// called at targetApp/profileWSID
func provideQryInitiateEmailVerification(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens,
	asp istructs.IAppStructsProvider, federation coreutils.IFederation) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryInitiateEmailVerification,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "InitiateEmailVerificationParams")).
			AddField(field_Entity, appdef.DataKind_string, true). // must be string, not QName, because target app could not know that QName. E.g. unknown QName «sys.ResetPasswordByEmailUnloggedParams»: name not found
			AddField(field_Field, appdef.DataKind_string, true).
			AddField(Field_Email, appdef.DataKind_string, true).
			AddField(field_TargetWSID, appdef.DataKind_int64, true).
			AddField(field_ForRegistry, appdef.DataKind_bool, false). // to issue token for sys/registry/pseudoWSID/c.sys.ResetPassword, not for the current app
			AddField(field_Language, appdef.DataKind_string, false).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "InitialEmailVerificationResult")).
			AddField(field_VerificationToken, appdef.DataKind_string, true).(appdef.IDef).QName(),
		provideIEVExec(cfg.Name, itokens, asp, federation),
	))
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQueryInitiateEmailVerification, istructs.RateLimit{
		Period:                InitiateEmailVerification_Period,
		MaxAllowedPerDuration: InitiateEmailVerification_MaxAllowed,
	})
}

// q.sys.InitiateEmailVerification
// called at targetApp/profileWSID
func provideIEVExec(appQName istructs.AppQName, itokens itokens.ITokens, asp istructs.IAppStructsProvider, federation coreutils.IFederation) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		entity := args.ArgumentObject.AsString(field_Entity)
		targetWSID := istructs.WSID(args.ArgumentObject.AsInt64(field_TargetWSID))
		field := args.ArgumentObject.AsString(field_Field)
		email := args.ArgumentObject.AsString(Field_Email)
		forRegistry := args.ArgumentObject.AsBool(field_ForRegistry)
		language := args.ArgumentObject.AsString(field_Language)

		as, err := asp.AppStructs(appQName)
		if err != nil {
			return err
		}
		appTokens := as.AppTokens()
		if forRegistry {
			// issue token for sys/registry/pseduoWSID. That's for c.sys.ResetPassword only for now
			asRegistry, err := asp.AppStructs(istructs.AppQName_sys_registry)
			if err != nil {
				// notest
				return err
			}
			appTokens = asRegistry.AppTokens()
			targetWSID = coreutils.GetPseudoWSID(istructs.NullWSID, email, istructs.MainClusterID)
		}

		verificationToken, verificationCode, err := NewVerificationToken(entity, field, email, appdef.VerificationKind_EMail, targetWSID, itokens, appTokens)
		if err != nil {
			return err
		}

		systemPrincipalToken, err := payloads.GetSystemPrincipalToken(itokens, appQName)
		if err != nil {
			return err
		}

		// c.sys.SendEmailVerificationCode
		body := fmt.Sprintf(`{"args":{"VerificationCode":"%s","Email":"%s","Reason":"%s","Language":"%s"}}`, verificationCode, email, verifyEmailReason, language)
		if _, err = coreutils.FederationFunc(federation.URL(), fmt.Sprintf("api/%s/%d/c.sys.SendEmailVerificationCode", appQName, args.Workspace), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(systemPrincipalToken)); err != nil {
			return fmt.Errorf("c.sys.SendEmailVerificationCode failed: %w", err)
		}

		return callback(&ievResult{verificationToken: verificationToken})
	}
}

func sendEmailVerificationCodeProjector(federation coreutils.IFederation, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		lng := event.ArgumentObject().AsString(field_Language)

		kb, err := st.KeyBuilder(state.SendMailStorage, appdef.NullQName)
		if err != nil {
			return
		}
		reason := event.ArgumentObject().AsString(field_Reason)
		kb.PutString(state.Field_Subject, EmailSubject)
		kb.PutString(state.Field_To, event.ArgumentObject().AsString(Field_Email))
		kb.PutString(state.Field_Body, getVerificationEmailBody(federation, event.ArgumentObject().AsString(field_VerificationCode), reason, language.Make(lng), translationsCatalog))
		kb.PutString(state.Field_From, EmailFrom)
		kb.PutString(state.Field_Host, smtpCfg.Host)
		kb.PutInt32(state.Field_Port, smtpCfg.Port)
		kb.PutString(state.Field_Username, smtpCfg.Username)
		pwd := ""
		if !coreutils.IsTest() {
			kbSecret, err := st.KeyBuilder(state.AppSecretsStorage, appdef.NullQName)
			if err != nil {
				return err
			}
			kbSecret.PutString(state.Field_Secret, smtpCfg.PwdSecret)
			sv, err := st.MustExist(kbSecret)
			if err != nil {
				return err
			}
			pwd = sv.AsString("")
		}
		kb.PutString(state.Field_Password, pwd)

		_, err = intents.NewValue(kb)

		return
	}
}

func (r *ievResult) AsString(string) string {
	return r.verificationToken
}

func (r ivvtResult) AsString(string) string {
	return r.verifiedValueToken
}

// called at targetApp/targetWSID
func provideQryIssueVerifiedValueToken(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens, asp istructs.IAppStructsProvider) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQueryIssueVerifiedValueToken,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueTokenParams")).
			AddField(field_VerificationToken, appdef.DataKind_string, true).
			AddField(field_VerificationCode, appdef.DataKind_string, true).
			AddField(field_ForRegistry, appdef.DataKind_bool, false).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "IssueVerifiedValueTokenResult")).
			AddField(field_VerifiedValueToken, appdef.DataKind_string, true).(appdef.IDef).QName(),
		provideIVVTExec(itokens, cfg.Name, asp),
	))

	// code ok -> buckets state will be reset
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQueryIssueVerifiedValueToken, RateLimit_IssueVerifiedValueToken)
}

// q.sys.IssueVerifiedValueToken
// called at targetApp/profileWSID
// a helper is used for ResetPassword that calls `q.sys.IssueVerifiedValueToken` at the profile
func provideIVVTExec(itokens itokens.ITokens, appQName istructs.AppQName, asp istructs.IAppStructsProvider) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		verificationToken := args.ArgumentObject.AsString(field_VerificationToken)
		verificationCode := args.ArgumentObject.AsString(field_VerificationCode)
		forRegistry := args.ArgumentObject.AsBool(field_ForRegistry)

		as, err := asp.AppStructs(appQName)
		if err != nil {
			return err
		}

		appTokens := as.AppTokens()
		if forRegistry {
			asRegistry, err := asp.AppStructs(istructs.AppQName_sys_registry)
			if err != nil {
				// notest
				return err
			}
			appTokens = asRegistry.AppTokens()
		}

		verifiedValueToken, err := IssueVerfiedValueToken(verificationToken, verificationCode, appTokens, itokens)
		if err != nil {
			return err
		}

		// code ok -> reset per-profile rate limit
		appBuckets := istructsmem.IBucketsFromIAppStructs(as)
		rateLimitName := istructsmem.GetFunctionRateLimitName(QNameQueryIssueVerifiedValueToken, istructs.RateLimitKind_byWorkspace)
		appBuckets.ResetRateBuckets(rateLimitName, irates.BucketState{
			Period:             RateLimit_IssueVerifiedValueToken.Period,
			MaxTokensPerPeriod: irates.NumTokensType(RateLimit_IssueVerifiedValueToken.MaxAllowedPerDuration),
			TakenTokens:        0,
		})

		return callback(&ivvtResult{verifiedValueToken: verifiedValueToken})
	}
}

func provideCmdSendEmailVerificationCode(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandSendEmailVerificationCode,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "SendEmailVerificationParams")).
			AddField(field_VerificationCode, appdef.DataKind_string, true).
			AddField(Field_Email, appdef.DataKind_string, true).
			AddField(field_Reason, appdef.DataKind_string, true).
			AddField(field_Language, appdef.DataKind_string, false).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		istructsmem.NullCommandExec,
	))
}

func getVerificationEmailBody(federation coreutils.IFederation, verificationCode string, reason string, lng language.Tag, ctlg catalog.Catalog) string {
	text1 := message.NewPrinter(lng, message.Catalog(ctlg)).Sprintf(`Here is your verification code`)
	text2 := message.NewPrinter(lng, message.Catalog(ctlg)).Sprintf(`Please, enter this code on the`)
	text3 := message.NewPrinter(lng, message.Catalog(ctlg)).Sprintf(reason)
	return fmt.Sprintf(`
<div style="font-family: Arial, Helvetica, sans-serif;">
	<div
		style="margin: 20px auto 30px; width: 50%%; min-width: 200px; padding-bottom: 20px; border-bottom: 1px solid #ccc;text-align: center;">
	</div>

	<div style="text-align: center;">
		<p style="font-size: 24px; font-weight: 300">%s</p>
		<p style="font-size: 50px; font-weight: bold; text-align: center; letter-spacing: 10px; line-height: 50px; margin: 20px auto;">
			%s</p>
		<p>%s %s %s</p>
	</div>

	<div
		style="color: #989898; margin: 20px auto 30px; width: 50%%; min-width: 200px; padding-top: 20px; border-top: 1px solid #ccc;text-align: center;">
		%d &copy; unTill
	</div>
</div>
`, text1, verificationCode, text2, federation.URL().String(), text3, time.Now().Year())
}
