# VaultPay — Product Requirements Document

**Wallet Infrastructure Platform**
Version 2.0 · Build-in-Public Edition
Author: Gracious Obeagu (Stay Liquid 💧) · July 2026 · Enugu, Nigeria

---

> **How to read this document.** This is a *teaching PRD*. It is not just a list of features — every non-trivial decision carries a **Why** block explaining the engineering reasoning from first principles, and many carry a **Post** block: a ready-to-publish hook for the *Stay Liquid* series. **Local** blocks tie the design to how money actually moves in Nigeria (NIBSS, NUBAN, CBN, NDPA).
>
> VaultPay is a **fictional, non-production** system built to be engineered to the standard a real Nigerian fintech would hold itself to. It never touches real money, real customer data, or a live bank rail.

---

## 0. What changed from v1.0 (and why)

v1.0 was structurally strong — ledger-first thinking, integer money, reversal-not-deletion. But a design review found several places where **two parts of the document promised incompatible things**. Those are exactly the bugs that are cheap to fix in a PRD and expensive to fix in code.

| # | v1.0 problem | v2.0 resolution |
|---|---|---|
| 1 | Idempotency scope contradicted itself: the key was described as unique per-user, globally, and per-endpoint in three places. It also lived on two tables. | One owner: the `idempotency_keys` table. Scope is `(actor_id, key)`. Same key + different payload ⇒ **422**. See §8.2. |
| 2 | The concurrency guard was ambiguous: the balance projection was both "not the source of truth" and "the row we lock." That is a designed-in double-spend. | Resolved explicitly: the balance row is the **write-serialization point** (pessimistic lock); the ledger is the source of truth for **history**. Updated atomically together. See §8.4. |
| 3 | `held_balance` (escrow) existed in the schema with **no lifecycle** — nothing ever put money into it or released it. | Full hold/release lifecycle modelled on NIP Deferred Net Settlement. See §8.5. |
| 4 | A row-level `CHECK` constraint was said to enforce "entries sum to zero" — **impossible**; a row CHECK cannot see sibling rows. | Replaced with a `DEFERRABLE` constraint trigger firing at COMMIT, plus an application-level assertion. See §9.3. |
| 5 | p95 latency targets were "under local load-test conditions" — unfalsifiable. | Targets now specify the exact load profile, data volume, and hardware. See §10. |
| 6 | "Writes succeed if the queue is down" asserted a guarantee with **no mechanism** — events would silently vanish. | **Transactional Outbox** added: events written to Postgres in the same transaction, relayed by a separate poller. See §5.4. |
| 7 | Redis did five jobs, two of them dangerous (idempotency storage; Redlock distributed locks) for a single monolith. | Redis scope cut to **caching + rate limiting only**. Idempotency lives in Postgres. Locking is Postgres row locks. See §5.3. |
| 8 | Phase 4 bundled fraud + KYC + multi-currency + event-driven rearchitecture — the point where a solo project quietly dies. | Split and re-sequenced. Multi-currency explicitly scoped (no cross-currency FX in MVP). See §17. |
| 9 | NDPR referenced as live law; `sessions` couldn't support the token-family revocation it promised; MinIO included for "mocked" uploads. | NDPR → **NDPA 2023 + GAID 2025**. `sessions` gains `token_family_id`. MinIO cut. See §11, §9, §16. |

---

## 1. Vision & Goals

### 1.1 Vision

VaultPay is a fictional wallet infrastructure platform: it lets businesses create digital wallets for their users, move money between wallets, and integrate payments through a documented API. It does not exist to acquire real users. It exists to be engineered to the same standard a licensed Nigerian fintech would hold itself to — **correctness, auditability, concurrency safety, fraud control, and observability** — reasoned from first principles rather than copied from a tutorial.

### 1.2 Goals

- **Engineering credibility** — a backend a fintech engineering manager could review and conclude the author understands how money movement actually has to work.
- **Teaching artifact** — the reasoning behind each decision *is* the product. Every milestone yields a write-up tied to one concrete lesson.
- **Depth over breadth** — fewer endpoints with real financial integrity, rather than 40 CRUD routes with none.
- **Locally grounded** — designed to reflect how money actually moves in Nigeria (NIBSS/NIP settlement semantics, NUBAN, kobo minor units, NDPA/CBN constraints), not a generic Stripe clone.

### 1.3 Non-Goals

- VaultPay will **not** process real money, real KYC/BVN documents, or connect to a live payment processor, NIBSS, or bank.
- VaultPay is **not** chasing production users or scale.
- VaultPay will **not** become a commercial product. Going commercial would trigger CBN licensing, NDPA registration, and a separate compliance program.

> **🇳🇬 Local.** A real VaultPay-like product in Nigeria would need a **CBN licence tied to function, not name**: a wallet issuer holding balances typically sits under a PSSP or Mobile Money / PSB authorisation, with capital requirements reaching ₦100m+ and NDPC data-controller registration. We build the *engineering* as if licensed, and treat the licensing itself as a documented non-goal.

---

