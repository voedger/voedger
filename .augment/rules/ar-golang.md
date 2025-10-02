---
type: "always_apply"
---

if you writting a program on golang then follow these rules:

- Use go 1.24 and above features
- Use `for <idx> = range(<slice>)` whenever appropriate (do not use for <idx> := 0; idx <... ; idx ++)
- in unit tests use require := require.New(t) and then use e.g. require.Equal(1, 1) instead of require.Equal(t, 1, 1)
- in unit tests avoid constructions like if a != b { t.Fatal() }. Use require.Equal(a,b) instead
- in unit tests if you are modifying any global state then save the initial state, then modify, then revert changes using defer func
- in unit tests if the test consits of few parts that are not impact on others then implement subtests with t.Run(description, func(t *testing.T()} instead of setting description as a comment
- write as short a code as possible
- avoid coments. Produce the code so that it is clear without comments what it does
