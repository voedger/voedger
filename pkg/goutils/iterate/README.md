Iterate - ForEach, FindFirst and FindFirstError generic iterators
=================================================================

Iterate package provides routines to iterate slice-like and map-like structures.

The main idea is the following. You have some interface or structure that has a simple method that iterates over all available values. Using the generic routines provided by this package, you can quickly implement data find and error find.

## Examples:

```go
type (
  ITested interface {
    Fields(enum func(name string))
  }
  testedStruct struct {
    fields []string
  }
)

func (s *testedStruct) Fields(enum func(name string)) {
  for _, name := range s.fields {
    enum(name)
  }
}
```

### Find:
```go
    var tested ITested = &testedStruct{fields: []string{"a", "b", "c"}}
    ok, data := FindFirstData(tested.Fields, "b")
    // ok = true
    // data = "b"

    // …

    ok, data := FindFirst(tested.Fields, func(data string) bool { return data > "a" })
    // ok = true
    // data = "b"
```

### Find first error:
```go
    var tested ITested = &testedStruct{fields: []string{"a", "b", "c"}}

    data, err := FindFirstError(tested.Fields, func(data string) error {
      if data == "b" {
        return fmt.Errorf("error at «%v»", data)
      }
      return nil
    })
    // err = `error at «b»`
    // data = "b"
```
