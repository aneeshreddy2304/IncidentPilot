# Security model

IncidentPilot is a private operator console. It has no application authentication and is not intended to be exposed directly to the internet. Use localhost bindings or a private Kubernetes port-forward. Adding an ingress or public listener changes the trust boundary and requires separate authentication and network controls.

## Readonly and assisted modes

`readonly` is the default deployment mode. Server handlers reject cluster mutation/exec operations in that mode; the default cluster RBAC also omits write privileges. The investigator uses read tools and stores insights/proposals rather than applying cluster changes itself.

In `assisted` mode, an operator can review a proposal and ask the server to apply it. The server fetches the stored proposal and checks its status and resource identity. Assisted mode also enables direct operator YAML editing and pod exec. Proposal approval is the AI-remediation boundary, not a second approval requirement for all manual actions.

## Credentials differ by deployment mode

| Deployment | Boundary to verify |
|---|---|
| Helm / raw cluster manifests | Separate server and investigator ServiceAccounts; mode-appropriate RBAC. The investigator should retain read-only access without Secrets access. |
| Docker Compose | Both containers mount a host kubeconfig. Mounting the file read-only does not reduce the permissions granted by its credentials. Supply restricted identities; do not assume the cluster-mode separation applies automatically. |
| Local binaries | Effective rights follow the selected kubeconfig/in-cluster identity. Verify that identity before use. |

`readonly` application checks are defense in depth, not a replacement for least-privilege Kubernetes credentials. Limit scope to the namespaces/resources actually needed for the environment.

## Model inputs and external providers

Investigation can read cluster state, events and pod logs. Those inputs may themselves contain application secrets or personal data. When a cloud provider is selected, relevant context is sent to that provider; the local Ollama option changes that data path but does not establish a comprehensive privacy guarantee.

Review data-handling requirements before connecting university, customer, or production clusters. Treat model output and cluster/log content as untrusted. Operator review remains necessary even for apparently small YAML changes.

## Stored provider settings

The agent encrypts persisted settings with AES-256-GCM. A generated `.settings.key` file lives beside the SQLite database with owner-only file permissions. Encryption protects the database contents in isolation; an actor who can read both the database and key can decrypt them. Protect the volume, host access and backups together. This is not external KMS-backed key separation.

Settings persistence requires the configured database/key path to remain writable and durable. Runtime settings can override first-boot environment defaults; see [configuration](configuration.md).

## Outbound requests

The `safehttp` client blocks link-local/cloud-metadata addresses after DNS resolution. It intentionally permits private cluster addresses for integrations such as Ollama and Prometheus; it is not a complete private-network egress restriction. Apply environment-level outbound policy where appropriate.

## Before connecting a cluster

- Keep the console private; do not publish its UI/agent ports without an authenticated access layer.
- Verify mode, RBAC and the actual kubeconfig permissions in the chosen deployment.
- Use a disposable cluster for demo failure/remediation commands.
- Keep API keys, kubeconfigs, logs and database backups out of source control and screenshots.
- Review the exact manifest diff and resource identity before applying a proposal.

No security certification or penetration-test result is claimed. See [architecture](architecture.md) and [deployment](deployment.md) for implementation details.
