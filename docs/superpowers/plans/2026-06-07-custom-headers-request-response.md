# Typed Request/Response Custom Headers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a reverse_proxy set custom headers in both directions — request (to upstream) and response (to client) — across backend model, config builder, API, and UI, with full backward compatibility for the existing flat shape.

**Architecture:** A dedicated `CustomHeaders` Go type (replacing `JSONField` for the `custom_headers` field only) with a flexible `UnmarshalJSON`/`Scan` that accepts both the legacy flat map (→ request headers) and the new nested `{request,response}` shape, and serializes back as nested (normalize on read, no DB migration). The builder applies request headers to `Headers.Request` and response headers to `Headers.Response` (both types already exist). The UI adds two key-value editors to the reverse-proxy form.

**Tech Stack:** Go (GORM, encoding/json, database/sql/driver), Caddy JSON config, React + TanStack Form + Zod, testcontainers (traffic E2E).

**Reference spec:** `docs/superpowers/specs/2026-06-07-custom-headers-request-response-design.md`

---

## Background the implementer needs

**Verified facts:**
- `models.Proxy.CustomHeaders` is currently `JSONField` (`= map[string]interface{}`), `json:"custom_headers,omitempty" gorm:"type:text"` (`backend/internal/models/proxy.go:29`). `JSONField` (defined `proxy.go:42-` with `Value`/`Scan`/`MarshalJSON`/`UnmarshalJSON`) is shared by `load_balancing`, `redirect`, `static` — **do not modify `JSONField`**; introduce a new type for `custom_headers` only.
- The L7 proxy API has **no DTO**: `createProxyRequest`/`updateProxyRequest` embed `models.Proxy` and decode the body directly into it (`backend/internal/api/handlers/proxy.go:200,222,271,291`). So changing the model field type covers create + update automatically.
- The Caddy builder applies only request headers today (`backend/internal/caddy/config/http_builder.go:205-213`). `HeadersConfig{Request, Response *HeaderOps}` (`http_handlers.go:128-131`) and `HeaderOps.SetHeader(name, values...)` (`http_handlers.go:484`) already exist — response support is unused, not missing.
- UI: `custom_headers?: Record<string,string>` appears at `ui/src/types/proxy.ts:57` (`ProxyConfig`), `:74` (`CreateReverseProxyRequest`), `:112` (`UpdateProxyRequest`). The reverse-proxy **form does not currently expose a custom-headers editor** (this plan adds one). `ui/src/routes/_dashboard/proxies/$proxyId.tsx:65-67` has a change-detection comparison of `custom_headers`. There is no separate proxy zod schema in `lib/form-validation.ts`; the form's schema (`reverseProxySchema`) is inline in `reverse-proxy-form.tsx`.
- The form manages array fields via local `useState` + `form.setFieldValue(...)` (the upstreams pattern, `reverse-proxy-form.tsx:98,192-213`). We mirror that for headers.
- Traffic E2E: the `custom_headers` subtest in `backend/tests/integration/traffic_l7_test.go` is `t.Skip`-ped pending this feature. Backends are `mendhak/http-https-echo` (alias `echo1`/`echo2`), which reflect request headers in their JSON response under a lowercased `"headers"` object.

**Build/test commands** (module root is the repo root):
- Backend unit: `go test ./backend/internal/... -count=1`
- Traffic E2E (needs rebuilt image after backend changes): `make test-traffic` or `docker build -t waygates-test:latest . && go test -tags traffic -run 'TestTraffic_L7/custom_headers' ./backend/tests/integration/ -count=1 -v`
- UI: `pnpm --dir ui build` and `make lint-ui`

---

## File structure

- `backend/internal/models/proxy.go` — new `CustomHeaders` type; change `Proxy.CustomHeaders` field type.
- `backend/internal/models/proxy_test.go` — unit tests for `CustomHeaders` (create if absent; otherwise append).
- `backend/internal/caddy/config/http_builder.go` — apply request + response headers.
- `backend/internal/caddy/config/builder_test.go` — update the flat-shape test; add a response-header test.
- `backend/tests/integration/traffic_l7_test.go` — un-skip + assert both directions.
- `ui/src/types/proxy.ts` — `CustomHeaders` interface + 3 field replacements.
- `ui/src/components/proxy/forms/reverse-proxy-form.tsx` — request/response header editors.
- `ui/src/routes/_dashboard/proxies/$proxyId.tsx` — change-detection for nested shape.

