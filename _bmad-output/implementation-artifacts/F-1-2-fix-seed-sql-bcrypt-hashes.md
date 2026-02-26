# Story F-1.2: [BACK] Fix seed SQL to use Go-compatible bcrypt hashes

Status: ready-for-dev

## Story

As a developer,
I want seed data users to have passwords that work with the backend auth system,
so that I can log in with documented credentials after a fresh database setup.

## Acceptance Criteria (BDD)

**AC1: Seed users can log in**
- **Given** a fresh database seeded with `backend/testdata/seed.sql`
- **When** I try to log in with `admin@hopeitworks.dev` / `admin1234`
- **Then** login succeeds with 200 and JWT cookie is set

**AC2: All seed passwords are Go bcrypt compatible**
- **Given** seed SQL has been applied
- **When** the Go backend calls `bcrypt.CompareHashAndPassword(hash, password)`
- **Then** it returns nil (success) for all seeded seed users (admin, sarah, marc, dev, alice, bob)

**AC3: Documented passwords are correct**
- **Given** the seed.sql file
- **When** a developer reads the credentials comment block at the top of the file
- **Then** documented passwords match actual hash values and meet the minimum 8-character requirement

**AC4: No pgcrypto dependency for password hashing**
- **Given** the seed.sql file
- **When** executing the seed against a fresh database
- **Then** no `crypt()` or `gen_salt()` calls are used for password fields — pre-computed `$2a$` bcrypt hashes are used as string literals instead

## Tasks / Subtasks

- [ ] Read `backend/testdata/seed.sql` to confirm current state: 6 users (admin, sarah, marc, dev, alice, bob) all using `crypt(password, gen_salt('bf', 10))` with sub-8-char passwords (admin123, sarah123, etc.)
- [ ] Decide on standard seed passwords — use `admin1234` for admin-role users, `user1234` for user-role users (all meet the 8-char minimum enforced by the backend)
- [ ] Generate Go-compatible bcrypt hashes for each seed user using a Go one-liner, for example:
  ```bash
  go run -e 'package main; import ("fmt"; "golang.org/x/crypto/bcrypt"); func main() { h, _ := bcrypt.GenerateFromPassword([]byte("admin1234"), bcrypt.DefaultCost); fmt.Println(string(h)) }'
  ```
  Or write a small `backend/testdata/genhash/main.go` script, run it, then delete it.
- [ ] Replace all 6 `crypt(..., gen_salt('bf', 10))` expressions in the `INSERT INTO users` block with the corresponding pre-computed `$2a$...` string literals
- [ ] Update the credentials comment block at the top of the file to reflect the new passwords:
  - `admin@hopeitworks.dev  / admin1234  (admin)`
  - `sarah@hopeitworks.dev  / admin1234  (admin)`
  - `marc@hopeitworks.dev   / admin1234  (admin)`
  - `dev@hopeitworks.dev    / user1234   (user)`
  - `alice@hopeitworks.dev  / user1234   (user)`
  - `bob@hopeitworks.dev    / user1234   (user)`
- [ ] Check whether `pgcrypto` is used anywhere else in the seed file or in migrations — if it is not used for anything other than password hashing, remove the `CREATE EXTENSION IF NOT EXISTS pgcrypto` line if present; otherwise leave it
- [ ] Manually verify one hash by running a quick Go snippet calling `bcrypt.CompareHashAndPassword` against the inserted hash and the plain-text password

## Dev Notes

- Priority: P0 — developers cannot use seed data for local development; only users registered via the API can currently authenticate
- Root cause: `pgcrypto`'s `crypt()` uses the OpenBSD blowfish variant (`$2a$`) but the internal byte layout differs from Go's `golang.org/x/crypto/bcrypt`, making the hashes mutually incompatible
- Go bcrypt produces `$2a$10$...` hashes — embed these directly as SQL string literals; no runtime function call needed
- The seed file is idempotent (`ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash`) so re-running the seed after this fix will overwrite stale hashes
- All current seed passwords (admin123, sarah123, etc.) are only 8 characters if you count carefully — they are actually 8 chars but the backend minimum is 8, so they technically pass; however the documented password `admin123` is 8 chars and the context note says the known-working API-registered password is `admin1234` (9 chars) — use `admin1234` / `user1234` to match documented working credentials and avoid any edge-case ambiguity
- File to modify: `backend/testdata/seed.sql` (lines 9–14 for the comment block, lines 26–31 for the INSERT values)
- No migrations, no Go source changes, no OpenAPI changes required — this is a pure SQL data fix
