<p align="center"><img src="docs/assets/banner.svg" alt="IncidentPilot — Investigate the signal. Review the change. Stay in control." width="100%"></p>

<p align="center"><img src="docs/assets/badges/go.svg" alt="go" height="28"> <img src="docs/assets/badges/react.svg" alt="react" height="28"> <img src="docs/assets/badges/typescript.svg" alt="typescript" height="28"> <img src="docs/assets/badges/kubernetes.svg" alt="kubernetes" height="28"> <img src="docs/assets/badges/helm.svg" alt="helm" height="28"> <img src="docs/assets/badges/prometheus.svg" alt="prometheus" height="28"></p>

<p align="center"><strong>A local-first Kubernetes operations console with an AI investigator and human-reviewed remediation proposals.</strong></p>

## Overview

A local-first Kubernetes operations console with an AI investigator and human-reviewed remediation proposals.

### Why it matters

IncidentPilot brings events, logs, workload state, metrics, and model-assisted reasoning into one operator workflow without hiding the approval boundary.

## Key features

- Cluster resource, event, log, and terminal views
- Structured investigation tools for local or cloud LLMs
- Readonly and assisted operating modes
- Reviewable YAML remediation proposals

## Architecture

```mermaid
flowchart LR
  A[React console] --> B[Go server]
  B --> C[Kubernetes API]
  B --> D[AI investigator]
  D --> C
  D <--> E[LLM provider]
  D --> F[Proposal]
  F --> G[Human review]
  G -->|Approved| B
```

## Technology stack

**go** · **react** · **typescript** · **kubernetes** · **helm** · **prometheus**

## Getting started

Build the web app with Node.js 22+, then compile the Go server and investigator. Connect only to a disposable or explicitly authorized cluster, begin in readonly mode, and keep the console private.
