# Decisions: Derive `routing` context architecture

## Vagueness: which capabilities of `pkg/router` should the `routing` architecture cover?

Decision: Document all externally observable capabilities of `pkg/router` — API v1/v2 dispatch, BLOB endpoints, N10N SSE, reverse proxy, ACME challenge listener, admin/debug pprof port, per-route rate limiting, CORS — plus domain management and HTTPS termination. Keep descriptions boundary-only so behavior owned by `apps` is not duplicated.

- Pros: matches what the production binary actually exposes; reviewers can map each `pkg/router` handler to an architecture component; avoids hidden capabilities; consistent with the breadth of `apps/arch-processing.md`
- Cons: largest scope; risk of duplicating descriptions that conceptually belong to `apps` — must be kept boundary-only
- Confidence: high

Alternatives:

1. Document HTTP boundary + cross-cutting concerns only — request reception, TLS termination, ACME, domain management, reverse proxy, rate limiting, CORS, admin/debug port; treat API/BLOB/N10N as "dispatched to `apps`" without listing each route family
   - Pros: cleaner separation from `apps`; smaller, more stable architecture
   - Cons: hides that `pkg/router` owns N10N SSE state and BLOB endpoint shapes; reviewers can't tell from the architecture which routes exist
   - Confidence: medium
2. Narrow scope: only what `domain.md` already attributes to `routing` — request routing, domain management, HTTPS/ACME, reverse proxy; defer N10N, BLOB, admin port, rate limiting to other contexts or future change requests
   - Pros: smallest, fastest to write; sticks to the issue text literally
   - Cons: misrepresents the codebase; reviewers will flag missing coverage; likely to be revisited soon
   - Confidence: low

## Ambiguity: should the `routing` architecture be a single `arch.md` or split into `arch-{subsystem}.md` chapters?

Decision: Split into four chapters — `arch.md` (overview + admin endpoint + cross-subsystem components) + `arch-ingress.md` (public HTTP/HTTPS listener, TLS termination, ACME, CORS, query limit, dispatch to `apps`, N10N SSE, BLOB) + `arch-reverse-proxy.md` (the four route kinds, host/path matching, upstream forwarding) + `arch-debug.md` (pprof endpoints mounted on the admin endpoint). The admin/debug placement is refined further in a separate decision below.

- Pros: mirrors the `apps` precedent; isolates the reverse proxy (a self-contained subsystem with its own configuration model) from the main ingress pipeline; keeps each chapter focused
- Cons: more files to maintain; small risk of cross-chapter duplication for shared concepts (e.g., domain config drives both ACME and reverse-proxy host routes)
- Confidence: high

Alternatives:

1. Single `arch.md` covering everything
   - Pros: smallest footprint; everything in one place; easy to navigate for a context that is essentially "the HTTP boundary"
   - Cons: one large file mixing ingress pipeline, reverse-proxy route model, ACME state machine, admin port, and domain config; harder to review in chunks; diverges from the `apps` precedent
   - Confidence: medium
2. Split by lifecycle, mirroring `apps`: `arch.md` + `arch-deployment.md` + `arch-processing.md`
   - Pros: maximum consistency with `apps`; clear separation between "what the operator configures" and "what happens per request"
   - Cons: forced fit — reverse-proxy matching is per-request but its route table is deployment-time, so it would straddle both chapters; ACME is mostly background, neither pure deployment nor per-request
   - Confidence: medium

## Inconsistency: "cluster and per-app domain configuration" vs. what the codebase actually supports

Decision: Drop "per-app" and describe domain management as cluster-wide operator configuration supplied by `@Admin` at VVM startup via `RouterParams` (reverse-proxy route table and ACME-managed hostname list). Per-app dynamic domain registration is not implemented and is out of scope.

- Pros: matches the codebase exactly (`pkg/router/types.go::RouterParams`); matches `uspecs/specs/prod/domain.md`; nothing aspirational sneaks into a derived architecture
- Cons: under-promises if per-app domain provisioning is on a near-term roadmap (architecture would need to be revised then)
- Confidence: high

Alternatives:

1. Keep "per-app" as an aspirational/future capability and mark it explicitly
   - Pros: signals known direction; reviewers see what is missing
   - Cons: this change is `type: docs` deriving architecture from the existing codebase; aspirational content belongs in a different change request; risks confusing readers about what is built
   - Confidence: low
2. Investigate further before deciding
   - Pros: avoids both under- and over-promising
   - Cons: extra research step; likely to return the same "not implemented" answer
   - Confidence: medium

