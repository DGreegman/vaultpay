# VaultPay

**A wallet infrastructure platform, built to be engineered — not shipped.**

VaultPay is a **fictional** wallet system: businesses create digital wallets for their users, move money between them, and integrate payments through a documented API. It does not exist to acquire users. It exists to be built to the standard a real Nigerian fintech would hold itself to — correctness, auditability, concurrency safety, fraud control, and observability — reasoned from first principles rather than copied from a tutorial.

> **This project handles no real money, no real customer data, and connects to no live bank or payment rail.** Every external integration is mocked. See [SECURITY.md](SECURITY.md).

---

## Why this exists

Most portfolio backends prove the author can wire CRUD endpoints to a database. They rarely prove the author understands the failure modes unique to systems that move money:

- **Double-spends** under concurrency
- **Duplicate charges** from retried requests
- **Balances that silently drift** from reality
- **Fraud vectors** that only appear under adversarial use

VaultPay is built to close that gap, and to document the reasoning behind each decision so the *process* becomes the artifact — not just the finished code.

---

## The core idea, in one paragraph

**No code is ever permitted to run `balance -= amount`.** Every movement of money creates at least two ledger entries — a debit and a credit — that net to zero. A wallet's balance is not a column you edit; it is a **projection** of its ledger entries, rebuildable from scratch at any time. The ledger is append-only and is the source of truth for *history*. A separate balance row is the source of authority for *concurrency*, protected by a pessimistic row lock. They are written in the same database transaction, so they can never disagree.

That distinction — history vs. concurrency, two roles, two mechanisms — is the spine of the whole system.

---

## Key engineering decisions

Each of these is a deliberate choice with a documented rationale. The full reasoning lives in the [PRD](docs/PRD.md).

