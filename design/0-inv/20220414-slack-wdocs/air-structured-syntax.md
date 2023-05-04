# Workspaces

```javascript
class Channel extends sys.Workspace {

  // WDoc
  bills1: []pbill
  bills2: []pbill

  // CDoc
  articles: []article
  currencies: []currency
}

```


# Structs

```javascript

struct WDoc{}

@DenyReference(WDoc)
struct CDoc{}


class currency extends CDoc{
  // bill_id: Ref<pbill> // validation error since CDoc may not refer WDoc
}

```