## 2. Problem Statement

Most portfolio backends prove the author can wire CRUD endpoints to a database. They rarely prove the author understands the failure modes unique to money movement: **double-spends under concurrency, duplicate charges from retried requests, balances that silently drift from reality, and fraud vectors that only appear under adversarial use.**

VaultPay closes that gap — building a wallet system the way an experienced fintech team would, and documenting the reasoning so the process itself becomes the learning artifact.

> **📣 Post.** *"Most 'fintech' portfolio projects are a to-do list with a ₦ sign in front of the numbers. Here's the difference between a CRUD app and a system that actually moves money — and why the gap is entirely in the parts you can't see. 🧵💧"*

---

## 3. Target Users

### 3.1 In-product personas (fictional)

- **Business (API consumer)** — a company issuing wallets to its own customers or staff: a marketplace holding seller balances, a payroll platform holding employee wallets.
- **End user (wallet holder)** — an individual who deposits, sends, receives, and withdraws within the limits the business configures.
- **Admin / Ops** — an internal operator who reviews flagged transactions, manages limits, and can *read* (never mutate) the ledger and audit trail.

### 3.2 Real audience (the build-in-public series)

- **Backend / fintech engineers** reading for the design reasoning, not just the code.
- **Hiring managers & recruiters** evaluating the repo and write-ups as evidence of production-grade thinking.
- **The "Stay Liquid" audience** — developers primed for the hook → design breakdown → deep technical article format.

---

## 4. Scope

| Area | Description | Status |
|---|---|---|
| Auth & identity | Registration, login, JWT + rotating refresh tokens, RBAC, device binding | In scope |
| Wallet management | One wallet per currency per user; balances derived from ledger | In scope |
| Double-entry ledger | Append-only source of truth for all balances | In scope |
| Deposits, transfers, withdrawals | Core money-movement flows (single-currency only) | In scope |
| Held/escrow balance | Funds held during PENDING settlement or fraud review, modelled on NIP DNS | In scope |
| Transaction reversal | Compensating entries, never deletes | In scope |
| Idempotency | `Idempotency-Key` on all money-moving endpoints, stored in Postgres | In scope |
| Transactional outbox | Durable event publication that survives a queue outage | In scope |
| Webhooks (in + out) | Mock gateway → VaultPay; VaultPay → merchant, with signed retries | In scope |
| Fraud & risk rules | Velocity checks, limits, PIN lockouts, manual review queue | In scope |
| KYC workflow (mocked) | Tiers 1/2/3 driving limits; no real document/BVN verification | In scope |
| Audit logs | Immutable log of sensitive actions | In scope |
| Admin API + minimal UI | Ops review and configuration | In scope |
| Multi-currency (display + hold) | NGN primary, USD secondary. Wallets are per-currency; **no cross-currency transfer/FX in MVP** | In scope (scoped) |
| Real gateway / NIBSS / bank | Live Paystack / Flutterwave / NIP connection | Out of scope |
| Real KYC / BVN / NIN | Any real identity verification provider | Out of scope |
| CBN licensing & NDPA registration | Referenced conceptually; not implemented | Out of scope |
| Cross-currency FX conversion | Converting NGN↔USD inside a transfer | Out of scope (MVP) |
| Mobile app | Web/API only | Out of scope |
| Microservices from day one | Modular monolith first (§5) | Deferred |

> **💡 Why.** Multi-currency is the classic scope trap. "Support USD too" sounds small but silently drags in FX rates, rate-timing, and — fatally — **mixed-currency ledger entries that break the sum-to-zero invariant** (§9.3). We keep the teachable 90% (per-currency wallets, a currency column on every entry, USD as a demo) and cut the 10% that would consume a month. A ₦→$ transfer, if ever added, becomes **two** single-currency transactions bridged by an FX settlement account — a clean later write-up, not MVP risk.

---

## 5. System Architecture

### 5.1 Approach: modular monolith first

VaultPay is a single deployable service with strict internal module boundaries. Each domain (auth, wallets, ledger, transactions, fraud, notifications, admin) lives in its own package behind a clear interface, so a module can be extracted into its own service later **without a rewrite** — but that extraction only happens when there's a concrete reason.

> **💡 Why.** Monolith vs. microservices is one of the most over-copied, under-reasoned choices in junior-to-mid system design. A single developer with no real load who reaches for microservices pays the full distributed-systems tax (network partitions, distributed transactions, eventual consistency) to solve a problem they don't have. Strict package boundaries give ~90% of the modularity benefit and 0% of the operational cost. In Go, `internal/` makes this **compiler-enforced**, not aspirational.

> **📣 Post.** *"I didn't use microservices for my fintech project. Here's the one-sentence reason — and why reaching for them would have been the junior move, not the senior one. 💧"*

### 5.2 High-level components