---

## Task 1: Backend `CustomHeaders` model type

**Files:**
- Modify: `backend/internal/models/proxy.go`
- Test: `backend/internal/models/proxy_test.go`

- [ ] **Step 1: Write failing unit tests**

Append to `backend/internal/models/proxy_test.go` (create the file with `package models` + imports `encoding/json`, `testing`, and `github.com/stretchr/testify/assert`/`require` if it does not exist):

```go
func TestCustomHeaders_UnmarshalJSON_FlatLegacy(t *testing.T) {
	var c CustomHeaders
	require.NoError(t, json.Unmarshal([]byte(`{"X-A":"1","X-B":"2"}`), &c))
	assert.Equal(t, map[string]string{"X-A": "1", "X-B": "2"}, c.Request)
	assert.Empty(t, c.Response)
}

func TestCustomHeaders_UnmarshalJSON_Nested(t *testing.T) {
	var c CustomHeaders
	require.NoError(t, json.Unmarshal([]byte(`{"request":{"X-Req":"r"},"response":{"X-Res":"s"}}`), &c))
	assert.Equal(t, map[string]string{"X-Req": "r"}, c.Request)
	assert.Equal(t, map[string]string{"X-Res": "s"}, c.Response)
}

func TestCustomHeaders_UnmarshalJSON_HeaderNamedRequest_IsFlat(t *testing.T) {
	// A flat header literally named "request" has a STRING value -> flat shape.
	var c CustomHeaders
	require.NoError(t, json.Unmarshal([]byte(`{"request":"some-value"}`), &c))
	assert.Equal(t, map[string]string{"request": "some-value"}, c.Request)
	assert.Empty(t, c.Response)
}

func TestCustomHeaders_UnmarshalJSON_Null(t *testing.T) {
	var c CustomHeaders
	require.NoError(t, json.Unmarshal([]byte(`null`), &c))
	assert.True(t, c.IsEmpty())
}

func TestCustomHeaders_MarshalJSON_AlwaysNested(t *testing.T) {
	c := CustomHeaders{Request: map[string]string{"X-A": "1"}}
	out, err := json.Marshal(c)
	require.NoError(t, err)
	assert.JSONEq(t, `{"request":{"X-A":"1"}}`, string(out))
}

func TestCustomHeaders_ScanValue_RoundTripLegacyFlat(t *testing.T) {
	var c CustomHeaders
	require.NoError(t, c.Scan(`{"X-Legacy":"v"}`)) // legacy flat row from DB
	assert.Equal(t, map[string]string{"X-Legacy": "v"}, c.Request)

	v, err := c.Value()
	require.NoError(t, err)
	assert.JSONEq(t, `{"request":{"X-Legacy":"v"}}`, string(v.([]byte)))
}

func TestCustomHeaders_Value_EmptyIsNil(t *testing.T) {
	v, err := CustomHeaders{}.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./backend/internal/models/ -run TestCustomHeaders -count=1`
Expected: FAIL — `undefined: CustomHeaders`.

- [ ] **Step 3: Implement the `CustomHeaders` type**

In `backend/internal/models/proxy.go`, ensure imports include `strings` (alongside the existing `database/sql/driver`, `encoding/json`). Add after the `JSONField` definitions:

```go
// CustomHeaders holds custom HTTP headers to set on the request forwarded to the
// upstream (Request) and/or on the response returned to the client (Response).
// It is stored as JSON text. To stay backward compatible it accepts BOTH the
// nested {"request":{...},"response":{...}} shape and the legacy flat
// {"X-Header":"value"} map (interpreted as request headers), and always
// serialises back as the nested shape.
type CustomHeaders struct {
	Request  map[string]string `json:"request,omitempty"`
	Response map[string]string `json:"response,omitempty"`
}

// IsEmpty reports whether no headers are configured in either direction.
func (c CustomHeaders) IsEmpty() bool {
	return len(c.Request) == 0 && len(c.Response) == 0
}

// UnmarshalJSON accepts the nested shape or a legacy flat map. HTTP header values
// are always strings, so a top-level value that is a JSON object marks the nested
// shape; otherwise the whole object is a flat request-header map.
func (c *CustomHeaders) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*c = CustomHeaders{}
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	nested := len(raw) > 0
	for key, val := range raw {
		if key != "request" && key != "response" {
			nested = false
			break
		}
		v := strings.TrimSpace(string(val))
		if len(v) == 0 || v[0] != '{' {
			nested = false
			break
		}
	}

	if nested {
		type alias CustomHeaders // break the UnmarshalJSON recursion
		var a alias
		if err := json.Unmarshal(data, &a); err != nil {
			return err
		}
		*c = CustomHeaders(a)
		return nil
	}

	flat := make(map[string]string)
	if err := json.Unmarshal(data, &flat); err != nil {
		return err
	}
	*c = CustomHeaders{Request: flat}
	return nil
}

// Value implements driver.Valuer: store the nested JSON shape (SQL NULL if empty).
func (c CustomHeaders) Value() (driver.Value, error) {
	if c.IsEmpty() {
		return nil, nil
	}
	type alias CustomHeaders
	return json.Marshal(alias(c))
}

// Scan implements sql.Scanner: load JSON text, tolerating legacy flat rows.
func (c *CustomHeaders) Scan(value interface{}) error {
	if value == nil {
		*c = CustomHeaders{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	if len(bytes) == 0 {
		*c = CustomHeaders{}
		return nil
	}
	return c.UnmarshalJSON(bytes)
}
```

> Note: `MarshalJSON` is intentionally NOT defined — the default struct marshaling already produces the nested `{"request":...,"response":...}` shape with `omitempty`, which is what `TestCustomHeaders_MarshalJSON_AlwaysNested` asserts. `Value()` uses an `alias` to get that same default marshaling.

- [ ] **Step 4: Change the `Proxy.CustomHeaders` field type**

In `backend/internal/models/proxy.go`, change line 29 from:

```go
	CustomHeaders         JSONField   `json:"custom_headers,omitempty" gorm:"type:text"`
```
to:
```go
	CustomHeaders         CustomHeaders `json:"custom_headers,omitempty" gorm:"type:text"`
```

(The field is a value type; empty serialises as `{}`. That is acceptable.)

- [ ] **Step 5: Run model tests**

Run: `go test ./backend/internal/models/ -count=1`
Expected: PASS (all `TestCustomHeaders*` pass; existing model tests still pass). If an existing test set `CustomHeaders` via `JSONField{...}`, update it to `CustomHeaders{Request: map[string]string{...}}`.

- [ ] **Step 6: Make the package compile (builder will break — fix in Task 2)**

Run: `go build ./backend/internal/models/`
Expected: models package builds. (Other packages referencing the old `len(proxy.CustomHeaders)` will fail to compile; Task 2 fixes the builder.)

- [ ] **Step 7: Commit**

```bash
git add backend/internal/models/proxy.go backend/internal/models/proxy_test.go
git commit -m "feat: typed CustomHeaders model with flexible request/response parsing"
```

---

## Task 2: Builder applies request + response headers

**Files:**
- Modify: `backend/internal/caddy/config/http_builder.go:205-213`
- Test: `backend/internal/caddy/config/builder_test.go`

- [ ] **Step 1: Update the existing test + add a response-header test**

In `backend/internal/caddy/config/builder_test.go`, replace the body of `TestHTTPBuilder_BuildReverseProxyRoutes_WithCustomHeaders` so it uses the new type and asserts request headers are applied, and add a new response-header test:

```go
func TestHTTPBuilder_BuildReverseProxyRoutes_WithCustomHeaders(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.CustomHeaders = models.CustomHeaders{
		Request: map[string]string{"X-Custom-Header": "custom-value", "X-Another": "another-value"},
	}

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	handler := routes[0].Handle[0]
	require.NotNil(t, handler["headers"])
	hc := handler["headers"].(*HeadersConfig)
	require.NotNil(t, hc.Request)
	assert.Equal(t, []string{"custom-value"}, hc.Request.Set["X-Custom-Header"])
	assert.Equal(t, []string{"another-value"}, hc.Request.Set["X-Another"])
}

func TestHTTPBuilder_BuildReverseProxyRoutes_WithResponseHeaders(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.CustomHeaders = models.CustomHeaders{
		Request:  map[string]string{"X-Req": "r"},
		Response: map[string]string{"X-Res": "s"},
	}

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	hc := routes[0].Handle[0]["headers"].(*HeadersConfig)
	require.NotNil(t, hc.Request)
	assert.Equal(t, []string{"r"}, hc.Request.Set["X-Req"])
	require.NotNil(t, hc.Response)
	assert.Equal(t, []string{"s"}, hc.Response.Set["X-Res"])
}
```

> Verify `HeaderOps`'s set field name/shape (`http_handlers.go:134-` and `SetHeader` at line 484). The asserts above assume `HeaderOps.Set map[string][]string` populated by `SetHeader`. If the field has a different name (e.g. lowercase or `Set http.Header`), adjust the asserts to match the real struct — read it first.

- [ ] **Step 2: Run tests to verify they fail (or fail to compile)**

Run: `go test ./backend/internal/caddy/config/ -run 'WithCustomHeaders|WithResponseHeaders' -count=1`
Expected: FAIL/compile error — builder still references the old `len(proxy.CustomHeaders)` map iteration and does not set response headers.

- [ ] **Step 3: Update the builder**

In `backend/internal/caddy/config/http_builder.go`, replace the custom-headers block (lines ~205-213):

```go
	// Add custom headers
	if len(proxy.CustomHeaders) > 0 {
		for key, value := range proxy.CustomHeaders {
			if strVal, ok := value.(string); ok {
				handler.Headers.Request.SetHeader(key, strVal)
			}
		}
	}
```
with:
```go
	// Add custom request headers (forwarded to the upstream).
	for key, value := range proxy.CustomHeaders.Request {
		handler.Headers.Request.SetHeader(key, value)
	}
	// Add custom response headers (returned to the client).
	if len(proxy.CustomHeaders.Response) > 0 {
		if handler.Headers.Response == nil {
			handler.Headers.Response = &HeaderOps{}
		}
		for key, value := range proxy.CustomHeaders.Response {
			handler.Headers.Response.SetHeader(key, value)
		}
	}
```

- [ ] **Step 4: Run the config package tests**

Run: `go test ./backend/internal/caddy/config/ -count=1`
Expected: PASS (new + updated tests pass; rest of package unaffected).

- [ ] **Step 5: Verify the whole backend builds and unit-tests pass**

Run: `go build ./... && go test ./backend/internal/... -count=1`
Expected: builds clean; all backend unit tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/caddy/config/http_builder.go backend/internal/caddy/config/builder_test.go
git commit -m "feat: apply custom request and response headers in reverse proxy builder"
```

---

## Task 3: Traffic E2E — un-skip and assert both directions

**Files:**
- Modify: `backend/tests/integration/traffic_l7_test.go` (the `custom_headers` subtest)

- [ ] **Step 1: Rewrite the `custom_headers` subtest**

Replace the existing skipped `custom_headers` subtest in `TestTraffic_L7` with (remove the `t.Skip` and the old comments):

```go
	t.Run("custom_headers", func(t *testing.T) {
		host := "hdr.test.local"
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", map[string]any{
			"type": "reverse_proxy", "name": "hdr", "hostname": host,
			"upstreams": []map[string]any{{"host": "echo1", "port": 8080, "scheme": "http"}},
			"custom_headers": map[string]any{
				"request":  map[string]string{"X-Req-Test": "req-value"},
				"response": map[string]string{"X-Res-Test": "res-value"},
			},
		})
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create proxy: %d", resp.StatusCode)
		}
		env.triggerSync(t)
		env.waitL7(t, host, http.StatusOK)

		got, body := env.l7Get(t, host, "/", nil)
		// Response header reaches the client.
		if got.Header.Get("X-Res-Test") != "res-value" {
			t.Fatalf("expected X-Res-Test response header, headers: %v", got.Header)
		}
		// Request header reaches the backend (mendhak echo reflects request
		// headers under a lowercased "headers" object in its JSON body).
		var echoed struct {
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal(body, &echoed); err != nil {
			t.Fatalf("parse echo body: %v (body=%s)", err, string(body))
		}
		if echoed.Headers["x-req-test"] != "req-value" {
			t.Fatalf("expected backend to receive X-Req-Test request header, got headers: %v", echoed.Headers)
		}
	})