## Ambiguity: "per-route rate limiting" mischaracterizes what `pkg/router` actually limits

Decision: Rephrase as a per-workspace concurrent-query limit on API endpoints — a cap on in-flight queries per `WSID` configured via `MaxQueriesPerWS`, with backpressure rejections when the cap is reached.

- Pros: matches `pkg/router/impl_limiter.go::wsQueryLimiter` exactly; avoids inventing a feature that does not exist; correctly conveys that throttling is per-workspace, not per-route or per-client-IP
- Cons: slightly longer phrasing; reviewers expecting traditional rate limiting may be surprised
- Confidence: high

Alternatives:

1. Drop the bullet entirely from `## What`
   - Pros: smallest `## What`; avoids the framing debate altogether
   - Cons: the limit is externally observable — clients get rejections when they exceed it; omitting it understates what the routing context guarantees
   - Confidence: low
2. Keep "per-route rate limiting" wording but expand scope to add a true per-route rate limiter
   - Pros: would deliver a commonly expected capability
   - Cons: this change is `type: docs` deriving architecture from existing code; adding behavior belongs in a separate `feat` change request
   - Confidence: low

## Ambiguity: how `routing`'s BLOB and N10N endpoints relate to the `apps` context's `[BLOB processor]` and `[N10n broker]`

Decision: Make the seam explicit in `## What` for both capabilities. BLOB endpoints in `routing` validate the HTTP request and delegate to the `apps`-owned `[BLOB processor]`; N10N SSE endpoints in `routing` translate HTTP calls into subscriptions on the `apps`-owned `[N10n broker]` and stream the resulting events back to the client. The routing architecture references these `apps` components by name and cross-references `apps/arch-processing.md`; it does not redescribe their behavior.

- Pros: removes overlap with `apps/arch-processing.md`; makes the cross-context dependency visible; gives the technical-design pass a clear instruction for `arch-ingress.md`
- Cons: requires the routing `arch-ingress.md` to cross-reference `apps/arch-processing.md` for the BLOB processor and N10N broker
- Confidence: high

Alternatives:

1. Keep the current high-level phrasing and resolve the seam during `uimpl`
   - Pros: keeps `## What` shorter
   - Cons: the boundary rule in `## What` is too generic to enforce for BLOB/N10N without naming the receiving components
   - Confidence: medium
2. Treat BLOB processor and N10N broker as `routing` components instead
   - Pros: consolidates each capability in one context
   - Cons: contradicts the existing `apps` architecture; the processor and broker are invoked from `apps` actualizers and command processors too, not only from HTTP; misplaces them
   - Confidence: low

## Inconsistency: `## Why` and the last `## What` bullet still imply a narrow, single-file scope

Decision: Widen `## Why` to reflect the full HTTP-boundary scope (request routing, reverse-proxy forwarding, HTTPS/ACME, BLOB and N10N endpoints, domain management, admin/debug listener) and reword the closing `## What` bullet from "single architecture reference" to "complete architecture reference" so the three-file split is no longer contradicted.

- Pros: aligns the framing of the change with the resolved scope; removes the "single file" implication; reviewer can read `## Why` and predict the deliverable
- Cons: minor edits across two sections
- Confidence: high

Alternatives:

1. Fix only the last `## What` bullet ("single" → "complete"), leave `## Why` as-is
   - Pros: smaller diff; `## Why` rule allows brevity
   - Cons: `## Why` continues to undersell the change; reviewers may misjudge blast radius
   - Confidence: medium
2. Leave both as-is — readers will infer scope from `## What`
   - Pros: no further edits
   - Cons: keeps a real inconsistency between sections
   - Confidence: low

## Inconsistency: the admin endpoint description is duplicated in two places with inconsistent depth vs. peer bullets

Decision: Trim the `arch.md` file-split sub-bullet to a peer-style one-liner ("the admin endpoint") so it matches the brevity of the other three file-split sub-bullets. Keep the full admin-endpoint description (localhost port, bootstrap rationale, `federation.AdminFunc`, file markers `(arch.md)` and `(arch-debug.md)`) only in the capability bullet.

- Pros: matches the structure of the peer sub-bullets; removes the duplication; capability bullets remain the single source of truth for what is documented and where
- Cons: a reader scanning only the file-split bullet learns less about `arch.md`'s contents
- Confidence: high

Alternatives:

1. Trim the capability bullet to a one-liner pointer; keep the full description only in the file-split sub-bullet for `arch.md`
   - Pros: collocates the description with the file it belongs to
   - Cons: breaks symmetry with the other capability bullets (each of which carries the full description); the capability list becomes a non-uniform mix of full descriptions and pointers
   - Confidence: medium
