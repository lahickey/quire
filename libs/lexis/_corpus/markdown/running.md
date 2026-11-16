---
title: Running the ledger service
summary: Install it, configure it, and keep it up.
tags: [ops, ledger]
draft: false
---

# Running the ledger service

This is the short version. The [design notes](./design.md) say *why* it is shaped
this way; this page says **what to type**.

> If you only want to try it, `docker run -p 8080:8080 ledgerd` is enough.
> Come back here when you need it to stay up.

## Install

Pick one:

1. A release binary from the [releases page](https://example.com/acme/ledgerd/releases).
2. `go install example.com/acme/ledgerd/cmd/ledgerd@latest`
3. The container image, `ghcr.io/acme/ledgerd`.

Then check it runs:

```sh
ledgerd --version
ledgerd migrate --config /etc/ledgerd/config.toml  # creates the tables
```

### Requirements

- Linux or macOS, `amd64` or `arm64`
- PostgreSQL 15 or newer
  - with the `pgcrypto` extension
  - and a database it can create tables in
- About 60 MB of memory at rest

* * *

## Configure

Configuration is one TOML file. The smallest that works:

```toml
[server]
addr = ":8080"

[database]
url = "postgres://ledger@db.internal/ledger"
pool_size = 20
```

Every key can also be set from the environment: `server.addr` is `LEDGER_SERVER_ADDR`.
The environment wins, so a secret need not be in the file.

| Key | Default | Does |
|-|-|-|
| `server.addr` | `:8080` | Where it listens |
| `database.pool_size` | `10` | Connections kept open |
| `log.format` | `text` | `text` or `json` |

## Posting an entry

An entry moves one account by an amount in pence. From Go:

```go
l := ledger.New()
err := l.Post(ctx, ledger.Entry{Account: "acme", Kind: ledger.Credit, Pence: 12_50})
if errors.Is(err, ledger.ErrOverdrawn) {
    log.Printf("refused: %v", err)
}
```

Or over HTTP, which takes JSON:

```json
{ "account": "acme", "kind": "credit", "pence": 1250, "note": "invoice 88" }
```

A debit that would take the balance below zero is refused with `409 Conflict`.
~~It used to go through.~~

## Keeping it up

__Health.__ `GET /healthz` answers `200` when the database is reachable, and
`503` when it is not. Point your load balancer at it.

_Backups._ The service holds no state of its own; back up the database.
A nightly `pg_dump` is enough for most:

~~~yaml
schedule: "15 3 * * *"
command: pg_dump --format=custom ledger > /backups/ledger.dump
keep: 14
~~~

### When it will not start

- [ ] Is the config file readable by the `ledger` user?
- [ ] Does `ledgerd migrate` succeed on its own?
- [x] Is port 8080 free? <kbd>ss -ltn</kbd> will say.

Still stuck? The log line to look for is `listening on`: everything before it
is start-up, and an error there names the setting at fault.[^1]

![The request path, from load balancer to database](./request-path.svg "Request path")

Write to <ops@example.com>, or see <https://example.com/acme/ledgerd/issues>.

[^1]: With `log.format = "json"` it is the `msg` field.

[design]: ./design.md "Design notes"