```

Add `"encoding/json"` to the imports of `traffic_l7_test.go` if not already present.

- [ ] **Step 2: Rebuild the image (backend Go changed in Tasks 1-2) and run the subtest**

Run: `docker build -t waygates-test:latest . && go test -tags traffic -run 'TestTraffic_L7/custom_headers' ./backend/tests/integration/ -count=1 -v`
Expected: PASS — response carries `X-Res-Test: res-value`; backend received `x-req-test: req-value`.

> CRITICAL: do not weaken this test. If the response header does not appear, the builder/model change is incomplete — fix it (Tasks 1-2), do not relax the assertion. If `echoed.Headers` uses a different JSON layout in this mendhak version, inspect the real body (it is printed on failure) and adjust the parse to the real shape — but keep asserting both directions.

- [ ] **Step 3: gofmt/vet**

Run: `gofmt -l backend/tests/integration/traffic_l7_test.go && go vet -tags traffic ./backend/tests/integration/`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add backend/tests/integration/traffic_l7_test.go
git commit -m "test: assert custom request and response headers end-to-end"
```

---

## Task 4: Frontend types

**Files:**
- Modify: `ui/src/types/proxy.ts`

- [ ] **Step 1: Add the `CustomHeaders` interface and replace the 3 field types**

In `ui/src/types/proxy.ts`, add near the other shared interfaces:

```ts
export interface CustomHeaders {
  request?: Record<string, string>;
  response?: Record<string, string>;
}
```

Then change all three `custom_headers?: Record<string, string>;` occurrences (lines ~57, ~74, ~112 — `ProxyConfig`, `CreateReverseProxyRequest`, `UpdateProxyRequest`) to:

```ts
  custom_headers?: CustomHeaders;
```

- [ ] **Step 2: Typecheck**

Run: `pnpm --dir ui build`
Expected: FAIL — `$proxyId.tsx` and/or the form now have type mismatches around `custom_headers` (fixed in Tasks 5-6). Confirm the errors are confined to those files.

- [ ] **Step 3: Commit**

```bash
git add ui/src/types/proxy.ts
git commit -m "feat: nested CustomHeaders type in UI proxy types"
```

---

## Task 5: Frontend form — request/response header editors

**Files:**
- Modify: `ui/src/components/proxy/forms/reverse-proxy-form.tsx`

- [ ] **Step 1: Add header local state, init helper, and add/remove/update helpers**

Near the top of the component (after the `upstreams` state at line ~98), add:

```tsx
  type HeaderPair = { name: string; value: string };
  const toPairs = (rec?: Record<string, string>): HeaderPair[] =>
    Object.entries(rec ?? {}).map(([name, value]) => ({ name, value }));
  const toRecord = (pairs: HeaderPair[]): Record<string, string> => {
    const out: Record<string, string> = {};
    for (const p of pairs) {
      const name = p.name.trim();
      if (name) out[name] = p.value;
    }
    return out;
  };

  const [requestHeaders, setRequestHeaders] = useState<HeaderPair[]>(() =>
    toPairs(initialData?.custom_headers?.request),
  );
  const [responseHeaders, setResponseHeaders] = useState<HeaderPair[]>(() =>
    toPairs(initialData?.custom_headers?.response),
  );

  const addHeader = (setter: typeof setRequestHeaders) =>
    setter((prev) => [...prev, { name: '', value: '' }]);
  const removeHeader = (setter: typeof setRequestHeaders, index: number) =>
    setter((prev) => prev.filter((_, i) => i !== index));
  const updateHeader = (
    setter: typeof setRequestHeaders,
    index: number,
    key: keyof HeaderPair,
    val: string,
  ) => setter((prev) => prev.map((h, i) => (i === index ? { ...h, [key]: val } : h)));
```

