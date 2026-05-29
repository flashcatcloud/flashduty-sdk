# CLAUDE.md

Guidance for AI coding agents (and humans) working in this repository. `AGENTS.md`
is a symlink to this file.

## What this is

`github.com/flashcatcloud/flashduty-sdk` is the official **typed Go client** for the
[Flashduty](https://flashcat.cloud) open API. It is a public, `go get`-able library
with external consumers; two first-party consumers are the Flashduty CLI and the
Flashduty MCP server. Treat every exported symbol as a public API surface.

## Architecture

- **Single root package `flashduty`.** No subpackages. Flat, discoverable.
- **One file per API domain** — `incidents.go`, `schedules.go`, `alerts.go`,
  `statuspage.go`, `audit.go`, `reports.go`, `insight.go`, … Each holds that domain's
  `Client` methods and its request/response structs.
- **`client.go`** — transport: `makeRequest`, and the generic `postData[T]` / `getData[T]`
  helpers that decode the API envelope into a typed value.
- **`types.go`** — shared response types used across domains.
- **`errors.go`** — typed API errors. **`logger.go`**, **`client_options.go`** — cross-cutting.

## Conventions (our standards — follow them exactly)

### Timestamps
- **All absolute time fields in RESPONSE structs use `flashduty.Timestamp`**
  (Unix **seconds**) or **`flashduty.TimestampMilli`** (Unix **milliseconds**),
  matching the wire unit. Both marshal to an RFC3339 string and unmarshal from
  epoch-or-RFC3339; pick the variant the API actually sends (most fields are
  seconds; feed/audit endpoints are milliseconds). This keeps machine-readable
  output human- and LLM-friendly without any downstream guessing. Do **not** add
  bare `int64` "...At/...Time" fields to responses.
- **Durations, cyclic-window offsets, and counts stay `int64`** (e.g. a rotation
  length, a notification lead-time). They are not instants. When a time-typed field
  must stay `int64`, state its unit and meaning in a comment — silence is a bug.
- **Request/input struct time fields stay `int64`** (callers pass epochs; inputs are
  never rendered for humans). Document the unit (`// Unix seconds`).
- If you genuinely cannot tell whether an `int64` is an instant or a duration, it
  stays `int64` until the API contract proves otherwise. Never guess a field into a
  timestamp — a wrong rendered date is worse than a raw integer.

### API methods
- Every exported `Client` method returns a **typed struct/slice**, never `any` /
  `interface{}` / `map[string]any`. If the API adds a new response, add the struct.
- Signature shape: `func (c *Client) Verb(ctx context.Context, in *VerbInput) (*VerbResult, error)`.
- Decode through `postData[T]` / `getData[T]`; don't hand-roll `json.Unmarshal` per method.

## Engineering principles

1. **First principles.** Reason from the API contract and the caller's real need, not
   from analogy. Ask "what does this field/type *have* to be?"
2. **No over-engineering (YAGNI).** Build for the requirement that exists. No speculative
   config knobs, no extension points without a concrete second caller.
3. **Root cause first.** Fix the cause, not the symptom. If a change touches many call
   sites, route them through one shared helper rather than patching each.
4. **No transitional shims.** Land at the end state in one change — no bridging
   feature flags, no dead-code paths, no "clean it up later."

## Testing & verification

- Gate (no Makefile/CI yet — run all four, all must be clean):
  ```
  go test ./... && go vet ./... && go build ./... && gofmt -l .
  ```
  `gofmt -l .` must print nothing.
- Unit tests are **table-driven** and **hermetic** — no live network. Use `httptest`
  to stub the API; pin request shape and decode behavior.
- New behavior ships with a test that fails before the change and passes after.
- "Verified" means *I ran the gate and saw it pass.* Never infer success from a build
  alone; never claim done on an unrun test.

## Versioning & compatibility

- Pre-1.0 (`v0.x`). Under Go module rules a `v0.x` bump carries no compat guarantee, but
  this is a published SDK: treat any exported-type or signature change as **breaking** and
  call it out explicitly in the PR description. Breaking changes bump the **minor** (`v0.(x+1).0`).
- Consumers pin via pseudo-version (`go get github.com/flashcatcloud/flashduty-sdk@<sha>`).

## Git workflow

- Feature work lives on its own branch off `main`; deliver via PR. Never commit to `main`.
- Never `git push --force`.
- Commit messages stay clean — **no `Co-Authored-By` / "Generated with" trailers** for
  any code agent.
- One feature per branch/PR so review stays scoped.
