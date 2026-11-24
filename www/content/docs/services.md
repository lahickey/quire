---
title: Services
summary: What runs while a site is served and that features call - a store, sessions, and services of your own.
---

A service is something features call, and puts nothing at an address: a store, sessions, a mailer.
It is a plain value. The site makes it, adds it, and hands it to whatever uses it.

```go
func main() {
  carts, visitors := store.NewMemory(), sessions.New()
  quire.Run(identity, site.FeatureFunc(func(s *site.Site) error {
    s.AddServices(carts, visitors)
    return s.Include(basket.New(carts, visitors, "/basket", items...))
  }))
}
```

```go
type Service interface {
  Start(context.Context, model.Env) error
  Stop(context.Context) error
}
```

- **A constructor only keeps what it is given.** Connections and goroutines are opened in `Start`. A bake starts nothing, so a service does nothing in one.
- **Every service is added**, with `AddServices`, whether or not it has anything to start yet. One that grows a connection later then changes no site.
- **They start in the order added and stop in reverse**, when the site is served. If one will not start, those already started are stopped and the site does not serve.
- **`Start` is told the site's environment**, once: `model.Env` is its identity, whether it is in dev, and its logger. A service settles there what is fixed for the life of the site, and never works it out again for a request.
- **A feature takes a service as an argument**, typed by what it needs, so a missing one does not compile and a test passes its own.

## A store

`service/store` keeps bytes by key. What the bytes are is the feature's business: the basket keeps a cart as JSON under the visitor's id.

```go
type Store interface {
  Get(ctx context.Context, key string) ([]byte, bool, error)
  Put(ctx context.Context, key string, value []byte) error
  Delete(ctx context.Context, key string) error
  List(ctx context.Context, prefix string) iter.Seq2[string, []byte]
}
```

- `store.NewMemory()` is one that a restart empties. It keeps 8MB of keys and values, and when it is full it drops what was written longest ago, so nobody can fill a server's memory through it.
- `store.Under(st, "basket")` is a store's keys beneath a name. A feature puts its own name there, so every feature can be handed the same store and none can read another's.

What has to ask questions of its data brings a database of its own, as a service.

## Sessions

`service/sessions` is who a visitor is, by a cookie.

- `visitors.ID(w, r)` is the visitor's id, given to them now if they had none. Ask before the response is begun.
- `visitors.Read(r)` is their id if they have one, for a page that only shows what they hold.

A visitor is given the `qsid` cookie the first time something asks who they are, so a site where nothing asks sets none.
It is kept to HTTPS when the site's address is `https` and it is not in dev, which the service settles when it starts; `sessions.New(sessions.Secure(false))` says otherwise.
Two cold tabs can race; last cookie wins. That is inherent to HTTP.
