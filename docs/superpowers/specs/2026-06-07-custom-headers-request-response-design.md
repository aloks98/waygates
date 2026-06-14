# Typed Request/Response Custom Headers — Design

**Date:** 2026-06-07
**Status:** Approved (design)
**Author:** Alok Kumar Sahoo (with Claude)

## Problem

Waygates lets users set `custom_headers` on a `reverse_proxy`. The proxy traffic
E2E suite revealed that these are applied **only as request headers to the
upstream** (`backend/internal/caddy/config/http_builder.go:206-212` calls
`handler.Headers.Request.SetHeader`). There is no way to set **response** headers
returned to the client — even though the supporting type
`HeadersConfig.Response *HeaderOps` already exists
(`backend/internal/caddy/config/http_handlers.go:130`). Common needs like adding
security headers (HSTS, `X-Frame-Options`) to responses are impossible.

The `custom_headers` E2E scenario is currently `t.Skip`-ped pending this feature
(`backend/tests/integration/traffic_l7_test.go`).

## Goal

Support custom headers in **both directions** — request (to upstream) and
response (to client) — across the backend (model, config builder, API) and the
frontend (types, form), with full backward compatibility for the existing flat
shape. Then un-skip the E2E scenario and assert both directions.

### Non-goals

- Per-header operations beyond set (no add/delete/replace semantics). Matches
  today's "set" behavior.
- Header value validation/limits beyond what exists today (none specific).
- Changes to L4 proxies (no header concept there).

## Current state (verified)

- `models.Proxy.CustomHeaders` is `JSONField` (`= map[string]interface{}`),
  stored as JSON text via GORM (`type:text`). `JSONField` is shared by
  `load_balancing`, `redirect`, `static` too, so it must NOT be changed in place
  — a dedicated type is introduced for `custom_headers` only.
- Builder applies only request headers (`http_builder.go:206-212`).
- `HeadersConfig{ Request *HeaderOps; Response *HeaderOps }`
  (`http_handlers.go:128-131`); `HeaderOps.SetHeader(name, values...)`
  (`http_handlers.go:484`) — response support exists, just unused.
- Frontend type: `custom_headers?: Record<string, string>` in
  `ui/src/types/proxy.ts` (3 occurrences: `Proxy`, create, update).

## Backward-compatibility decision

**Flexible parse, normalize on read** (chosen):

- **Write:** accept BOTH the old flat map (`{"X-A":"1"}`) and the new nested
  object (`{"request":{...},"response":{...}}`). A flat map is interpreted as
  **request** headers (preserving today's behavior).
- **Stored data:** old rows remain flat in the DB; a flexible `Scan` loads them
  transparently. **No DB migration.**
- **Read:** the API always returns the **normalized nested shape**, so the UI and
  any consumer get one consistent structure. (This changes the `custom_headers`
  field shape in proxy API responses; acceptable — the UI is the primary
  consumer and is updated in lockstep.)

## Architecture

### 1. Backend data model — `CustomHeaders` type

New type in `backend/internal/models/proxy.go` (replacing `JSONField` for the
`custom_headers` field only):

```go
type CustomHeaders struct {
    Request  map[string]string `json:"request,omitempty"`
    Response map[string]string `json:"response,omitempty"`
}
```

Methods:

- **`UnmarshalJSON([]byte) error` (flexible).** Decode into
  `map[string]json.RawMessage`. The shape is **nested** iff every key is in
  `{"request","response"}` AND every value is a JSON object; otherwise it is
  **flat** and the whole object is decoded as `map[string]string` into `Request`.
  This is unambiguous because HTTP header values are always strings: a flat
  header literally named `request` has a string value (not an object) and is
  therefore correctly treated as flat. `null`/empty → zero value.
- **`MarshalJSON() ([]byte, error)` (normalize).** Always emit the nested shape
  `{"request":{...},"response":{...}}` (omitting empty sub-maps via `omitempty`).
- **`Value() (driver.Value, error)`** — marshal to JSON text (nested) for
  storage; return `nil` (SQL NULL) when both maps are empty, mirroring
  `JSONField` behavior.
- **`Scan(src interface{}) error`** — read JSON text/bytes from the DB and run
  the flexible `UnmarshalJSON`, so legacy flat rows load correctly.

`Proxy.CustomHeaders` field type changes from `JSONField` to `CustomHeaders`
(tag stays `json:"custom_headers,omitempty" gorm:"type:text"`).

### 2. Backend config builder

In `backend/internal/caddy/config/http_builder.go` (the `custom_headers` block
around line 206), replace the request-only loop with:

- For each `proxy.CustomHeaders.Request` entry → `handler.Headers.Request.SetHeader(k, v)`
  (preserving the existing `Headers.Request` that also holds standard proxy headers).
- If `proxy.CustomHeaders.Response` is non-empty → ensure `handler.Headers.Response`
  is a `*HeaderOps` and `SetHeader(k, v)` each entry.

No new builder types are needed (`HeadersConfig.Response` and `HeaderOps.SetHeader`
already exist).

### 3. Backend API

The create/update proxy request path must accept the flexible shape. Implementation
confirms whether the handler decodes into `models.Proxy` directly or via a DTO in
`backend/internal/validation/` (the L7 proxy validation file); the
`custom_headers` field in whichever type is decoded must be the `CustomHeaders`
type (or its flexible unmarshal). Read responses return the normalized nested
shape automatically via `CustomHeaders.MarshalJSON`. Validation stays as today.

### 4. Frontend

- **`ui/src/types/proxy.ts`** (3 spots): change `custom_headers?: Record<string,string>`
  to:
  ```ts
  custom_headers?: { request?: Record<string, string>; response?: Record<string, string> };
  ```
- **`ui/src/components/proxy/forms/reverse-proxy-form.tsx`:** the existing custom
  headers section becomes two labeled key-value editors — **"Request headers (sent
  to upstream)"** and **"Response headers (sent to client)"** — each reusing the
  current add/remove key-value editor pattern. On load, populate both sub-maps
  from the (now nested) API value; old proxies populate request only. On submit,
  emit the nested shape (omit empty sub-maps).