- [ ] **Step 2: Reset header state when `initialData` changes**

In the existing `useEffect` that resets the form on `initialData` change (around line 178-183), add:

```tsx
      setRequestHeaders(toPairs(initialData.custom_headers?.request));
      setResponseHeaders(toPairs(initialData.custom_headers?.response));
```

(Place inside the `if (initialData) { ... }` block alongside `setUpstreams(...)`.)

- [ ] **Step 3: Include headers in the submit payload**

In the form's `onSubmit` (around line 144-173), after the `load_balancing` block and before `onSubmit(data, ...)`, add:

```tsx
      const reqHeaders = toRecord(requestHeaders);
      const resHeaders = toRecord(responseHeaders);
      if (Object.keys(reqHeaders).length > 0 || Object.keys(resHeaders).length > 0) {
        data.custom_headers = {
          ...(Object.keys(reqHeaders).length > 0 ? { request: reqHeaders } : {}),
          ...(Object.keys(resHeaders).length > 0 ? { response: resHeaders } : {}),
        };
      }
```

- [ ] **Step 4: Render the two header editors**

Add a new card section in the form's JSX (after the upstreams/load-balancing section, before the submit buttons). This renders both editors via a small inline helper block — paste it twice (request, then response) with the respective state/labels:

```tsx
        <Card>
          <CardHeader>
            <CardHeading>
              <CardTitle>Custom Headers</CardTitle>
              <CardDescription>
                Add headers sent to the upstream (request) or returned to the client (response).
              </CardDescription>
            </CardHeading>
          </CardHeader>
          <CardContent className="space-y-6">
            {(
              [
                { label: 'Request headers (to upstream)', pairs: requestHeaders, setter: setRequestHeaders },
                { label: 'Response headers (to client)', pairs: responseHeaders, setter: setResponseHeaders },
              ] as const
            ).map((group) => (
              <div key={group.label} className="space-y-2">
                <div className="flex items-center justify-between">
                  <FieldLabel>{group.label}</FieldLabel>
                  <Button type="button" variant="outline" size="sm" onClick={() => addHeader(group.setter)}>
                    <Plus className="mr-1 size-4" />
                    Add header
                  </Button>
                </div>
                {group.pairs.map((pair, index) => (
                  // biome-ignore lint/suspicious/noArrayIndexKey: header rows are positional, edited in place
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      placeholder="Header-Name"
                      value={pair.name}
                      onChange={(e) => updateHeader(group.setter, index, 'name', e.target.value)}
                      className="flex-1"
                    />
                    <Input
                      placeholder="value"
                      value={pair.value}
                      onChange={(e) => updateHeader(group.setter, index, 'value', e.target.value)}
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeHeader(group.setter, index)}
                    >
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            ))}
          </CardContent>
        </Card>
```

(`Card`, `CardContent`, `CardHeader`, `CardHeading`, `CardTitle`, `CardDescription`, `FieldLabel`, `Input`, `Button`, `Plus`, `Trash2` are already imported in this file.)

- [ ] **Step 5: Typecheck/build the UI**

Run: `pnpm --dir ui build`
Expected: the form file's `custom_headers` usage typechecks (errors may remain only in `$proxyId.tsx`, fixed in Task 6).

- [ ] **Step 6: Commit**

```bash
git add ui/src/components/proxy/forms/reverse-proxy-form.tsx
git commit -m "feat: request/response custom header editors in reverse proxy form"
```

---

## Task 6: Frontend change-detection + lint/build green

**Files:**
- Modify: `ui/src/routes/_dashboard/proxies/$proxyId.tsx`

- [ ] **Step 1: Read the change-detection block and fix the custom_headers comparison**

