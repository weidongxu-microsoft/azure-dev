# Passwordless PostgreSQL proposal

Related epic: https://github.com/Azure/azure-dev/issues/9960

Two separately live-validated examples supporting a shared-core scaffolding
proposal. These are not supported templates or Spring extension output.
Core and extension code are unchanged.

| Example | Complete configuration | Live result |
| --- | --- | --- |
| Petclinic with PostgreSQL | [petclinic](petclinic): `azure.yaml`, infrastructure, SQL bootstrap, JDBC properties, Dockerfile, dependencies | Form insert/update and independent HTML/SQL readback passed; data persisted after restart with fresh managed-identity database connections. |
| Bicep-only ACA registry staging | [aca-bicep-staging](aca-bicep-staging): all four Bicep modules | One parent deployment created ACA, granted `AcrPull`, and configured registry access. Private-image HTTP and identical redeployment passed. |

Both disposable experiments ran on an authorized subscription in `westus3`.
All test resource groups were deleted and absence verified.
The two configurations have **not** been combined into one end-to-end deployment.
The Petclinic bootstrap still uses CLI registry/ingress updates; the second
example demonstrates their Bicep replacement separately.

## Shared-core gap

Password-based PostgreSQL scaffolding plus an attached application identity does
not establish passwordless SQL access. Desired capabilities:

- Entra-only PostgreSQL, selectable system identity, and non-secret connection bindings.
- Ordered identity creation, registry authorization, and explicit SQL bootstrap.
- A non-admin runtime SQL role with an explicit grant policy, separate from migrations.
- Restricted network access; identity does not bypass database firewalls.

SQL principal mapping/grants are data-plane operations, not Azure RBAC.
The JDBC token plugin remains language/application-specific.
No token-expiry refresh endurance test or delayed-RBAC propagation test was performed.

## Publication boundary

Only deployment inputs and instructions are included. There are no hard-coded
passwords, tokens, registry credentials, subscription/tenant IDs, administrator
identities, test IP addresses, or session artifacts. The bootstrap acquires an
administrator token at runtime and keeps it in memory; the app obtains its own
managed-identity tokens.

Petclinic application source is fetched separately at the documented revision.
Monitoring scripts, logs, `.azure` state, credential stores, and compiled artifacts
are deliberately excluded. Publication removes the Petclinic script's private
audit/evidence-file handling; its provisioning operations are preserved.
