---
---

```go
package main

import (
  "github.com/lahickey/quire"
  "github.com/lahickey/quire/model"
)

func main() {
  quire.Run(
    model.Site{Name: "Example", BaseURL: "https://example.com"},
    func(e *quire.Engine) error {
      _, err := e.AddCollection(model.Basic{ID: "posts"})
      return err
    })
}
```