At `ui/src/routes/_dashboard/proxies/$proxyId.tsx:65-67` the current code is:

```tsx
        const oldHeaders = originalProxy.custom_headers || [];
        const newHeaders = newData.custom_headers || [];
        if (JSON.stringify(oldHeaders) !== JSON.stringify(newHeaders)) return true;
```

`custom_headers` is now a nested object, not an array. Trace how `newData` is produced for this comparison (it is the edited/submitted data). Make the comparison robust to the nested object by defaulting to `{}` and normalizing key order. Replace with:

```tsx
        const sortedHeaders = (h?: { request?: Record<string, string>; response?: Record<string, string> }) =>
          JSON.stringify({
            request: Object.fromEntries(Object.entries(h?.request ?? {}).sort()),
            response: Object.fromEntries(Object.entries(h?.response ?? {}).sort()),
          });
        if (sortedHeaders(originalProxy.custom_headers) !== sortedHeaders(newData.custom_headers)) {
          return true;
        }
```

If `newData` here is NOT the submitted `CreateReverseProxyRequest` (e.g. it is the flat form values that do not carry `custom_headers`), adjust so the comparison reads the actual edited header state. Verify by reading the surrounding function and how `newData` is assembled; the dirty-check must reflect real header edits (test by editing a header in the running app and confirming the save button enables).

- [ ] **Step 2: Build + lint the UI**

Run: `pnpm --dir ui build && make lint-ui`
Expected: build succeeds; Biome reports no NEW errors (the header-row `biome-ignore` keeps `noArrayIndexKey` quiet, consistent with the existing form arrays).

- [ ] **Step 3: Commit**

```bash
git add ui/src/routes/_dashboard/proxies/$proxyId.tsx
git commit -m "fix: nested custom_headers comparison in proxy change detection"
```

---

## Task 7: Full verification + docs

**Files:**
- Modify (optional): `docs/API_PROXY.md` if it documents the `custom_headers` shape.

- [ ] **Step 1: Backend full unit suite**

Run: `go test ./backend/... -short -count=1`
Expected: all packages `ok`.

- [ ] **Step 2: Full traffic suite (rebuilds image)**

Run: `make test-traffic`
Expected: `TestTraffic_L7` all subtests PASS (custom_headers now PASSING, no SKIP); `TestTraffic_L4` all PASS.

- [ ] **Step 3: UI build + lint**

Run: `pnpm --dir ui build && make lint-ui`
Expected: clean.

- [ ] **Step 4: Update API docs if present**

If `docs/API_PROXY.md` documents `custom_headers` as a flat map (e.g. `custom_headers TEXT, -- JSON: {"X-Header": "value"}`), add a note that it now accepts `{"request":{...},"response":{...}}` and that a flat map is still accepted as request headers. Commit:

```bash
git add docs/API_PROXY.md
git commit -m "docs: document request/response custom_headers shape"
```

(Skip this commit if no such doc reference exists.)

---

## Self-review notes (for the implementer)

- **Spec coverage:** Task 1 = model type + flexible parse/normalize/Scan/Value; Task 2 = builder both directions; Task 3 = E2E both directions (un-skip); Tasks 4-6 = UI types/form/change-detection; Task 7 = full verification + docs. All spec sections map to a task.
- **Back-compat:** flat input → request headers (model UnmarshalJSON + builder); legacy DB rows via `Scan`; no migration.
- **Type consistency:** Go `CustomHeaders{Request, Response map[string]string}`; TS `CustomHeaders{request?, response?}`; builder uses `HeadersConfig.Response *HeaderOps` + `HeaderOps.SetHeader`; E2E sends `{"request":{...},"response":{...}}`.
- **Known iteration points:** the exact `HeaderOps` set-field name (Task 2 asserts), and the mendhak echo JSON header layout (Task 3) — both verifiable from the real code/output; adjust assertions to the real shape without weakening intent.
- **Empty serialization:** the value-type field serialises empty as `{}`; acceptable. If the team prefers omitting it, switch `Proxy.CustomHeaders` to `*CustomHeaders` and nil-guard the builder — not required by this plan.
