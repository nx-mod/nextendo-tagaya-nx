# nextendo-tagaya-nx

Tagaya — the Nintendo Switch **title-version-list** service — for [Nextendo Network](https://nextendo.network). Source only. Not affiliated with Nintendo.

## Why

A console asks Tagaya (`tagaya.hac.lp1.eshop.nintendo.net/tagaya/hac_versionlist`) for the latest version of each title and the **minimum version online play requires**. On the Nextendo stack this falls through to Nintendo today; without a controlled answer a game can nag *"software update required"* or gate online on `required_version`. This serves a version list the stack owns.

## What it serves

```
GET /tagaya/hac_versionlist
  -> { "format_version": 1, "last_modified": <unix>, "titles": [ { "id", "version", "required_version" } ] }
```

The list comes from `versions.json` (read per request, so edits apply live). An `ETag` + `If-None-Match` give the console a cheap `304 Not Modified` when nothing changed. `required_version` defaults to `version` when omitted.

sni-router sends the Tagaya host here (TLS passthrough), or set `CERT_FILE`/`KEY_FILE` for direct TLS. Auth: the console sends `X-Nintendo-DenebEdgeToken` (from dauth); this server does not require it on a local stack.

## Run

```sh
go build -o server .        # Go 1.23+, stdlib only
go test ./...
./server
```

| setting | default | meaning |
|---|---|---|
| `TAGAYA_PORT` | 8471 | HTTP(S) port (behind sni-router) |
| `TAGAYA_VERSIONS` | versions.json | the title version list |
| `CERT_FILE`/`KEY_FILE` | — | direct TLS (else sni-router terminates) |
| `DASH_PORT`/`DASH_TOKEN` | 8100 | `/api/stats`, `/healthz` |

`versions.json`: `id` is the title id as a **decimal** integer.

## Credits

- **[Nextendo Network](https://nextendo.network)** — the stack (sni-router, dashboard) this plugs into.
- **[kinnay/NintendoClients](https://github.com/kinnay/NintendoClients/wiki/Tagaya-Server-(Switch))** — the Tagaya endpoint and response format.

Protocol facts were read and reimplemented; no code was copied.