| Decision | Why |
|---|---|
| **Double-entry ledger, append-only** | Balances can be *proven* from first principles, not merely believed. Reversals post compensating entries; nothing is ever edited or deleted. |
| **Money as integers (kobo/cents)** | ₦5,000 is stored as `500000`. Floating-point arithmetic drifts, and in a ledger a fraction of a kobo that doesn't reconcile is a failed audit. |
| **Pessimistic locking (`SELECT … FOR UPDATE`)** | Two simultaneous transfers against the same wallet must never both succeed against a balance that can't cover both. The blocked transaction waits, then fails cleanly. Impossible to get subtly wrong — unlike a retry loop. |
| **Idempotency in Postgres, not Redis** | An idempotency key guards a money movement. It must commit atomically with the ledger write. If Redis evicts the key, the dedupe record and the money disagree — and a retry double-spends. |
| **Transactional outbox** | Writing to the DB *and* publishing to a queue is a dual write that can partially fail. Events are written to Postgres in the same transaction and relayed asynchronously, so a queue outage delays side-effects but never loses them. |
| **Modular monolith, not microservices** | One developer, no real load. Strict package boundaries (enforced by Go's `internal/`) give ~90% of the modularity benefit at 0% of the distributed-systems cost. Extraction stays cheap if it's ever justified. |
| **Liveness ≠ readiness** | `/healthz` never checks the database. If it did, a Postgres blip would make an orchestrator restart every API container — turning a dependency outage into a crash loop. `/readyz` checks dependencies and returns 503; traffic stops, the container lives. |

---

## Built for Nigeria, specifically

The design is grounded in how money actually moves here, not in a generic Stripe clone.

- **Kobo integers** — the same convention NIBSS, Paystack, and Flutterwave transact in.
- **Held / escrow balance modelled on NIP** — NIBSS Instant Payments is a *Deferred Net Settlement* system: the beneficiary sees funds immediately, but settlement between institutions happens later, in one of ~12 daily sessions. That window — *available but not yet settled* — is the held balance. Availability and settlement are not the same event, and the ledger models both.
- **Mock NUBAN virtual accounts** — 10-digit, the real format.
- **NDPA 2023 + GAID 2025** — the current data-protection framework (the NDPR was retired in September 2025). Breach-notification readiness, data minimisation, and DPIA-triggering automated decisions are treated as design constraints.
- **CBN direction on device binding** — passwords and OTP alone are no longer considered sufficient; sessions are tied to a device fingerprint.

---

## Architecture

```
                  ┌─────────────┐
   HTTP ─────────▶│  API (Fiber)│
                  └──────┬──────┘
                         │
                  ┌──────▼──────┐
                  │   Domain    │  auth · wallets · transactions
                  │   Services  │  fraud · kyc · admin
                  └──────┬──────┘
                         │
                  ┌──────▼──────┐
                  │   Ledger    │  every balance change flows through here
                  │   Engine    │
                  └──────┬──────┘
                         │
        ┌────────────────▼────────────────┐
        │          PostgreSQL             │
        │  ledger_entries (append-only)   │
        │  wallet_balances (projection)   │
        │  idempotency_keys               │
        │  outbox_events                  │
        └────────────────┬────────────────┘
                         │  relay poller
                  ┌──────▼──────┐
                  │  RabbitMQ   │──▶ workers: notifications, webhooks
                  └─────────────┘
```

**Redis** does exactly two things: caching and rate limiting. It deliberately holds *nothing* that must be consistent with money.

---

## Stack

| Layer | Choice |
|---|---|
| Language | Go 1.22 |
| Web framework | Fiber |
| Database | PostgreSQL 16 (pgx, sqlc, golang-migrate) |
| Cache / rate limiting | Redis |
| Queue | RabbitMQ *(Phase 3)* |
| Observability | Prometheus, Grafana, OpenTelemetry *(Phase 3)* |
| Testing | Testcontainers, k6 |

The stack grows with the roadmap. Nothing is added until a phase gives a real reason.

---

## Getting started

**Prerequisites:** Go 1.22+, Docker, [golang-migrate](https://github.com/golang-migrate/migrate), [air](https://github.com/air-verse/air) (optional, for live reload).

```bash
git clone git@github.com:DGreegman/vaultpay.git
cd vaultpay

cp .env.example .env      # adjust ports if 5433/6380 are taken
make setup                # docker up + wait + migrate
make dev                  # run with live reload
```

Verify:

```bash
curl localhost:8080/healthz   # {"status":"ok"}
curl localhost:8080/readyz    # {"status":"ready"}
```

Run `make help` to see every available target.

---

## Roadmap

Each phase ends with a working, demonstrable system. No phase bundles two hard, independent problems.

- [x] **Phase 0 — Scaffolding.** Module layout, config, server, Docker, migrations, connection pool.
- [ ] **Phase 1 — Core Wallet.** Auth (token families, device binding), users, wallets, deposits, transfers, withdrawals, history.
- [ ] **Phase 2 — Financial Correctness.** Double-entry ledger, idempotency, `FOR UPDATE` concurrency, held/escrow lifecycle, sum-to-zero enforcement, reconciliation, load tests.
- [ ] **Phase 3 — Production Concerns.** Transactional outbox, queue, signed webhooks with retries, audit logs, observability.
- [ ] **Phase 4 — Fraud & Risk.** Rules engine, velocity checks, PIN lockouts, manual review queue.
- [ ] **Phase 5 — KYC & Limits.** Mocked tiers driving transaction limits, wallet freezes.
- [ ] **Phase 6 — Multi-currency.** USD as a second per-currency wallet. *No cross-currency FX* — explicitly out of scope.

---

## Build in public

This project is being built in the open, one engineering decision at a time. Each milestone produces a write-up tied to a specific lesson — the bugs, the trade-offs, and the things that turned out to be harder than they looked.

Follow along: **[@Greegman](https://twitter.com/Greegman)**

---

## Documentation

- **[docs/PRD.md](docs/PRD.md)** — the full Product Requirements Document. Every decision carries its reasoning.
- **[SECURITY.md](SECURITY.md)** — what this project is, what it isn't, and what would change if it were real.

---

## License

MIT — see [LICENSE](LICENSE).

---

*Keep your code fluid and your dreams even bigger. Stay Liquid 💧*