2. Leave both as-is
   - Pros: no further edits
   - Cons: keeps the duplication and the structural inconsistency with peer sub-bullets
   - Confidence: low

## Ambiguity: where the admin/debug HTTP listener belongs in the file split

Decision (user-provided): The admin listener is a single endpoint that listens on `localhost:AdminPort` (default `55555`) and mounts the full handler set, including the debug endpoints. Describe the admin endpoint in `arch.md`, including its bootstrapping role — the VVM service pipeline starts the admin endpoint before the bootstrap operator so that bootstrap can call internal API functions (e.g. `c.cluster.DeployApp`) via `federation.AdminFunc` over the localhost-only port before the public endpoint opens to external traffic. Describe the debug endpoints (pprof) in a separate `arch-debug.md`. This adds a fourth chapter (`arch-debug.md`) to the file split decided earlier.

- Pros: matches the production code shape (`pkg/router/provide.go` builds the admin endpoint via `getRouterService` bound to `httpu.LocalhostIP:AdminPort`; `pkg/vvm/provide.go::provideServicePipeline` orders admin-endpoint → bootstrap → public-endpoint; `pkg/btstrp/impl.go::callDeployApp` uses `federation.AdminFunc`); keeps `arch.md` focused on the operator-local endpoint concept; isolates pprof handlers in their own chapter so future debug surface can grow there without bloating `arch.md`
- Cons: introduces a fourth chapter; readers must follow a link from `arch.md` to `arch-debug.md` to see which debug routes are mounted
- Confidence: user-provided

Alternatives:

1. Move the admin/debug listener into `arch-ingress.md` as a sibling of the public and ACME listeners
   - Pros: all listeners in one chapter; uniform listener model
   - Cons: stretches "ingress" to include an operator-facing localhost port; mixes user-facing and operator-facing surfaces
   - Confidence: high
2. Give the admin/debug listener its own chapter `arch-admin.md` (instead of splitting admin endpoint and debug endpoints)
   - Pros: clean separation between user-facing ingress and operator-facing admin
   - Cons: less granular than the chosen split; conflates "the localhost endpoint" with "what is mounted on it"
   - Confidence: medium
3. Keep the admin/debug listener in `arch.md` and accept that `arch.md` mixes overview and one concrete component
   - Pros: no file change; smallest deliverable
   - Cons: violates the overview/subsystem split rationale from the prior decision
   - Confidence: low

## Vagueness: `pkg/ihttpimpl` out-of-scope marker does not say where the alternative implementation is actually used

Decision (user-provided): Leave the bullet as-is — keep the terse "Document only the production implementation wired through the VVM (`pkg/router`); note the separate HTTP abstraction (`pkg/ihttpimpl`) as out of scope" wording. Do not expand it to name `cmd/voedger server`, `pkg/ihttp`, or `pkg/ihttpctl`.

- Pros: shortest framing; keeps `## What` focused on the in-scope deliverable; out-of-scope marker is sufficient for reviewers who only need to know it is excluded
- Cons: any reviewer who has not read `cmd/voedger/server.go` will wonder what `pkg/ihttpimpl` is and may grep before commenting
- Confidence: user-provided

Alternatives:

1. Name the consumer: expand the bullet to "the HTTP processor abstraction used by the standalone `cmd/voedger server` command (`pkg/ihttp` + `pkg/ihttpctl` + `pkg/ihttpimpl`) is out of scope"
   - Pros: factual; reviewer immediately understands what is being excluded and why
   - Cons: longer bullet; introduces extra package names that may not matter at `## What` level
   - Confidence: high
2. Drop the `pkg/ihttpimpl` mention entirely
   - Pros: simplest; no out-of-scope debate
   - Cons: loses the negative-space signal that there is a second HTTP stack in the repo
   - Confidence: medium

## Inconsistency: `## What` (admin endpoint bullet) claims "the full handler set" but BLOB endpoints and the per-WS query limiter are not mounted on the admin endpoint

Decision: Replace "that mounts the full handler set" with "that mounts the same handler set as the public endpoint except BLOB endpoints and the per-WS query limiter" in the admin-endpoint bullet of `## What`. The fact that BLOB requests would 404 on the admin port and that queries are not rate-limited there is caller-observable behavior, so it belongs in `## What`. Reasons for the carve-outs (`blobRequestHandler == nil`, `limiter is nil for Admin and ACME services`) stay in `arch.md`.

