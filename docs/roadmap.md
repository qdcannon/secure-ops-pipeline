# Secure Ops Pipeline — Roadmap

This roadmap sequences the work so each phase produces something
independently working and demoable, rather than one large all-at-once build.

---

## Phase 0 — Repo & Project Setup
- [ ] Initialize repo structure (`services/`, `deploy/`, `docs/`, `scripts/`)
- [ ] Write `architecture.md`, `roles-and-rbac.md`, `threat-model.md`, `adapters.md`
- [ ] Set up `docker-compose.yml` skeleton (empty services, no logic yet)
- [ ] `scripts/gen-keys.sh` — generate test PKI keypairs for devices + roles

## Phase 1 — Device Adapter Interface (current focus)

**Language decision:** the Device Gateway and adapter framework are built in
**Go**, not Python. The gateway's core job — handling many independent,
concurrent device connections — is Go's strongest fit (goroutine-per-device,
channel-based fan-in), and it closes a recurring Go/CNCF gap on the resume
with a genuine architectural reason, not a bolted-on justification. The
Processor and API layers remain Python.

- [x] Define the `Adapter` interface (Go): `DeviceType()`, `DeviceID()`,
      `Connect()`, `Listen()` (returns event + error channels), `Sign()`, `Close()`
- [x] Define the shared `Event` struct: device ID/type, timestamp, payload,
      checksum, PKI signature, metadata
- [x] Gateway orchestration (`gateway.go`): one goroutine per adapter,
      fan-in to a shared channel, publish to ZeroMQ; per-device errors log
      and continue rather than killing the stream
- [ ] Implement `camera` adapter (MVP)
- [x] Implement `motion_sensor` adapter (MVP)
- [ ] Stub `rfid_access`, `alarm`, `radio`, `presence` adapters (interface only, no logic)
- [x] Unit tests for adapter contract compliance (mock adapter satisfying `Adapter`)
- [x] Adapter registry pattern (`Register`/`Build`) + `devices.json` config-driven
      construction — added beyond original scope, so new devices/instances
      don't require code changes to `main.go`

## Phase 2 — Core Pipeline (Local, No Orchestration Yet)
- [ ] Device Gateway: receive adapter events, verify signatures, forward via ZeroMQ
- [ ] Processor — camera: perceptual hashing, duplicate/loop detection
- [ ] Processor — motion sensor: heartbeat/gap + anomaly detection
- [ ] Storage wiring: MinIO for media, Postgres for metadata/results
- [ ] End-to-end test: device → gateway → processor → storage, running via `docker-compose`

## Phase 3 — DevSecOps Wrapper
- [ ] `.gitlab-ci.yml`: build + test each service in containers
- [ ] Container image publishing (local/self-hosted registry or Docker Hub)
- [ ] Containerized integration test stage in CI

## Phase 4 — Kubernetes + GitOps
- [ ] Stand up single-machine k3s cluster (k3d or minikube)
- [ ] Write k8s manifests per service
- [ ] Install ArgoCD, connect to repo, verify GitOps sync loop
- [ ] Migrate from `docker-compose` dev flow to k3s for integration testing

## Phase 4.5 — AWS Cloud Validation (closes resume gap)

**Match verdict: Strong** — high leverage relative to effort, since AWS/cloud-native
depth is a recurring gap across almost every target posting. Scoped to
**1-2 focused weekends**, not an ongoing build — spin up, do the work,
document it, tear it down (free tier or a few dollars, not a running cost).

Core services to touch (this is the checklist to be able to speak to in an interview):
- [ ] **VPC** — one custom VPC with public/private subnets (even a simple
      2-subnet setup teaches real concepts: routing, security groups, NACLs)
- [ ] **EC2** — one instance running k3s (single-node cluster is fine — this
      doesn't need to be EKS to count as "real")
- [ ] **S3** — store chain-of-custody evidence artifacts/attestation logs
      here instead of local disk (also strengthens the actual PSIM
      narrative — evidence integrity + cloud storage is a natural pairing)
- [ ] **IAM** — scoped roles/policies for the EC2 instance and any service
      accounts (often what separates "used AWS" from "understands AWS
      security," which matters given a security-forward resume framing)
- [ ] **Security Groups** — lock down access to the k3s API and any exposed
      services (ties directly into DevSecOps identity)

Optional, only if time allows — don't add scope just to check a box:
- [ ] **RDS** (small Postgres instance) — only if chain-of-custody metadata
      needs a real database; skip otherwise
- [ ] **CloudWatch** — basic logging/monitoring; cheap to add, reinforces
      "operational excellence" language common in target postings

**What NOT to do:**
- Don't migrate to EKS — significant complexity/cost jump for marginal
  resume value at this stage
- Don't build CI/CD to deploy to AWS yet — that's a phase-2 nice-to-have,
  not required to close this specific gap
- Don't leave it running — tear down after documenting

**Resume payoff:** a single new bullet — *"Deployed PSIM evidence-chain
platform components to AWS (EC2, S3, VPC, IAM) to validate cloud-native
chain-of-custody workflows"* — short, true, and specific enough to survive
a follow-up question in an interview.

## Phase 5 — Observability
- [ ] Prometheus scraping: ingestion throughput, processing latency,
      integrity-check failure rate, device uptime/heartbeat
- [ ] Grafana dashboards on top of the above
- [ ] Alerting rule(s) for a device going silent (ties back to threat model)

## Phase 6 — API + RBAC Layer
- [ ] FastAPI service scaffold
- [ ] Signature/identity verification middleware
- [ ] `admin` router — full oversight, device onboarding, key management
- [ ] `operator` router — live feeds/alerts, ack/dismiss
- [ ] `auditor` router — read-only, verified incident history
- [ ] RBAC integration tests (verify each role *cannot* access out-of-scope data)

## Phase 7 — Documentation & Portfolio Polish
- [ ] Finalize `README.md` (problem, architecture diagram, setup instructions, CI/CD explanation)
- [ ] Record short demo (loop-detection catch, sensor-gap alert, RBAC denial)
- [ ] Finalize `threat-model.md` with concrete before/after scenarios

## Future / Stretch (post-MVP)
- [ ] Implement `rfid_access` adapter for real (ties RBAC to physical access)
- [ ] Implement `alarm` adapter
- [ ] Implement `radio` adapter (ZeroMQ pub/sub broadcast channel)
- [ ] `guest` role — scoped, limited-zone access
- [ ] Live "connected devices/users" presence dashboard
- [ ] Vault/HSM-backed key management (replacing test keypairs)

---

**Current status:** Phase 0 mostly complete (design discussions done — this
roadmap and `architecture.md` are the deliverables). Next up: **Phase 1,
device adapter interface.**
