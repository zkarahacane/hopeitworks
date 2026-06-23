# Local microsandbox (microVM) substrate on a Linux VM

The hardened agent substrate, **microsandbox** (libkrun microVMs), only runs on
**Linux with KVM** (or macOS Apple Silicon natively). It is *embedded* — no server,
no daemon — so it boots the microVM in the **same process** that calls it, on a host
that has `/dev/kvm`. Two consequences:

- The hopeitworks **API must run where `/dev/kvm` is** (you can't point it at a remote
  microsandbox — there is no server mode).
- **Docker Desktop on macOS does not expose `/dev/kvm` to containers** (no nested-virt
  passthrough). So on macOS you run the whole stack inside a **Linux VM with nested
  virtualization**, where `/dev/kvm` is available.

Requirements for the VM path on macOS: **Apple M3+ and macOS 15+** (nested
virtualization), via [Lima](https://lima-vm.io)'s `vz` backend. Validated on M4 Pro /
macOS 26.5.

> The default substrate is plain Docker (`SUBSTRATE=docker`), which runs anywhere
> Docker runs. Everything below is only needed for the microsandbox substrate.

## 1. Bring up the VM (reproducible)

```bash
brew install lima                 # if needed
limactl start ./deploy/lima/microsandbox-vm.yaml --name microsandbox --tty=false
```

This provisions a Linux guest with nested-virt `/dev/kvm`, Docker, and microsandbox.
Verify:

```bash
limactl shell microsandbox -- bash -lc 'ls -l /dev/kvm && ~/.local/bin/msb run python -- python3 -c "print(\"microVM OK\")"'
```

## 2. Run the stack inside the VM

The API auto-runs its schema migrations at boot (golang-migrate), so seed **data
only** — do **not** also run the migrations by hand (that double-applies them and
leaves the DB `dirty`).

```bash
limactl shell microsandbox
# --- inside the VM ---
git clone --depth 1 -b develop https://github.com/zkarahacane/hopeitworks.git
cd hopeitworks
docker compose -p hopeitworks-test -f deploy/docker-compose.agent-test.yml up -d --build
# let the API create the schema, then seed data
docker exec -i hopeitworks-test-postgres psql -U hopeitworks -d hopeitworks_test --set ON_ERROR_STOP=1 \
  < backend/testdata/seed.sql
```

Seed login: `admin@hopeitworks.dev` / `admin1234`.

## 3. Networking — nothing to configure

Lima **auto-forwards** the guest's published ports to the Mac's `localhost`. The
agent-test API listens on `8081`, so from the Mac:

```bash
curl http://localhost:8081/healthz      # → 200
```

No `portForwards` config is required for local access. (Add a `portForwards:` block to
the YAML only to change ports or bind to the LAN.)

## Lifecycle

```bash
limactl stop microsandbox      # stop (frees RAM)
limactl start microsandbox     # restart
limactl delete microsandbox    # remove the VM
```

## Notes / known gotchas

- **mailhog** (`mailhog/mailhog:v1.0.1`) is amd64-only and crash-loops on arm64 guests.
  It's dev mail only — ignore it, or swap for an arm64 mail catcher.
- Resource sizing: the VM defaults to 4 vCPU / 6 GiB. Building the Go API image inside
  the VM is CPU/RAM heavy; bump `cpus`/`memory` in the YAML if your Mac has headroom.
- `agent-stack.sh reset` re-applies migrations via psql and conflicts with the API's
  boot-time auto-migration — inside the VM, prefer the data-only seed shown above.
