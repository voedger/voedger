[![codecov](https://codecov.io/gh/voedger/voedger/appdef/branch/main/graph/badge.svg?token=u6VrbqKtnn)](https://codecov.io/gh/voedger/voedger/appdef)

# Application Definition

## Restrictions

### Names
- Only letters (from `A` to `Z` and from `a` to `z`), digits (from `0` to `9`) and underscore symbol (`_`) are used.
- First symbol must be letter or underscore.
- Maximum length of name is 255.

Valid names examples:
```
  Foo
  bar
  FooBar
  foo_bar
  f007
  _f00_bar
```

Invalid names examples:
```
  Fo-o
  7bar
```

### Fields
- Maximum fields per definition is 65536.

### Containers
- Maximum containers per definition is 65536.

### Uniques
- Maximum fields per unique is 256
- Maximum uniques per definition is 100.
