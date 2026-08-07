# Secure Ops Pipeline — Design Document

## 1. Overview

Secure Ops Pipeline is a distributed, cryptographically-attested ingest and
processing platform for physical security systems — starting with cameras
and motion sensors, and designed to extend to RFID/door access, alarms,
radios, and live presence tracking without redesigning the core system.

It is modeled on real commercial Physical Security Information Management
(PSIM) platforms (e.g. Genetec Security Center, Verkada), combining:

- **Device attestation** — every device has its own PKI identity; data is
  signed at the source, so the system can prove *which device* produced
  *which data*, and that the data has not been altered since capture.
- **Tamper/anomaly detection** — camera feeds are checked for duplicate or
  looped footage; sensor telemetry is checked for gaps, drops, or
  out-of-bounds readings.
- **Role-based access control (RBAC)** — admin, operator, auditor, and
  (future) guest roles each see a different, deliberately scoped slice of
  the system.
- **DevSecOps-first delivery** — every service is containerized, tested in
  CI, and deployed via GitOps, with observability built in from the start
  rather than bolted on later.

## 2. Problem Statement

Security systems combine many independent, often untrusted data sources
(cameras, sensors, badges, radios). Two problems recur across real
incidents:

1. **Spoofed or tampered evidence** — e.g. looping old camera footage to
   mask an intrusion, or physically disabling a sensor so it silently stops
   reporting.
2. **Opaque access** — every person interacting with the system (installer,
   operator, auditor) often has the same broad access, making it hard to
   reason about who could have altered what.

This project addresses both: signed ingest makes tampering detectable, and
RBAC makes access explicit and auditable.

## 3. Goals

- Prove out a signed, multi-device ingest pipeline using ZeroMQ + PKI.
- Detect two concrete attack patterns: looped/duplicate camera footage, and
  sensor tampering via reporting gaps/anomalies.
- Enforce RBAC across four roles with genuinely different data access, not
  cosmetic permission levels.
- Deploy the full system via GitOps (GitLab CI → ArgoCD → k3s) with
  Prometheus/Grafana observability.
- Design a device-adapter interface so new device types (RFID, alarms,
  radios) can be added without changing core ingest/processing logic.

## 4. Non-Goals (for now)

- Real multi-machine/networked deployment (simulated on one machine via
  containers/k3s for this phase).
- Full support for RFID/door access, alarms, radios, and presence tracking
  — these are stubbed as adapter interfaces only, not implemented, in the
  initial build.
- Production-grade key management (e.g. HSM/Vault) — test keypairs are
  sufficient for this phase.

## 5. Architecture

```
Device (camera / motion sensor)
        │  signed + checksummed payload
        ▼
Device Gateway (adapter framework) — Go
        │  ZeroMQ, PKI-signed messages
        ▼
Processor (camera dedup/tamper · sensor anomaly detection) — Python
        │
        ▼
Storage (MinIO for media, Postgres for metadata/results)
        │
        ▼
FastAPI + RBAC layer
        │
   ┌────┴─────┬─────────────┐
   ▼           ▼              ▼
Admin       Operator       Auditor
(full)    (live/alerts)  (read-only,
                          verified history)
```

Deployment wrapper (applies to every service above):
- **GitLab CI** — build, test each service in containers.
- **ArgoCD** — GitOps sync of k8s manifests to a local k3s cluster.
- **Prometheus + Grafana** — pipeline health: ingestion throughput,
  processing latency, integrity-check failure rate, device uptime.

## 6. Device Adapter Framework

Every device type implements a common adapter interface responsible for:
- Reading/receiving raw device output.
- Attaching device identity (signing key reference).
- Computing an integrity checksum.
- Publishing a normalized event to the gateway over ZeroMQ.

MVP adapters: `camera`, `motion_sensor`.
Stubbed for later: `rfid_access`, `alarm`, `radio`, `presence`.

This means adding a new device type is "write one adapter," not "modify the
ingest, processing, and API layers."

## 7. Roles & RBAC

| Role | Access |
|---|---|
| **Admin** | Full oversight, device onboarding, key management |
| **Operator** | Live feeds/alerts, acknowledge or dismiss events |
| **Auditor** | Read-only, verified/unaltered incident history only |
| **Guest** (future) | Scoped, limited-zone access only |

## 8. Threat Model (summary — see `threat-model.md` for full detail)

| Attack | Defense |
|---|---|
| Looped/replayed camera footage | Duplicate/near-duplicate detection on ingested frames |
| Sensor physically disabled or cut | Anomaly detection on expected heartbeat/reporting interval |
| Unauthorized device injecting data | PKI signature verification at ingest — unsigned/invalid data rejected |
| Operator over-reach | RBAC scoping — operators cannot access auditor's verified history view or admin key management |

## 9. Tech Stack

- **Languages:** Go (Device Gateway — concurrent, per-device ingest),
  Python (Processor, API/RBAC services)
- **Messaging:** ZeroMQ, PKI-based signing
- **API:** FastAPI
- **Storage:** MinIO (S3-compatible), Postgres
- **Orchestration:** Docker → k3s (single-machine) → ArgoCD (GitOps)
- **Cloud:** AWS (VPC, EC2, S3, IAM, Security Groups) — minimal viable
  deployment to validate cloud-native workflows and close a recurring
  resume gap (see `roadmap.md`, Phase 4.5)
- **CI/CD:** GitLab CI
- **Observability:** Prometheus, Grafana

## 10. Phased Scope

See `roadmap.md` for the full phase-by-phase build plan.
