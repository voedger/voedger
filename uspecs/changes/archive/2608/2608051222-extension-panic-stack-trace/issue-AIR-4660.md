# extengine: provide the stack on extension panic

- URL: https://untill.atlassian.net/browse/AIR-4660
- ID: AIR-4660
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

panic in extension → have no idea where it happened

[example](https://untillmon.grafana.net/a/grafana-lokiexplore-app/explore/app/airs-bp/logs?from=2026-08-05T08%3A34%3A59.502Z&to=2026-08-05T08%3A35%3A00.502Z&var-ds=fejlmj7udkv7ka&var-filters=app%7C%3D%7Cairs-bp&var-filters=cluster%7C%3D%7Cair-dev&patterns=%5B%5D&var-lineFormat=&var-jsonFields=&var-patterns=&var-lineFilterV2=&var-lineFilters=&displayedFields=%5B%22time%22%2C%22reqid%22%2C%22vapp%22%2C%22wsid%22%2C%22stage%22%2C%22extension%22%2C%22msg%22%2C%22woffset%22%2C%22poffset%22%5D&visualizationType=%22table%22&timezone=browser&var-all-fields=&userDisplayedFields=true&prettifyLogMessage=false&sortOrder=%22Descending%22&wrapLogMessage=false&panelState=%7B%22logs%22%3A%7B%22displayedFields%22%3A%5B%22time%22%2C%22reqid%22%2C%22vapp%22%2C%22wsid%22%2C%22stage%22%2C%22extension%22%2C%22msg%22%2C%22woffset%22%2C%22poffset%22%5D%2C%22id%22%3A%221785918900002307815_42ed2b8f%22%2C%22sortOrder%22%3A%22Descending%22%7D%7D&var-levels=detected_level%7C%3D%7Cerror&var-fields=extension%7C%3D%7C__CV%CE%A9__%7B%22value%22%3A%22job.air.RefreshTranslations%22__gfc__%22parser%22%3A%22mixed%22%7D%2Cjob.air.RefreshTranslations&var-fields=msg%7C%3D%7C__CV%CE%A9__%7B%22value%22%3A%22extension+github.com%2Funtillpro%2Fairs-bp3%2Fpackages%2Fair.RefreshTranslations+panic%3A+runtime+error%3A+invalid+memory+address+or+nil+pointer+dereference%22__gfc__%22parser%22%3A%22mixed%22%7D%2Cextension+github.com%2Funtillpro%2Fairs-bp3%2Fpackages%2Fair.RefreshTranslations+panic%3A+runtime+error%3A+invalid+memory+address+or+nil+pointer+dereference)

```
extension github.com/untillpro/airs-bp3/packages/air.RefreshTranslations panic: runtime error: invalid memory address or nil pointer dereference
```

## What

include stack trace in the log message on panic