- **API layer (Go + Fiber)** — REST endpoints, validation, auth & rate-limit middleware.
- **Domain / service layer** — business logic per domain, behind interfaces.
- **Ledger engine** — the accounting core; every balance change flows through it, no exceptions.
- **PostgreSQL** — system of record: the append-only ledger, the balance projection, the idempotency store, and the outbox.
- **Redis** — caching and rate limiting **only** (see §5.3).
- **Queue (RabbitMQ)** — background jobs and domain events, fed by the outbox relay.
- **Worker processes** — notifications, webhook delivery, report generation.
- **Observability** — Prometheus + Grafana, OpenTelemetry, structured JSON logs with correlation IDs.

### 5.3 What Redis deliberately does NOT do

- **Idempotency storage → moved to Postgres.** An idempotency key guards a money movement. If Redis evicts the key (memory pressure, restart) while the transfer is committed in Postgres, the dedupe record and the money disagree, and a retry double-spends. The idempotency record must be **transactional with the ledger write**, which means it lives in the same database.
- **Distributed locks (Redlock) → removed entirely.** Redlock's correctness is contested, and a single-instance monolith doesn't need cross-node locks. Postgres row-level locks (§8.4) are simpler, correct, and already transactional.

> **💡 Why.** The rule: **anything that must be consistent with money lives in Postgres; Redis only holds data you can safely lose and rebuild.** Losing a rate-limit counter costs nothing; losing an idempotency key costs a customer a duplicate transfer. Matching the storage's durability guarantee to the data's consequence-of-loss is the actual skill.

> **📣 Post.** *"I deleted Redis from my payment idempotency layer. Not because Redis is bad — because I was using it for the one job it must never do. Here's the failure sequence that would double-charge a customer. 🧵"*

### 5.4 The Transactional Outbox

v1.0 claimed "writes succeed even if the queue is down." True — but if you publish `transfer.completed` to RabbitMQ *after* committing to Postgres, and the queue is down at that moment, **the money moved but the notification, webhook, and analytics never fire.** The event is lost. This is the classic **dual-write problem**.

**Solution:**

1. Inside the same DB transaction that posts the ledger entries, also **INSERT the event row into an `outbox_events` table.** One commit, both or neither.
2. A separate **relay poller** reads unpublished rows, publishes them to RabbitMQ, and marks them published — retrying on failure.
3. Consumers are made **idempotent** so a re-delivered event is harmless.

> **💡 Why.** This converts an unreliable *dual write* (DB + queue, which can partially fail) into a reliable *single write* (DB only) plus an *asynchronous relay*. If the queue is down, events pile up durably in Postgres and drain when it recovers — nothing is lost, only delayed. This is precisely the "graceful degradation" v1.0 asserted but didn't build.

> **📣 Post.** *"Your database write succeeded. Your queue was down. The money moved but the receipt never sent. This is the dual-write problem, and here's the ~15-line table that fixes it. 💧"*

### 5.5 Request flow — a transfer

1. Client sends `POST /v1/transfers` with an `Idempotency-Key` header.
2. API authenticates, validates, and checks the idempotency store. Replay of a completed key ⇒ return the stored response, do nothing else.
3. Fraud engine runs **synchronous** checks (limits, velocity, PIN attempts). Fail ⇒ `FAILED` or `PENDING_REVIEW`, no ledger entries.
4. Ledger engine opens a DB transaction, **locks the sender's balance row with `SELECT … FOR UPDATE`**, verifies sufficiency, posts debit + credit entries, updates the balance projection, and writes the outbox event — **all in one atomic commit.**
5. Transaction status → `SUCCESS` (or `FAILED`, everything rolled back).
6. The outbox relay later publishes `transfer.completed`; consumers react independently and idempotently.
7. API responds. The idempotency store now returns this exact response for any retry of the same key.

---

## 6. Business Domains

| Domain | Responsibility |
|---|---|
| Authentication & Identity | Registration, login, JWT/refresh token families, sessions, device binding |
| Users | Profile, roles, status, KYC tier |
| Wallets | Wallet lifecycle, per-currency balances (derived), available vs. held |
| Ledger | Double-entry accounting core; source of truth for money history |
| Transactions | Deposits, transfers, withdrawals, reversals, explicit state machine |
| Payments | Mock gateway & bank integration, mock virtual accounts |
| KYC | Tiered verification status and the limits each tier unlocks |
| Fraud & Risk | Rule engine, velocity checks, freezes, manual review queue |
| Notifications | Queued email/SMS via mocked providers, driven by outbox events |
| Audit Logs | Immutable record of sensitive actions across all domains |
| Admin | Internal endpoints + minimal UI for ops review |

---

## 7. Functional Requirements

Endpoints are representative, not exhaustive. Full schemas live in the OpenAPI/Scalar spec (§13). This document defines **behaviour and rules**; the spec defines exact contracts.

### 7.1 Authentication & Identity

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `POST /v1/auth/register` | Create account | Email/phone uniqueness; password policy; emits `UserRegistered` via outbox |
| `POST /v1/auth/login` | Authenticate, issue tokens | Device fingerprint + IP captured; rate-limited; new device may require step-up |
| `POST /v1/auth/refresh` | Rotate access token | Refresh-token reuse detection revokes the entire `token_family` |
| `POST /v1/auth/logout` | Revoke session | Revokes refresh token; idempotent |

