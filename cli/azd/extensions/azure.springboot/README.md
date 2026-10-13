# Azure Spring Boot extension prototype

This branch is a prototype and knowledge archive, not a release candidate.
Real implementation will start from a clean branch; use the linked handoff
for findings, evidence, and unresolved decisions.

This prototype targets:

```text
existing Spring Boot repository -> azd spring init -> azd up -> Azure Container Apps
```

Start with an application-only repository, then run:

```bash
azd spring init
# Review and accept the proposed services, resources, and files.
azd up
# Standard azd provisioning and deployment.
```

`spring init` writes `azure.yaml`, Bicep under `infra/`, Dockerfiles, and
`.dockerignore`. It does not provision resources, deploy, or create an azd
environment. Select the environment and supply required inputs before
provisioning. Generated projects do not require the Spring extension to deploy.

For automation, use `azd spring init --yes --no-prompt`. `--yes` accepts the
generation plan; `--no-prompt` without it fails. Existing files are never
overwritten, and failed generation removes files created by that attempt.
Re-running init is not a merge or synchronization operation.

The extension no longer registers an automatic init provider or intercepts
bare `azd up`. Future standard `azd init` integration can reuse the analyzer
and generator.

This is an unreleased branch prototype. The explicit `spring init` entry point
is a milestone-specific exception to the
[extension guide's verb-first recommendation](../../docs/extensions/extensions-style-guide.md).
See the [implementation handoff](../../../../SPRING-BOOT-AZD-EXTENSION-IMPLEMENTATION-INVESTIGATION.md)
for the current direction and remaining work.

## Analysis and generation

Analysis and generation are separate:

1. Spring inspection identifies deployable services, ports, health capability,
   and configured service dependencies.
2. azd's existing Java detector resolves the effective Maven model.
3. Runtime dependencies become evidence-backed resource requirements.
4. Generation consumes only the analysis result.

`azure.yaml` uses main's public `project.ProjectConfig` types and `project.Save`
writer, staged before the extension's exclusive file writes. These are Go
library APIs, not host gRPC generation APIs; an independently versioned extension
must pin its azd module dependency. Other extensions invoke core workflows
(for example `azure.ai.models`) or embed their own infrastructure templates
(`azure.ai.agents` and `azure.ai.projects`).

Bicep remains extension-owned. Main's `infra generate` supports resource-based
scaffolding, but is not a drop-in replacement for these Spring datasource/model
bindings, ACA secret references, internal ingress, or Actuator probes. Its
PostgreSQL template also permits all IP addresses. Switching generators would
require explicitly resolving these differences, not silently dropping them.

Container Apps, Container Registry, Log Analytics, and managed identity are the
current hosting policy. Other resources are conditional:

| Evidence | Generated resource and binding |
| --- | --- |
| PostgreSQL runtime dependency | PostgreSQL Flexible Server, database, Key Vault credential, datasource environment variables, and `db.postgres` metadata |
| Spring AI OpenAI starter, Azure Identity, and `AZURE_INFERENCE_*` configuration | Microsoft Foundry resource, model deployment, managed identity access, inference environment variables, and `ai.openai.model` metadata |
| `azure.openai.endpoint` and `azure.openai.api-key` properties referencing `AZURE_OPENAI_ENDPOINT` and `AZURE_OPENAI_API_KEY` | External Azure OpenAI endpoint and secure Container App API-key secret; no inferred Azure OpenAI resource or model |
| A service URL property referencing another detected service | Deployment ordering, internal ingress for the dependency, and the configured service URL environment variable |

The preview lists resources the prototype will provision. A PostgreSQL
dependency is not proof of ownership: this prototype retains its earlier
provisioning policy, not a general provision-versus-reuse resolver.

If PostgreSQL is not present in the analysis, none of its Bicep, parameters,
secrets, environment variables, or `azure.yaml` entries are generated.

Direct Azure OpenAI clients can declare those placeholders in
`application.properties`, `application.yml`, or `application.yaml` (including
nested YAML keys). Set `AZURE_OPENAI_ENDPOINT` and `AZURE_OPENAI_API_KEY` in the
azd environment before provisioning; if the app uses an
`AZURE_OPENAI_API_VERSION` placeholder without a default, set that value too.
If Spring defines a default (for example `${AZURE_OPENAI_API_VERSION:preview}`),
the generated parameter uses it unless the azd environment supplies an override.
Generated service
parameters use azd environment substitution; the key is a required secure
Bicep parameter and a Container App secret reference, never a generated value.
No default is generated: azd must resolve the required environment substitutions
before deployment, and Bicep rejects empty values.
Actuator is optional; health probes are generated only when its dependency is
declared.

## Validation status (October 13, 2026)

Seven application-only cases passed explicit init with stock azd 1.34.2,
Bicep compilation, and Maven packaging (`package -DskipTests`). The run checked
application-source preservation, required/absent resources, repeat-init
overwrite refusal, secret non-disclosure, and one deployable JAR per service.
It did not build Docker images, deploy Azure resources, or exercise runtime
connections. The upstream YAML serializer change left all 56 generated
infrastructure/container files byte-identical.

| Case | Application purpose | Generation/binding coverage |
| --- | --- | --- |
| `spring-todo` | REST Todo CRUD backed by PostgreSQL | One app, database, datasource variables, and probes |
| `spring-ai-todo-agent` | Foundry chat with in-memory Todo tools | One app, model deployment, and managed identity; no database |
| `spring-orders-microservices` | Orders API calls a product catalog and stores orders | Two apps, PostgreSQL, internal/public ingress, URL binding, and library exclusion |
| `spring-external-openai` | Minimal external OpenAI configuration fixture | Supplied endpoint/key/version; no AI resource inferred |
| PostgreSQL-before case | Minimal Spring app without a database | No database infrastructure |
| PostgreSQL-after case | Same app with JDBC/passwordless settings | Database detection; passwordless runtime gap remains |
| `roryp/sora` | Third-party video-generation app | Existing OpenAI bindings and Spring API-version default |

The two-app fixture has three Maven modules: public `orders-service` on port
8080, internal `catalog-service` on port 8081, and the non-deployable
`order-contracts` library. Order creation looks up the catalog item's price,
calculates a total, and writes PostgreSQL. Catalog data is in memory.

`TestSpringInitExistingCases` exercises explicit init on all four checked-in
fixtures, checking generated services/resources, unchanged application files,
and repeat-init overwrite protection. Other Go regressions cover command
consent/flag validation, runtime dependency scopes, optional resources, YAML
round trips/quoted paths, external bindings, and confined writes. Extension
Go tests, vet, and build passed.

Earlier live deployments, before the explicit-init pivot, proved Todo CRUD
and persistence across restart, Foundry managed-identity model/tool calls, and
orders-to-catalog calls with database persistence. Sora's UI and health/readiness
checks passed, but video generation was not validated.

## Managed identity and PostgreSQL

Managed identity already supports ACR pulls, Key Vault secret access, and
Foundry calls. Foundry disables local auth and uses an Azure RBAC assignment.
PostgreSQL needs database authorization as well: the current analyzer does not
retain passwordless intent, and its template disables Entra authentication and
uses a generated administrator password stored in Key Vault.

The intended fix is passwordless binding with a least-privilege app database
role, not rejection of managed identity. Bicep configures Entra authentication
and an administrator; a generated, idempotent `postprovision` script can create
the identity-mapped role and SQL grants. It needs explicit administrator access
and network reachability and must work through standard azd without the Spring
extension installed. This script is not implemented. Making the app identity
an Entra administrator is a simpler but elevated-privilege alternative.

[Spring Petclinic](https://github.com/spring-projects/spring-petclinic) is a
proposed additional acceptance app, not a completed test. Its default H2 and
selected PostgreSQL-profile variants should prove that driver presence alone
does not cause database provisioning.

## Current boundaries

- Root or multi-module Maven projects with explicit Spring Boot service modules.
- Maven must be trusted because effective-model resolution can load Maven core
  extensions.
- PostgreSQL and Microsoft Foundry Models are the application resource requirements currently translated;
  direct Azure OpenAI configuration is treated as an external dependency.
- Service dependencies require an explicit `*_SERVICE_URL` property contract.
- Gradle, profile resolution, and existing-resource selection remain future
  work.
- The prototype compiles in the core Go module and imports `internal/appdetect`;
  an independently versioned implementation must resolve that coupling.
- Generated Dockerfiles currently use Java 21, not the detected Java version.
- Existing Dockerfiles and `.dockerignore` files are protected rather than
  adopted; container-build reuse remains future work.
- Passwordless PostgreSQL bindings are not supported: generated PostgreSQL
  disables Entra authentication and uses an administrator password.
