## Story
- As a Heeus app developer I want to declare fields like Email and phone numbers that must be verified
- As a Heeus app developer I want to verification be limited by tries amount or whatever to eliminate security holes

## Solution principles
- verifiable fields are verified by 6-digit code got by crypto-safe randomize algorhythm
- case with a link sent via email instead of code is bad because it could cause e.g. multiple payments after multiple opening the link
- deny Token usage in a wrong WSID
- Rate Limiter API is used to limit rates:
  - `q.sys.InitiateEmailVerification` calls - not often than 100 times per hour per workspace (profile)
  - `q.sys.IssueVerifiedValueToken` calls - not often than 3 times per hour per workspace (profile)
	- code verification passed -> counter is reset to zero
- `q.sys.InitiateEmailVerification` and `q.sys.IssuerVerifiedValueToken` are called at targetApp/profileWSID - to protect against unauthenticated users
  - to e.g. reset password these funcs should be called with sys auth via helper funcs like `q.sys.InitiateResetPasswordByEmail`

## Rates

```mermaid
    flowchart LR

	Token:::H
	IBucket:::S


	Profile[(Profile)]:::H
	Profile --- InitiateEmailVerification["q.sys.InitiateEmailVerification()"]:::S
		InitiateEmailVerification ---  bucketsIEV
	subgraph bucketsIEV["InitiateEmail buckets"]
		direction LR
		Bucket["Bucket[targetApp, 'InitiateEmailVerification', profileWSID]"]:::H
	end
	Profile --- IssueVerifiedValueToken["q.sys.IssueVerifiedValueToken(targetID)"]:::S
	IssueVerifiedValueToken --- bucketsIVVT

	subgraph bucketsIVVT["IssueVerifiedValueToken buckets"]
		direction LR
		Bucket3["Bucket[targetApp, 'IssueVerifiedValueToken', profileWSID]"]:::H
	end


	IssueVerifiedValueToken -.-> Token[VerifiedValueToken]


	TargetWS[(TargetWS)]:::H
	UsingToken:::B

	bucketsIVVT --- IBucket
	bucketsIEV --- IBucket

	Token -.-> UsingToken[Using VerifiedValueToken]

	UsingToken --- TargetWS

	classDef B fill:#FFFFB5
    classDef S fill:#B5FFFF
    classDef H fill:#C9E7B7
    classDef G fill:#FFFFFF,stroke:#000000, stroke-width:1px, stroke-dasharray: 5 5
```

## Limitations
- it is unclear how to control the rate per ID when a doc is created
- once obtained Verified Value could be used an ulimited number of times during the token validity time (10 minutes).
  - not a problem, ok to reset password for the login many times during 10 minutes

## Functional design
Declare a schema with a verified field:
```go
AppConfigType.AppDef.Add(QName, e.g. appdef.TypeKind_CDoc).
	AddVerifiedField(name, kind, false, e.g. appdef.VerificationKind_EMail)
```

Issue verification token and code:
```go
token, code, err := verifier.NewVerificationToken(entity, field, email, e.g. appdef.VerificationKind_EMail, targetWSID, ITokens, IAppTokens)
```

Issue verified value token:
```go
verifiedValueToken, err := verifier.IssueVerifiedValueToken(token, code)
```

## Technical design

```mermaid
sequenceDiagram
	participant c as Client
	participant p as targetApp/profileWSID
	participant tp as targetApp/targetWSID
	c->>p: q.sys.InitiateEmailVerification(entity, field, email, targetWSID) (auth profile user)
	activate p
		opt if IAppStructs.IsFunctionRateLimitExceeded
			p->>c: 429 to many requests
		end
		p->>p: generate token for targetWSID and code
		p->>p: c.sys.SendEmailVerificationCode() (sys auth)
		p->>c: token
	deactivate p

	note over p: Projector<A, SendEmailVerificationCodeProjector>
	p->>p: send email

	c->>p: q.sys.IssueVerifiedValueToken(token, code) (auth null)
	activate p
		opt if IAppStructs.IsFunctionRateLimitExceeded
			p->>c: 429 too many requests
		end
		p->>p: decrypt token
		p->>p: check code
		alt code is ok
			p->>p: reset rate limit
			p->>c: 200 ok + verifiedValueToken
		else code is wrong
			p->>c: 400 bad request
		end
	deactivate p

	c->>tp: use VerifiedValueToken (e.g. c.sys.ResetPassword(email: verifiedValueToken))
	activate tp
		tp->>tp: decrypt token
		opt wrong wsid
			tp->>c: error
		end
		tp->>tp: set the value from the token to the target field
	deactivate tp
```
### Verifiable fields in application schema
```mermaid
erDiagram

appdef_Schema ||..|| ResetPasswordByEmailUnloggedParams: "describes e.g."
appdef_Schema ||..|| CDocAirUserProfile: "describes e.g."

ResetPasswordByEmailUnloggedParams ||--|| authnz_Email: "has field"
air_Email ||..|| VerifiableField: is
authnz_Email ||..|| VerifiableField: is
CDocAirUserProfile ||--|| air_Email: "has field"
ResetPasswordByEmailUnloggedParams ||..|| cmd_ResetPasswordByEmail: "is an arg of"
```

### Using Verified Value Token to set the value of the Verifiable field
```mermaid
flowchart LR

verifiedValueTokenInCUD_request -->|decrypt to| VerifiedValuePayload
VerifiedValuePayload --> Entity_QName
VerifiedValuePayload --> Field
VerifiedValuePayload --> Value
Entity_QName -->|check| VerifiableField
Field -->|check| VerifiableField
Value -->|assign| VerifiableField
```

```mermaid
erDiagram

VerifiedValuePayload ||--|| VerificationKind: has
VerifiedValuePayload ||--|| Entity: has
VerifiedValuePayload ||--|| Field: has
VerifiedValuePayload ||--|| Value: has

VerificationKind ||..|| kind: "checked for equality to"
Value ||..|| value: "assigned to"
Entity ||..|| QName: "checked for equality to"
Field ||..|| name: "checked for equality to"


value ||--|| VerifiedField: "is part of"
name ||--|| VerifiedField: "is part of"
kind ||--|| VerifiedField: "is part of"

VerifiedField }|--|| appdef_Schema: "belongs to"
QName ||..|| appdef_Schema: "refs"

appdef_Schema ||..|| ResetPasswordByEmailUnloggedParams: "e.g."
appdef_Schema ||..|| CDocAirUserProfile: "e.g."

ResetPasswordByEmailUnloggedParams ||--|| authnz_Email: "has field"
VerifiedField ||..|| air_Email: "can be"
VerifiedField ||..|| authnz_Email: "can be"
CDocAirUserProfile ||--|| air_Email: "has field"
ResetPasswordByEmailUnloggedParams ||..|| cmd_ResetPasswordByEmail: "is an arg of"
```
