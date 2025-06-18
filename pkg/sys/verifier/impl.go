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
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

var translationsCatalog = coreutils.GetCatalogFromTranslations(translations)

// called at targetApp/profileWSID
func provideQryInitiateEmailVerification(sr istructsmem.IStatelessResources, itokens itokens.ITokens,
	asp istructs.IAppStructsProvider, federation federation.IFederation) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		QNameQueryInitiateEmailVerification,
		provideIEVExec(itokens, federation, asp),
	))
}

// q.sys.InitiateEmailVerification
// called at targetApp/profileWSID
func provideIEVExec(itokens itokens.ITokens, federation federation.IFederation, asp istructs.IAppStructsProvider) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		entity := args.ArgumentObject.AsString(field_Entity)
		targetWSID := istructs.WSID(args.ArgumentObject.AsInt64(field_TargetWSID)) // nolint G115
		field := args.ArgumentObject.AsString(field_Field)
		email := args.ArgumentObject.AsString(Field_Email)
		forRegistry := args.ArgumentObject.AsBool(field_ForRegistry)
		lng := args.ArgumentObject.AsString(field_Language)

		as := args.State.AppStructs()
		appTokens := as.AppTokens()
		if forRegistry {
			// issue token for sys/registry/pseduoWSID. That's for c.sys.ResetPassword only for now
			asRegistry, err := asp.BuiltIn(istructs.AppQName_sys_registry)
			if err != nil {
				// notest
				return err
			}
			appTokens = asRegistry.AppTokens()
			targetWSID = coreutils.GetPseudoWSID(istructs.NullWSID, email, istructs.CurrentClusterID())
		}

		verificationToken, verificationCode, err := NewVerificationToken(entity, field, email, appdef.VerificationKind_EMail, targetWSID, itokens, appTokens)
		if err != nil {
			return err
		}

		systemPrincipalToken, err := payloads.GetSystemPrincipalToken(itokens, as.AppQName())
		if err != nil {
			return err
		}

		// c.sys.SendEmailVerificationCode
		body := fmt.Sprintf(`{"args":{"VerificationCode":"%s","Email":"%s","Reason":"%s","Language":"%s"}}`, verificationCode, email, verifyEmailReason, lng)
		if _, err = federation.Func(fmt.Sprintf("api/%s/%d/c.sys.SendEmailVerificationCode", as.AppQName(), args.WSID), body,
			coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(systemPrincipalToken)); err != nil {
			return fmt.Errorf("c.sys.SendEmailVerificationCode failed: %w", err)
		}

		return callback(&ievResult{verificationToken: verificationToken})
	}
}

func applySendEmailVerificationCode(federation federation.IFederation, smtpCfg smtp.Cfg) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		eventTime := time.UnixMilli(int64(event.RegisteredAt()))
		if eventTime.Add(threeDays).Before(time.Now()) {
			// skip old emails to prevent re-sending after projector rename
			// see https://github.com/voedger/voedger/issues/275
			return nil
		}
		lng := event.ArgumentObject().AsString(field_Language)

		kb, err := st.KeyBuilder(sys.Storage_SendMail, appdef.NullQName)
		if err != nil {
			return
		}
		reason := event.ArgumentObject().AsString(field_Reason)
		translatedEmailSubject := message.NewPrinter(language.Make(lng), message.Catalog(translationsCatalog)).Sprintf(EmailSubject)
		kb.PutString(sys.Storage_SendMail_Field_Subject, translatedEmailSubject)
		kb.PutString(sys.Storage_SendMail_Field_To, event.ArgumentObject().AsString(Field_Email))
		kb.PutString(sys.Storage_SendMail_Field_Body, getVerificationEmailBody(federation, event.ArgumentObject().AsString(field_VerificationCode), reason, language.Make(lng), translationsCatalog))
		kb.PutString(sys.Storage_SendMail_Field_From, smtpCfg.GetFrom())
		kb.PutString(sys.Storage_SendMail_Field_Host, smtpCfg.Host)
		kb.PutInt32(sys.Storage_SendMail_Field_Port, smtpCfg.Port)
		kb.PutString(sys.Storage_SendMail_Field_Username, smtpCfg.Username)
		kbSecret, err := st.KeyBuilder(sys.Storage_AppSecret, appdef.NullQName)
		if err != nil {
			return err
		}
		kbSecret.PutString(sys.Storage_AppSecretField_Secret, smtpCfg.PwdSecret)
		sv, err := st.MustExist(kbSecret)
		if err != nil {
			return err
		}
		pwd := sv.AsString("")
		kb.PutString(sys.Storage_SendMail_Field_Password, pwd)

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
func provideQryIssueVerifiedValueToken(sr istructsmem.IStatelessResources, itokens itokens.ITokens, asp istructs.IAppStructsProvider) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		QNameQueryIssueVerifiedValueToken,
		provideIVVTExec(itokens, asp),
	))
}

// q.sys.IssueVerifiedValueToken
// called at targetApp/profileWSID
// a helper is used for ResetPassword that calls `q.sys.IssueVerifiedValueToken` at the profile
func provideIVVTExec(itokens itokens.ITokens, asp istructs.IAppStructsProvider) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		verificationToken := args.ArgumentObject.AsString(field_VerificationToken)
		verificationCode := args.ArgumentObject.AsString(field_VerificationCode)
		forRegistry := args.ArgumentObject.AsBool(field_ForRegistry)

		as := args.State.AppStructs()

		appTokens := as.AppTokens()
		if forRegistry {
			asRegistry, err := asp.BuiltIn(istructs.AppQName_sys_registry)
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
			MaxTokensPerPeriod: RateLimit_IssueVerifiedValueToken.MaxAllowedPerDuration,
			TakenTokens:        0,
		})

		return callback(&ivvtResult{verifiedValueToken: verifiedValueToken})
	}
}

func provideCmdSendEmailVerificationCode(sr istructsmem.IStatelessResources) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		QNameCommandSendEmailVerificationCode,
		execCmdSendEmailVerificationCode,
	))
}

func execCmdSendEmailVerificationCode(args istructs.ExecCommandArgs) (err error) {
	email := args.ArgumentObject.AsString(Field_Email)
	return coreutils.ValidateEMail(email)
}

func getVerificationEmailBody(federation federation.IFederation, verificationCode string, reason string, lng language.Tag, ctlg catalog.Catalog) string {
	text1 := message.NewPrinter(lng, message.Catalog(ctlg)).Sprintf(`Here is your verification code`)
	text2 := message.NewPrinter(lng, message.Catalog(ctlg)).Sprintf(`Please, enter this code on`)
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
`, text1, verificationCode, text2, federation.URLStr(), text3, time.Now().Year())
}