- Pros: matches reality exactly (per `pkg/router/provide.go::Provide`); matches `arch.md` line 89; stays at behavior level — no implementation symbols leak into `## What`
- Cons: slightly longer bullet
- Confidence: user-provided

Alternatives:

1. Drop the qualifier — "that mounts the routing handlers"
   - Pros: shortest fix
   - Cons: loses the comparison-to-public-endpoint signal that helps readers locate the admin endpoint in the spec
   - Confidence: medium
2. Enumerate the mounted families — "that mounts the router-checker, API v1, API v2, debug, and reverse-proxy handlers (no BLOB endpoints, no query limiter)"
   - Pros: most precise; no follow-up read of `arch.md` needed
   - Cons: leaks handler-family names into `## What`, contrary to the boundary rule that `## What` should not name implementation symbols just to make a bullet concrete
   - Confidence: medium

## Inconsistency: `tls--td.md` (Feature Technical Design) was added but `change.md` only authorizes architecture files

Decision: Convert `tls--td.md` into `arch-tls.md`, a fifth Context Subsystem Architecture chapter, and extract the TLS-related components (`[ACME HTTP-01 listener]`, `[autocert.Manager]`, `[(autocert.Cache)]`, the "Provision and renew TLS certificate" scenario, the `@ACME` external actor, and the "TLS material" layer) from `arch-ingress.md` into the new chapter. The public listener in `arch-ingress.md` keeps its TLS-termination role and cross-references `arch-tls.md` for certificate lifecycle. Update `change.md` `## What`, `## How` Decisions, and the `## Technical design` checklist to list five files. Drop the Mermaid sequence diagram and the protocol Concepts section from the original TD because both deviate from the convention used by the other four arch files (ASCII flows; voedger components only); preserve protocol references in a `## Notes` section.

- Pros: stays inside the originally clarified "architecture only" scope; eliminates the inconsistency cleanly; gives TLS provisioning its own first-class subsystem chapter parallel to ingress / reverse-proxy / debug; aligns with the file-naming convention used by the other three arch chapters (`arch-{subsystem}.md`); removes the cross-file overlap that would otherwise exist between `tls--td.md` and `arch-ingress.md`
- Cons: loses the Mermaid sequence diagram and the protocol-background Concepts section the original TD carried; substantial rewrite of `arch-ingress.md` (removed `@ACME` actor, removed `[ACME HTTP-01 listener]`, removed "TLS material" layer, trimmed `[Public listener]` TLS sub-bullet to a cross-reference)
- Confidence: user-provided

Alternatives:

1. Convert to `arch-tls.md` but keep the Mermaid diagram and the Concepts section as exceptions
   - Pros: keeps the diagram and the protocol narrative; minimal content loss
   - Cons: dilutes the arch-file convention; `arch-tls.md` would look structurally different from the other arch files
   - Confidence: medium
2. Keep `tls--td.md` as-is and widen the change scope to include topic TDs
   - Pros: minimal edits; preserves the file unchanged
   - Cons: widens the originally clarified scope from pure architecture to mixed; mixes deliverable classes inside one change request
   - Confidence: high
3. Keep `tls--td.md` as-is and split the TLS work into a separate Change Folder
   - Pros: each change request stays focused
   - Cons: extra bureaucracy; the file is already on the current branch
   - Confidence: medium

## Inconsistency: `## Why` enumerates the routing-context capabilities but omits the per-workspace concurrent-query limit covered by `## What` and by `arch-ingress.md`

Decision: Add "the per-workspace concurrent-query limit" to the `## Why` capability enumeration between "BLOB and N10N endpoints" and "domain management". The query limit is a first-class capability documented in `## What` (`MaxQueriesPerWS`, backpressure rejections) and in `arch-ingress.md` (`### Reject excess concurrent query` scenario, `[Query limiter]` operator), so it belongs in the `## Why` scope list too.

- Pros: aligns `## Why` with `## What` and with the implemented chapter coverage; minimal edit (one inserted phrase); reviewers scanning `## Why` see the full scope
- Cons: lengthens the `## Why` sentence by one item
- Confidence: user-provided

Alternatives:

1. Drop the comma-separated capability list from `## Why` entirely — replace with a single phrase like "for the public HTTP boundary of voedger", letting `## What` enumerate capabilities authoritatively
   - Pros: shorter `## Why`; removes the risk of `## Why` drifting out of sync with `## What` again
   - Cons: weaker scoping signal in `## Why`; reviewers must click through to `## What` to gauge breadth
   - Confidence: medium
