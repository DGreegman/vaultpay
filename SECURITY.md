# Security & Responsible Use

## What VaultPay is

VaultPay is a **fictional, non-production** wallet infrastructure platform built as a learning and portfolio project. It is engineered to the standard a real fintech would hold itself to, but it is not a real fintech.

**It handles no real money. It stores no real customer data. It connects to no live bank, payment processor, or settlement rail.**

Every external integration — payment gateway, bank API, virtual accounts, OTP, email — is **mocked**. The mocks are built to behave like the real thing, *including realistic failure modes*, so that the engineering problems are genuine. But there is nothing on the other side of them.

---

## What VaultPay is not

This project is **not licensed** to operate as a financial service and makes no attempt to be.

Operating a real wallet system in Nigeria requires, at minimum:

- A **CBN licence** appropriate to the function — typically a Payment Solution Service Provider (PSSP), Switching & Processing, or Mobile Money / Payment Service Bank authorisation, with capital requirements reaching ₦100 million and above. The licence is determined by what the product *functionally does*, not what it is called.
- **NDPC registration** as a data controller under the Nigeria Data Protection Act (NDPA) 2023, read together with the General Application and Implementation Directive (GAID) 2025.
- An **AML/CFT program**, real-time transaction monitoring, and regulatory reporting.
- Real **KYC/BVN verification** through an approved provider.

VaultPay implements none of these. Where the code references KYC tiers or fraud rules, those are **simulations of the engineering problem**, not a compliance program.

If this project were ever taken toward commercial use, that would trigger a separate legal and compliance review entirely outside the scope of this repository.

---

## Credentials in this repository

**No real secrets are committed to this repository.** All credentials present in tracked files are throwaway values for local development only.

- `docker-compose.yml` uses `vaultpay/vaultpay` as the Postgres username and password. These are **obviously fake by design**, bind only to `localhost`, and exist solely so the local stack comes up with one command.
- `.env.example` contains **placeholder values only**. It documents the shape of the configuration, not its contents.
- `.env` is **gitignored** and is never committed.

### If you clone this

Generate your own values for anything that would matter. In particular:

- **JWT signing secrets** and **webhook HMAC keys** — if any demo value ships in this repo, treat it as public knowledge and replace it. A predictable signing key is not a key.
- **Database credentials** — the local defaults are fine for a container on your laptop and are wrong everywhere else.
- **`sslmode=disable`** in the local `DATABASE_URL` is correct for a local container and would be **wrong in production**.

### A note on git history

`.gitignore` only prevents *untracked* files from being added. It has no effect on files git already tracks. If a secret is ever committed — even once, even if immediately deleted in the next commit — **it remains in the repository history and must be considered compromised.** Removing the file does not remove the secret. The only real remediation is to **rotate the credential**.

The discipline is checking `git status` *before* committing, not after.

---

## On publishing fraud rules

This repository will eventually contain a fraud and risk rules engine, including velocity thresholds, transaction limits, and PIN lockout counts — all visible in the source.

**This is safe here precisely because nothing is real.** There is no money to steal and no system to game.

In a production fintech, **fraud detection thresholds are among the few things you genuinely do not open-source.** Publishing your exact velocity limits tells an attacker precisely how to stay under them. Real systems keep rule *parameters* in configuration or a database, tune them continuously, and treat them as sensitive operational data — even when the rules *engine* itself is well-understood.

The distinction matters: the *architecture* of a fraud engine is a legitimate thing to teach and discuss. The *specific numbers* a live system uses are not.

---

## Reporting an issue

VaultPay is not a production system, so there is no vulnerability disclosure process in the usual sense.

That said — if you find a genuine security flaw in the *design or implementation*, that is exactly the kind of feedback this project exists to receive. Open an issue, or reach out directly. A bug in the ledger's concurrency handling is not an embarrassment to be hidden; it is the whole point of building this in public.

**Please do not** open issues that are simply reports of the intentionally-fake credentials described above.

---

## Data protection

No real personal data is collected, processed, or stored. All user records, KYC submissions, and transaction data in any running instance are synthetic.

Where the design references NDPA obligations — a 72-hour breach notification window, data minimisation, Data Protection Impact Assessments for automated decision-making — these are treated as **design constraints to reason about and document**, not as live compliance obligations.

The reasoning is written up in the [PRD](docs/PRD.md), §11.

---

*Last reviewed: July 2026*