> **🇳🇬 Local.** CBN's 2026 direction is explicit: **passwords and OTP alone are no longer considered sufficient**; digital identity should be tied to trusted devices. VaultPay's device-binding and new-device step-up mirror that, even though our device "fingerprint" is mocked.

### 7.2 Wallets

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `POST /v1/wallets` | Create a wallet | One wallet per currency per user; KYC tier gate |
| `GET /v1/wallets/:id` | Wallet detail | Balance read from the projection, never a hand-edited column |
| `GET /v1/wallets/:id/balance` | Current balance | Distinguishes available vs. held (escrow) |

### 7.3 Transactions

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `POST /v1/deposits` | Fund a wallet (mock gateway) | `Idempotency-Key`; creates PENDING + held funds until gateway confirms |
| `POST /v1/transfers` | Move money between wallets | `Idempotency-Key`; fraud checks; atomic debit+credit; daily limits |
| `POST /v1/withdrawals` | Move money out (mock) | PIN + OTP above threshold; funds held pending settlement |
| `POST /v1/transactions/:id/reverse` | Reverse a transaction | Compensating entries; never deletes history |
| `GET /v1/transactions` | History | Paginated; filter by status/date/type |

### 7.4 Ledger (internal + admin-read)

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `GET /v1/admin/ledger/entries` | Raw ledger entries | Admin/ops only; append-only, never mutated |
| `POST /v1/admin/ledger/reconcile` | Trigger reconciliation | Recomputes balances from entries; flags drift |

### 7.5 Webhooks

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `POST /webhooks/gateway` | Inbound from mock gateway | HMAC verify; timestamp check; replay protection; idempotent |
| Outbound events | VaultPay → merchant callback | Signed; retries with exponential backoff; delivery logged |

### 7.6 Fraud & Risk

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `POST /v1/admin/fraud/rules` | Configure a rule | e.g. `transfer > daily_limit ⇒ require_approval` |
| `GET /v1/admin/fraud/flags` | Review flagged txns | Approve/reject moves txn out of `PENDING_REVIEW`; releases or reverses held funds |

### 7.7 KYC

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `POST /v1/kyc/submit` | Submit mock verification | No real document checks; transitions simulated |
| `GET /v1/kyc/status` | Current tier + limits | Tier directly drives wallet transaction limits |

### 7.8 Admin

| Method & Path | Purpose | Key rules / edge cases |
|---|---|---|
| `GET /v1/admin/users/:id` | User + wallet overview | RBAC-restricted to ops/admin |
| `POST /v1/admin/wallets/:id/freeze` | Freeze a wallet | Logged to audit trail with actor + reason |

---

## 8. Money Movement & the Ledger

This is the heart of the project — the part most portfolio projects skip.

### 8.1 Double-entry accounting

**No endpoint or service is ever permitted to run `balance -= amount` directly.** Every movement of money creates at least **two ledger entries — a debit and a credit — that net to zero.** A ₦5,000 transfer from Wallet A to Wallet B posts a 5,000 debit against A and a 5,000 credit to B, both inside one atomic operation.

- **Balances are derived, not stored as truth** — a wallet's balance is a projection of its ledger entries, refreshed on write and rebuildable from scratch at any time.
- **All money is integers in the smallest unit (kobo/cents)** — never floating point, to avoid rounding drift.

> **🇳🇬 Local.** Nigeria's currency has 100 kobo to ₦1. VaultPay stores ₦5,000 as the integer `500000` (kobo). This isn't pedantry: floating-point ₦ arithmetic drifts, and in a ledger, a fraction of a kobo that doesn't reconcile is a **failed audit**. Paystack, Flutterwave, and NIBSS all transact in kobo integers for the same reason.

> **📣 Post.** *"Why does every real fintech store ₦5,000 as the number 500000? Because 0.1 + 0.2 ≠ 0.3 in a computer — and in a ledger, that rounding error is money that vanishes. 💧"*

### 8.2 Idempotency

Every money-moving endpoint requires an `Idempotency-Key` header.

- **Single owner:** the `idempotency_keys` table in Postgres. The key is **not** duplicated onto `transactions`.
- **Scope:** unique on `(actor_id, key)`.
- **Stored with a `request_hash`:** a SHA-256 of the canonical request body.
- **Retry, same key + same hash:** return the stored response verbatim. Do nothing else.
- **Retry, same key + _different_ hash:** respond **`422 Unprocessable Entity`** — the client reused a key for a different request, which is a client bug we must surface, not silently execute.

> **💡 Why.** The killer question is the **mismatched-payload retry**. Silently executing it double-spends; silently returning the old response hides a client bug and refunds the wrong person. Returning 422 is the industry answer (Stripe does exactly this) because it makes the client's mistake **loud and safe** instead of quiet and expensive.

> **📣 Post.** *"Same idempotency key, different request body. What should your API do? There's exactly one safe answer, and most tutorials get it wrong. 🧵💧"*

