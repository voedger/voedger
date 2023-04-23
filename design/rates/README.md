# Motivation

[Verifiable Fields with Rate Limits](https://dev.heeus.io/launchpad/#!24713)

# Functional design
Declare func with rate limit:
```go
AppConfig.FunctionRateLimits.AddAppLimit(
	QName("sys.InitiateEmailVerification"), istructs.RateLimit{
		Period:                24*time.Hour,
		MaxAllowedPerDuration: 3,
	}
)
AppConfig.FunctionRateLimits.AddWorkspaceLimit(
	QName("sys.InitiateEmailVerification"), istructs.RateLimit{
		Period:                time.Hour,
		MaxAllowedPerDuration: 3,
	}
)
```

Check rate limit:
```go
if IAppStructs.IsFunctionRateLimitsExceeded(funcQName, WSID) {
	return utils.NewHTTPErrorf(http.StatusTooManyRequests)
}
```

# Technical design
```mermaid
erDiagram

IAppStructs ||--|| appConfig: "has internal"
IAppStructs ||--|| appStructs: "implemented by"
appConfig ||--|| FunctionRateLimits: "has object"
FunctionRateLimits ||--|| AddAppLimit: "has method"
FunctionRateLimits ||--|| AddWorkspaceLimit: "has method"
AddAppLimit ||..|| rateLimits: "writes to"
AddWorkspaceLimit ||..|| rateLimits: "writes to"
appStructs ||--|| IBuckets: has
FunctionRateLimits ||--|| rateLimits: "has map funcQName->istructs.RateLimit"
IBuckets ||--|| TakeTokens: "has method"
IAppStructs ||--|| IsFunctionRateLimitsExceeded: "has method"
rateLimits ||..|| IsFunctionRateLimitsExceeded: "used by"
IsFunctionRateLimitsExceeded ||..|| TakeTokens: calls
TakeTokens ||--|| bool: "has result"
IsFunctionRateLimitsExceeded ||--|| bool: returns
```
# Limitations
- IBuckets are per-app for now