- **`ui/src/lib/form-validation.ts`:** update the `custom_headers` zod schema to the
  nested shape (both sub-objects optional, string→string records).
- If a proxy detail/read-only view renders custom headers, update it to show both
  groups.

### 5. Tests

- **Model unit (`models` package):** `UnmarshalJSON` flat→`Request`;
  nested→both; a header named `request` with a string value→treated as flat;
  `MarshalJSON` always nested; `Scan`/`Value` round-trip including a legacy flat
  text row.
- **Builder unit (`caddy/config` package):** request headers applied to
  `Headers.Request`; response headers applied to `Headers.Response`; update the
  existing flat-shape custom-header builder test (`builder_test.go` ~line 537) to
  assert the back-compat (flat→request) path.
- **Traffic E2E (`tests/integration/traffic_l7_test.go`):** remove the `t.Skip`
  from the `custom_headers` subtest; create a reverse_proxy with BOTH a request
  and a response custom header; assert (a) the proxied **response** carries the
  response header (`resp.Header.Get`), and (b) the mendhak echo's reflected
  request headers (in its JSON body) contain the **request** header.
- **Frontend:** typecheck/build clean; add a form unit test if the UI test setup
  supports it (otherwise rely on build + the E2E coverage of the wire shape).

## Files

Backend:
- `backend/internal/models/proxy.go` — `CustomHeaders` type + field change.
- `backend/internal/caddy/config/http_builder.go` — apply request + response.
- `backend/internal/validation/<l7 proxy DTO>.go` — DTO field (if a separate DTO
  exists).
- `backend/internal/models/proxy_test.go`, `backend/internal/caddy/config/builder_test.go`
  (and/or `http_builder_test.go`) — unit tests.
- `backend/tests/integration/traffic_l7_test.go` — un-skip + assert both directions.

Frontend:
- `ui/src/types/proxy.ts`
- `ui/src/components/proxy/forms/reverse-proxy-form.tsx`
- `ui/src/lib/form-validation.ts`
- proxy detail view, if it renders custom headers.

## Risks & mitigations

- **Flat/nested ambiguity:** resolved by the "values are objects ⇒ nested" rule,
  which is unambiguous because header values are always strings.
- **API response shape change:** the UI is updated in lockstep; documented here.
- **Existing data:** flexible `Scan` handles legacy flat rows; no migration.
- **Builder regression:** existing request-header behavior preserved (flat→request),
  covered by the updated builder unit test and the traffic E2E.

## Success criteria

- A reverse_proxy can set request and response custom headers via API and UI.
- Old flat `custom_headers` (stored rows and API input) keep working as request
  headers.
- `make test-traffic` passes with the `custom_headers` subtest un-skipped and
  asserting both directions.
- Backend unit suite and UI build/lint are green.