### 8.3 Transaction lifecycle

| State | Meaning |
|---|---|
| `PENDING` | Awaiting external confirmation. Funds may be held. |
| `PROCESSING` | Internally being posted to the ledger. |
| `PENDING_REVIEW` | Held by the fraud engine for manual approve/reject. |
| `SUCCESS` | Ledger entries posted, balances updated, held funds released. |
| `FAILED` | No ledger entries posted, or rolled back; any hold released. |
| `REVERSED` | Compensating entries posted against a prior SUCCESS. |
| `EXPIRED` | A PENDING transaction that timed out; hold released. |

### 8.4 Concurrency safety — the core decision

**The failure we are preventing.** Wallet A has ₦5,000. Two ₦5,000 transfers arrive at the same instant. Both read the balance, both see ₦5,000, both post to the ledger — A is now ₦-5,000. **This double-spend is the single failure mode fintech exists to prevent.**

**The resolution — two roles, cleanly separated:**

- **The ledger is the source of truth for _history_.** Append-only, rebuildable, audited. You never lock it to make a decision — an append-only log has no natural "current balance" row to lock.
- **The `wallet_balances` row is the source of authority for _concurrency_.** It is a **write-serialization point**, protected by a pessimistic `SELECT … FOR UPDATE`.
- **They are updated in the _same_ DB transaction**, so at commit time they can never disagree.

```sql
BEGIN;
  -- 1. serialize on the sender's balance row
  SELECT available_balance, version
    FROM wallet_balances
   WHERE wallet_id = :sender FOR UPDATE;   -- 2nd txn blocks here

  -- 2. check sufficiency (in app code)
  --    if available < amount -> ROLLBACK, return 422

  -- 3. append the two ledger entries (debit + credit)
  INSERT INTO ledger_entries (...) VALUES (debit), (credit);

  -- 4. move the projection
  UPDATE wallet_balances SET available_balance = available_balance - :amt,
         version = version + 1 WHERE wallet_id = :sender;
  UPDATE wallet_balances SET available_balance = available_balance + :amt
         WHERE wallet_id = :receiver;

  -- 5. write the outbox event (same txn!)
  INSERT INTO outbox_events (...) VALUES (transfer_completed);
COMMIT;   -- deferred sum-to-zero trigger fires here (see 9.3)
```

The second transfer **blocks** at the `FOR UPDATE` until the first commits, then reads the *new* balance (₦0) and fails cleanly. No double-spend is possible.

> **💡 Why.** Pessimistic locking (block-and-wait) is chosen over optimistic (version-check-and-retry) for the MVP because it is **impossible to get subtly wrong**: no retry loop, no lost-update edge case. Its only cost is that transfers on the *same* wallet serialize — a non-issue for any realistic wallet. The `version` column is kept so the **optimistic alternative can be benchmarked in a branch and written up**, not shipped.

> **📣 Post.** *"Two requests. Same ₦5,000. Here's exactly how your wallet goes negative — and the one line of SQL (`SELECT … FOR UPDATE`) that stops it. Your balance and your transaction history are two different things with two different jobs. 🧵💧"*

### 8.5 Held / escrow balance lifecycle

> **🇳🇬 Local.** NIBSS Instant Payments (NIP) is a **Deferred Net Settlement** system: the beneficiary sees the funds *immediately*, but actual settlement between institutions happens later, in one of ~12 daily sessions. There is a real window where money is "available to the user but not yet settled." **That window *is* the held balance.** Modelling it teaches the single most important thing about Nigerian payments: **availability and settlement are not the same event.**

**Every hold is two ledger entries, and every release/capture is two more:**

1. **Deposit initiated** — debit a `gateway_clearing` account, credit the wallet's **held** balance. Transaction → `PENDING`.
2. **Gateway confirms** (webhook) — move funds from **held** to **available** (release). Transaction → `SUCCESS`.
3. **Gateway fails / times out** — reverse the hold with compensating entries. Transaction → `FAILED` / `EXPIRED`.

Withdrawals and fraud-review holds follow the same debit-to-held / release-or-reverse pattern. **Held balance is never a mutable counter** — it is always the sum of unreleased hold entries in the ledger, so it reconciles like everything else.

> **📣 Post.** *"In Nigeria, 'the money is in your account' and 'the money has settled' are two different moments. That gap has a name (Deferred Net Settlement) and here's how I modelled it as escrow in a double-entry ledger. 💧"*

### 8.6 Reversal, not deletion

A reversal never edits or removes a ledger entry. It posts a new, opposite entry referencing the original transaction, preserving a complete auditable history — including mistakes and their corrections.

> **💡 Why.** An append-only ledger is the difference between "we think the balance is X" and "we can **prove** every kobo of the balance from first principles." Deleting or editing an entry destroys that proof and is, in a real fintech, the kind of thing that ends careers and triggers regulators.

---

## 9. Database Design

| Table | Purpose | Key columns |
|---|---|---|
| `users` | Identity, credentials, role, KYC tier | `id, email, phone, password_hash, role, kyc_tier, created_at` |
| `wallets` | One row per user per currency | `id, user_id, currency, status, created_at` |
| `ledger_entries` | Append-only double-entry records; source of truth | `id, transaction_id, wallet_id, direction, entry_kind, amount, currency, created_at` |
| `wallet_balances` | Projection for fast reads + concurrency gate | `wallet_id, available_balance, held_balance, updated_at, version` |
| `transactions` | One row per user-facing money movement | `id, type, status, amount, currency, source_wallet_id, dest_wallet_id, created_at` |
| `idempotency_keys` | Dedupe store (sole owner of the key) | `actor_id, key, request_hash, response_body, status, created_at, expires_at` |
| `outbox_events` **(v2)** | Durable event publication log | `id, aggregate_id, event_type, payload, published, created_at, published_at` |
| `webhooks_outbound` | Delivery log for merchant callbacks | `id, event_type, payload, target_url, status, attempts, last_attempt_at` |
| `webhooks_inbound` | Log of received gateway events | `id, provider, signature, payload, verified, processed_at` |
| `fraud_rules` | Configurable rule definitions | `id, name, condition, action, enabled` |
| `fraud_flags` | Transactions held for review | `id, transaction_id, rule_id, status, reviewed_by, reviewed_at` |
| `audit_logs` | Immutable action trail | `id, actor_id, action, entity, entity_id, ip_address, device_id, created_at` |
| `sessions` | Refresh-token sessions | `id, user_id, token_family_id, device_id, ip_address, revoked, created_at` |

### 9.1 Key relationships & constraints

- `ledger_entries.transaction_id` → `transactions.id`: every entry traces to the transaction that produced it.
- `wallet_balances` is derived: it can be dropped and rebuilt entirely from `ledger_entries` — **a documented recovery drill.**
- `idempotency_keys` has a unique constraint on `(actor_id, key)` with a TTL expiry job.
- `sessions.token_family_id` **(v2)**: groups a refresh-token lineage so reuse detection can revoke the whole family.

### 9.2 Indexing

- `wallet_balances(wallet_id)` — unique, O(1) balance lookups.
- `ledger_entries(wallet_id, created_at)` — statement/history queries.
- `idempotency_keys(actor_id, key)` — unique, fast dedupe.
- `outbox_events(published, created_at)` — the relay poller scans unpublished rows.
- `audit_logs(actor_id, created_at)` and `(entity, entity_id)`.

### 9.3 The sum-to-zero invariant

A row-level `CHECK` **cannot see sibling rows**, so it literally cannot sum across a transaction's entries. Two enforcement layers replace it:

1. A **`DEFERRABLE INITIALLY DEFERRED` constraint trigger** that fires at `COMMIT` and rejects the transaction if the signed sum of entries for any `transaction_id` (per currency) is non-zero.
2. An **application-level assertion** inside the posting function, so the invariant fails fast in tests and code review.

> **💡 Why.** Deferring to COMMIT is the key trick: mid-transaction the entries are legitimately unbalanced (the debit is in, the credit isn't yet), so an immediate check would false-positive. Enforcing the invariant in **both** the DB and the app is defence-in-depth: the app catches it in milliseconds during development; the database guarantees it can never be violated even by a future buggy code path.

> **📣 Post.** *"'Just add a CHECK constraint that the ledger entries sum to zero.' You can't — a row CHECK can't see the other rows. Here's what actually enforces double-entry integrity in Postgres. 🧵"*

---

## 10. Non-Functional Requirements

| Category | Requirement |
|---|---|
| Security | JWT + rotating refresh tokens, RBAC, PIN + OTP for sensitive actions, per-user/IP rate limiting, secrets encrypted at rest |
| **Performance (defined)** | p95 < 200 ms reads, p95 < 500 ms transfer posting, measured at **200 concurrent virtual users against 1,000,000 seeded ledger rows on the GitHub Actions CI runner (2 vCPU / 7 GB)**, via a committed k6 script |
| Availability | Health checks on API + workers; graceful degradation via the outbox — writes succeed if the queue is down, async side-effects delayed, never lost |
| Scalability | Modular monolith so any domain can be extracted without a data-model rewrite |
| Consistency | Ledger postings atomic and never partial; balances always reconcilable against entries |
| Monitoring | Prometheus metrics: request latency, queue depth, outbox lag, reconciliation drift |
| Logging | Structured JSON logs with a correlation/trace ID from inbound request through every async job |
| Backup & recovery | Nightly Postgres backups; documented, tested restore; the ledger-rebuild drill |

> **💡 Why.** "Measure, don't target" is the honest stance: you cannot promise a p95 you haven't load-tested, and inventing one is exactly the overstated claim a senior reviewer smells instantly. Naming the profile turns the number into a **reproducible experiment** — a reviewer can clone the repo, run the committed k6 script, and see the same figure. That reproducibility is worth more than a bigger number.

---

## 11. Security & Data Protection

- **Authentication** — short-lived JWT access tokens; rotating refresh tokens; reuse detection that revokes an entire `token_family` if a used refresh token is replayed.
- **Authorization** — RBAC (user, business, admin, ops) enforced in middleware, not scattered through handlers.
- **Transaction security** — PIN required for withdrawals and transfers above a configurable threshold; OTP as second factor; 5 failed PIN attempts freezes the wallet pending review.
- **Rate limiting** — per-IP and per-user on auth and money-movement endpoints.
- **Audit trail** — every sensitive action written immutably with actor, IP, device, timestamp.
- **Device & IP binding** — sessions tied to a device fingerprint; new-device logins flagged for step-up.
- **Webhook security** — inbound HMAC-verified and timestamp-checked against replay; outbound signed.

### 11.1 Data protection — NDPA 2023 + GAID 2025

The **NDPR is no longer the governing instrument.** As of September 2025 the **Nigeria Data Protection Act (NDPA) 2023**, read with the **General Application and Implementation Directive (GAID) 2025**, is the live framework, enforced by the NDPC.

- **Breach-notification readiness** — the audit log and structured logging are designed so a breach could be reconstructed and reported within the NDPA's **72-hour** window (a documented tabletop drill, not a live obligation).
- **Data minimisation & purpose** — no real PII is collected; mocked KYC data is clearly synthetic. A real build would need a lawful basis and a **DPIA** for fraud-scoring/automated-decision flows.
- **Data-subject rights** — access/rectification/erasure noted as design constraints, not implemented as a compliance program.

> **🇳🇬 Local.** This matters commercially, not just legally: the NDPC has already levied penalties in the **₦-hundred-millions to $220m** range and has called 2026 an enforcement year. Knowing **which** law applies (NDPA, not the retired NDPR) is itself a senior signal.

> **📣 Post.** *"Half the 'Nigerian fintech compliance' blog posts online still cite the NDPR. It stopped being the law in Sept 2025. Here's what actually governs your users' data now — and the 72-hour clock you're on if you're breached. 💧"*

---

## 12. Integrations

All external integrations are **mocked**. Each mock behaves like the real thing, **including realistic failure modes**, so the engineering problems are genuine even though the providers are not.

| Integration | Mocked behaviour |
|---|---|
| Payment gateway | Simulated deposit confirmation with configurable delay and failure rate; sends signed webhook callbacks. Models a Paystack/Flutterwave-style flow |
| Bank API / virtual accounts | Generates mock 10-digit **NUBAN** virtual account numbers; simulates inbound bank-transfer notifications with NIP-style deferred settlement timing |
| OTP provider | Generates and validates OTPs; logs instead of sending real SMS |
| Email provider | Queued email jobs rendered to a preview endpoint instead of sent |

---

## 13. Engineering Standards & Conventions

### 13.1 API standards
- Versioned routes under `/v1`, with a documented deprecation policy.
- Consistent error shape `{ error: { code, message, details } }` across every endpoint.
- OpenAPI/Scalar spec maintained alongside code as the single source of truth.

### 13.2 Code & repo conventions
- Standard Go layout (`cmd/`, `internal/`), each business domain its own `internal` package.
- Trunk-based git with short-lived feature branches — one branch per milestone item.
- Conventional commits (`feat:`, `fix:`, `refactor:`, `docs:`).

### 13.3 Data & migrations
- Versioned, **reversible** SQL migrations (golang-migrate); **no manual schema edits** against any environment.
- Every new table/column ships with a comment explaining its purpose.

### 13.4 Events & logging
- Event naming `<domain>.<event>`, past tense — `transfer.completed`, `wallet.frozen`.
- Structured logs carry a correlation ID from the inbound request through every async job.

---

## 14. Testing Strategy

| Test type | Focus |
|---|---|
| Unit | Ledger posting logic, fraud rule evaluation, state-machine transitions, idempotency hash-compare |
| Integration | API endpoints against real PostgreSQL via Testcontainers |
| Concurrency | Simultaneous transfers on the same wallet under `go test -race`, asserting no negative balances or lost updates |
| Property / invariant | Randomised sequences of transfers, asserting sum-to-zero and available+held reconciles to entries every time |
| Load | Transfer throughput and latency via the committed k6 script (§10) |
| Reconciliation | Rebuild `wallet_balances` from `ledger_entries` and diff against live balances |

> **💡 Why.** The concurrency and reconciliation tests are where this project earns its "fintech" label. Anyone can test that `POST /transfers` returns 200. Proving that **100 simultaneous transfers can never produce a negative balance**, and that the balance projection can be **rebuilt from the ledger and still match**, is the evidence a reviewer actually looks for.

> **📣 Post.** *"I wrote a test that fires 100 transfers at the same wallet simultaneously and asserts the balance can never go negative. It failed the first time. Here's the bug — and why `-race` is non-negotiable for money code. 🧵💧"*

---

## 15. Deployment & Observability

- Docker for local dev; `docker-compose` for the full local stack.
- CI/CD via GitHub Actions: lint, `go vet`, unit + integration tests under `-race`, build, deploy on merge to main.
- Config via `.env` locally, injected by the deploy target later; **secrets never committed.**
- Health endpoints `/healthz` (liveness) and `/readyz` (readiness) for API and every worker.
- Prometheus + Grafana for latency, error rate, queue depth, **outbox lag**, and **reconciliation drift**; OpenTelemetry tracing across API → ledger → outbox → queue → worker.

---

## 16. Technology Stack

| Layer | Choice | Introduced in |
|---|---|---|
| Language | Go | Phase 1 |
| Web framework | Fiber | Phase 1 |
| Database | PostgreSQL (pgx, sqlc, golang-migrate) | Phase 1 |
| Cache / rate limiting | Redis (**only** these two jobs — §5.3) | Phase 1 |
| Idempotency & outbox store | PostgreSQL (**not** Redis) | Phase 2 |
| Queue | RabbitMQ | Phase 3 |
| Auth | JWT + rotating refresh tokens | Phase 1 |
| API docs | OpenAPI / Scalar | Phase 1 |
| Metrics & dashboards | Prometheus + Grafana | Phase 3 |
| Tracing | OpenTelemetry | Phase 3 |
| Load testing | k6 (committed script) | Phase 2 |
| Integration testing | Testcontainers | Phase 1 |
| Internal service comms | gRPC (only if a module is ever extracted) | Deferred |
| Containerization | Docker | Phase 1 |

---

## 17. Roadmap & Milestones

**Each phase ends with a working, demonstrable, write-up-able system. The hardest single feature never shares a phase with another hard feature.**

| Phase | Delivers | Candidate write-ups |
|---|---|---|
| **Phase 1 — Core Wallet** | Auth (+ token families, device binding), users, wallet creation, deposits, transfers, withdrawals, history. Single currency (NGN). | "Why fintechs never update a balance column directly" |
| **Phase 2 — Financial Correctness** | Double-entry ledger, integer money, idempotency (Postgres), `FOR UPDATE` concurrency, held/escrow lifecycle, sum-to-zero trigger, reconciliation job, k6 load test. | "Two requests, same ₦5,000"; "Same idempotency key, different body — now what?"; "Optimistic vs pessimistic locking, benchmarked" |
| **Phase 3 — Production Concerns** | Transactional outbox, RabbitMQ, webhooks (in/out, signed, retried), audit logs, structured logging, Prometheus/Grafana/OTel. | "The dual-write problem and the outbox that fixes it"; "Building a webhook retry system"; "What our audit log actually protects us from" |
| **Phase 4 — Fraud & Risk** | Fraud rules engine, velocity checks, PIN lockouts, manual review queue. | "Designing a fraud rules engine from scratch" |
| **Phase 5 — KYC & Limits** | Mocked KYC tiers 1/2/3, tier-driven limits, wallet freezes. | "How KYC tier drives every limit in the system" |
| **Phase 6 — Multi-currency (scoped)** | USD as a second per-currency wallet; per-currency reconciliation. **No cross-currency FX.** | "Multi-currency without the FX rabbit hole" |

> **💡 Why.** The re-sequencing follows one rule: **never put two hard, independent problems in the same phase.** Concurrency safety (P2) and the outbox (P3) are each a full mental model; shipping them together means neither gets done well. Each phase leaves you with something you can **demo and post about** — which is the whole point of build-in-public.

*Each phase closes with a short retro — what worked, what got redesigned, what the next write-up covers.*

---

## 18. Build-in-Public Content Plan

Every shipped feature gets a write-up before the next one, in the three-tier format:

- **Hook** — a short, punchy claim or scenario.
- **Design breakdown** — the decision and the trade-off, for a platform-native audience.
- **Technical article** — the implementation with real code.

**The backlog, pulled from the Post blocks above:**

1. CRUD app vs. money-moving system — the invisible difference (§2)
2. Why I skipped microservices, in one sentence (§5.1)
3. The one job Redis must never do (§5.3)
4. The dual-write problem and the outbox table that fixes it (§5.4)
5. Why ₦5,000 is stored as `500000` (§8.1)
6. Same idempotency key, different body — the only safe answer (§8.2)
7. Two requests, same ₦5,000: the double-spend and the one-line fix (§8.4)
8. Available vs. settled: modelling NIP deferred settlement as escrow (§8.5)
9. Why "sum to zero" can't be a row CHECK constraint (§9.3)
10. The p95 number you can actually reproduce (§10)
11. The NDPR is dead; here's what governs Nigerian user data now (§11.1)
12. 100 simultaneous transfers, one balance, zero negatives (§14)
13. Optimistic vs pessimistic locking, benchmarked (§17, P2)

### 18.1 Success signal

By the time VaultPay is public, a reviewer landing on the repo and the write-up series should come away thinking: **this developer understands the problems financial systems actually have to solve** — not just that endpoints exist, but why each one is built the way it is, and how those choices map to the Nigerian ecosystem it would really run in.

---

*Keep your code fluid and your dreams even bigger. Stay Liquid 💧*
