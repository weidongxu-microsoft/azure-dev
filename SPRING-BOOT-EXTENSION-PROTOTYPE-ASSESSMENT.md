# Spring Boot extension prototype assessment

## Current implementation status (October 13, 2026)

The October prototype now uses `azd spring init` to preview and generate
`azure.yaml`, Dockerfiles, and Bicep, followed by standard `azd up`.
Automatic bare-`up` initialization has been removed. Earlier results and epic
requirements below describe the previous experiment, not the current entry
point. See the [extension README](./cli/azd/extensions/azure.springboot/README.md).

The [implementation handoff](./SPRING-BOOT-AZD-EXTENSION-IMPLEMENTATION-INVESTIGATION.md)
is the starting point for real implementation. There are no remaining
prototype core changes. YAML generation now reuses main's public project
types/writer; Bicep remains extension-owned after a core-generation parity
experiment exposed binding, secret-reference, ingress, and probe differences.

The latest seven-case matrix passed init on stock azd, Bicep compilation, and
Maven packaging, with source/secret preservation and overwrite checks. This
is local generation/build coverage, not a new cloud/runtime validation.
Earlier live tests proved password-based PostgreSQL persistence, Foundry
managed-identity tool calling, and orders-to-catalog communication. Sora's
live UI/health checks did not validate video generation.

Passwordless PostgreSQL remains incomplete: analysis does not retain its auth
setting, and generation always enables password auth and disables Entra auth.
The intended fix is managed-identity binding plus non-admin database-role/grant
bootstrap through a generated `postprovision` script. The script is not
implemented. Petclinic's default H2 and PostgreSQL-profile variants are
proposed acceptance cases, not completed tests.

The remaining sections preserve historical epic analysis and experiment
results. In particular, init-provider lifecycle and core-owned writes below
describe the removed experiment, not the current extension's exclusive writer.

## Full epic scope

[The Spring Boot epic](https://github.com/Azure/azure-dev/issues/9960) is
substantially broader than inspecting and deploying an existing Spring
application. That workflow is the central Phase 2 payoff, but the epic also
requires template-based creation, composable additions, multiple deployment
shapes, production readiness, and migration scenarios.

### Major product experiences

The epic contains two primary developer experiences:

```text
Existing Spring app
    -> inspect and translate
    -> generate azd configuration and infrastructure
    -> deploy

Developer intent through azd init or azd add
    -> generate a new Spring service and/or Azure resources
    -> update azd configuration and infrastructure
    -> deploy
```

The epic does not propose generating a Spring application from an existing
Azure deployment. It may bind an existing Azure resource selected by the
developer into an azd project, but it does not reverse-engineer deployed
resources into application code.

The [database addition design discussion](./SPRING-BOOT-AZD-ADD-DATABASE-DESIGN.md)
examines a different proposed experience: `azd add` changing an existing
Spring project's dependencies and configuration without generating
`azure.yaml`. That experience is not covered by the current prototype or the
existing issue boundaries.

Beyond existing-application inspection and deployment, the major components
are:

1. **Create new Spring applications:** provide complete `azd init` templates
   and generate new Spring services through `azd add`.
2. **Compose Azure resources:** add databases, messaging, storage, and Key
   Vault through `azd add`, including infrastructure, identities,
   relationships, outputs, and Spring configuration.
3. **Support additional build and deployment shapes:** Gradle, existing
   container builds, WAR packaging, App Service, and Spring Batch on Container
   Apps Jobs.
4. **Provide broader platform and operational capabilities:** managed Spring
   components, additional Azure integrations, observability, Spring AI,
   validation, preview publishing, and Azure Spring Apps migration.

| Epic track | Required outcome | Issues |
| --- | --- | --- |
| Extension foundation | Create, build, test, package, publish, and maintain the first-party `azure.springboot` extension. | [#9961](https://github.com/Azure/azure-dev/issues/9961), [#9974](https://github.com/Azure/azure-dev/issues/9974) |
| Template-first applications | Provide complete Spring Boot templates that already contain azd configuration and deploy through standard commands. | [#9962](https://github.com/Azure/azure-dev/issues/9962) |
| Existing application inference | Detect and translate Maven or Gradle Spring projects, then support bare `azd up` without initial azd files. | [#9965](https://github.com/Azure/azure-dev/issues/9965), [#9966](https://github.com/Azure/azure-dev/issues/9966), [#9978](https://github.com/Azure/azure-dev/issues/9978), [#9969](https://github.com/Azure/azure-dev/issues/9969) |
| Hosting and packaging | Deploy executable JARs to Container Apps, support existing container builds, and later cover App Service and WAR deployment. | [#9963](https://github.com/Azure/azure-dev/issues/9963), [#9983](https://github.com/Azure/azure-dev/issues/9983), [#9970](https://github.com/Azure/azure-dev/issues/9970), [#9976](https://github.com/Azure/azure-dev/issues/9976) |
| Resource composition | Let users add Azure resources and newly scaffolded Spring services through the standard `azd add` experience. | [#9967](https://github.com/Azure/azure-dev/issues/9967), [#9977](https://github.com/Azure/azure-dev/issues/9977) |
| Spring and Azure integration | Map resources to Spring configuration, support managed Spring components, and add more Azure service integrations. | [#9964](https://github.com/Azure/azure-dev/issues/9964), [#9971](https://github.com/Azure/azure-dev/issues/9971), [#9973](https://github.com/Azure/azure-dev/issues/9973) |
| Production readiness | Provide validation, observability, deterministic repeated operations, diagnostics, cleanup, and end-to-end quality gates. | [#9979](https://github.com/Azure/azure-dev/issues/9979), [#9981](https://github.com/Azure/azure-dev/issues/9981), [#9968](https://github.com/Azure/azure-dev/issues/9968) |
| Additional application scenarios | Support Spring AI, Spring Batch as Container Apps jobs, and migration from Azure Spring Apps. | [#9984](https://github.com/Azure/azure-dev/issues/9984), [#9982](https://github.com/Azure/azure-dev/issues/9982), [#9972](https://github.com/Azure/azure-dev/issues/9972) |
| Future discovery | Identify and prioritize additional Spring Boot scenarios. | [#9985](https://github.com/Azure/azure-dev/issues/9985) |

The current prototype covers only a vertical slice of the epic:

```text
existing Maven application or controlled multi-module reactor
    -> evidence-driven PostgreSQL or Foundry Models inference
    -> generated azure.yaml and Bicep
    -> Azure Container Apps deployment
```

It does **not** prototype template-based creation, `azd add` for resources or
new Spring services, Gradle, existing container builds, App Service, WAR,
managed Spring components, broader Azure integrations, Spring Batch, or Azure
Spring Apps migration.

## Existing-application flow: critical path

This section covers the earlier automatic-init experiment. The other epic tracks
have separate implementation paths and risks that this prototype did not
investigate.

### azd and the extension framework

- Start extensions before project loading so `azd init` and bare `azd up` can
  operate when `azure.yaml` does not exist. Tracked by
  [#9838](https://github.com/Azure/azure-dev/issues/9838),
  [#9840](https://github.com/Azure/azure-dev/issues/9840), and
  [#9978](https://github.com/Azure/azure-dev/issues/9978).
- Select an init provider deterministically, invoke it through a versioned
  extension contract, let the user accept or reject the detected provider, and
  return explainable analysis provenance. Tracked by
  [#9838](https://github.com/Azure/azure-dev/issues/9838) and
  [#9978](https://github.com/Azure/azure-dev/issues/9978), with project
  translation covered by
  [#9839](https://github.com/Azure/azure-dev/issues/9839).
- Keep repository writes in azd core with path validation, overwrite
  protection, rollback, and generated-project validation. Tracked by
  [#9839](https://github.com/Azure/azure-dev/issues/9839) and
  [#9840](https://github.com/Azure/azure-dev/issues/9840).
- Refresh command-scoped project and environment state after materialization so
  the normal provision, package, publish, and deploy pipeline can continue.
  Tracked by [#9840](https://github.com/Azure/azure-dev/issues/9840) and
  [#9978](https://github.com/Azure/azure-dev/issues/9978).
- Translate the analyzed service/resource graph into valid `azure.yaml`,
  infrastructure, deployment parameters, and application bindings. Tracked by
  [#9966](https://github.com/Azure/azure-dev/issues/9966),
  [#9967](https://github.com/Azure/azure-dev/issues/9967),
  [#9964](https://github.com/Azure/azure-dev/issues/9964), and
  [#9771](https://github.com/Azure/azure-dev/issues/9771).
- Validate the complete generated resource graph against the selected
  subscription and location before creating resources. Tracked by
  [#9837](https://github.com/Azure/azure-dev/issues/9837),
  [#9978](https://github.com/Azure/azure-dev/issues/9978), and
  [#9979](https://github.com/Azure/azure-dev/issues/9979).

### Java and Spring

- Discover deployable Spring Boot services rather than treating every Java
  module as an application. Tracked by
  [#9965](https://github.com/Azure/azure-dev/issues/9965).
- Resolve effective Maven and Gradle models, including inheritance, profiles,
  dependency management, scopes, and multi-module relationships. Tracked by
  [#9965](https://github.com/Azure/azure-dev/issues/9965) and
  [#9969](https://github.com/Azure/azure-dev/issues/9969).
- Convert runtime dependency and configuration evidence into normalized
  resource requirements with confidence and provenance. Tracked by
  [#9965](https://github.com/Azure/azure-dev/issues/9965) and
  [#9966](https://github.com/Azure/azure-dev/issues/9966).
- Determine packaging, Java version, executable artifact, ports, and health
  probes without requiring Azure-specific application metadata. Tracked by
  [#9965](https://github.com/Azure/azure-dev/issues/9965) and
  [#9963](https://github.com/Azure/azure-dev/issues/9963).
- Map provisioned resource outputs to standard Spring configuration while
  preserving the application's existing configuration and migration strategy.
  Tracked by [#9964](https://github.com/Azure/azure-dev/issues/9964).

## Existing-application flow: risks

### azd and the extension framework

- **Extension discovery bootstrap:** before `azure.yaml` exists, the repository
  cannot declare `requiredVersions.extensions`. The prototype therefore
  participates in bare `azd up` only when `azure.springboot` is already
  installed. A formal design needs a safe way to discover, recommend, install,
  and start plausible providers without launching every registry extension.
  Partly tracked by [#9838](https://github.com/Azure/azure-dev/issues/9838),
  [#9978](https://github.com/Azure/azure-dev/issues/9978), and
  [#9974](https://github.com/Azure/azure-dev/issues/9974).
- **User intent and provider opt-out:** finding a Spring Boot project does not
  prove that the user wants Spring-owned generation rather than generic
  `azure.yaml` initialization. Interactive init needs a provider choice, and
  `--no-prompt` needs an explicit or deterministic selection and opt-out rule.
  Partly tracked by [#9838](https://github.com/Azure/azure-dev/issues/9838) and
  [#9978](https://github.com/Azure/azure-dev/issues/9978), but the generic-init
  fallback is not fully specified.
- **Generated-project lifecycle:** ownership, preview, merge, regeneration, and
  upgrade behavior are not defined for existing or previously generated files.
  The prototype stops invoking the init provider once `azure.yaml` exists, so
  later changes such as adding a database dependency, changing a port, adding
  a module, or changing the Java version do not update infrastructure or
  bindings. A formal design needs explicit drift detection and a previewable
  sync operation that preserves user-owned overrides.
  Tracked by [#9839](https://github.com/Azure/azure-dev/issues/9839),
  [#9840](https://github.com/Azure/azure-dev/issues/9840),
  [#9966](https://github.com/Azure/azure-dev/issues/9966), and
  [#9978](https://github.com/Azure/azure-dev/issues/9978).
- **Resource-graph validation:** one inferred resource can be unavailable in a
  subscription or location that supports the baseline host. The live test
  detected this only after partial provisioning. Tracked by
  [#9837](https://github.com/Azure/azure-dev/issues/9837),
  [#9978](https://github.com/Azure/azure-dev/issues/9978),
  [#9979](https://github.com/Azure/azure-dev/issues/9979), and
  [#9968](https://github.com/Azure/azure-dev/issues/9968).
- **Relational database binding:** the prototype does not create
  least-privilege managed-identity database principals and grants. A generated
  script invoked by the existing `postprovision` hook can perform that SQL
  data-plane work; a new core lifecycle phase is not required. Partly tracked by
  [#9964](https://github.com/Azure/azure-dev/issues/9964),
  [#9967](https://github.com/Azure/azure-dev/issues/9967), and
  [#9968](https://github.com/Azure/azure-dev/issues/9968); no issue directly
  owns the generic azd data-plane binding capability.
- **Failure recovery and diagnostics:** layered ARM failures can produce
  misleading top-level suggestions and leave billable partial resources.
  Partly tracked by [#9968](https://github.com/Azure/azure-dev/issues/9968) and
  [#9979](https://github.com/Azure/azure-dev/issues/9979).
- **Contract maturity:** the pre-project provider API and bare `azd up`
  middleware integration are prototype-specific and need stable lifecycle,
  compatibility, cancellation, and concurrency semantics. Tracked by
  [#9837](https://github.com/Azure/azure-dev/issues/9837),
  [#9838](https://github.com/Azure/azure-dev/issues/9838),
  [#9839](https://github.com/Azure/azure-dev/issues/9839), and
  [#9840](https://github.com/Azure/azure-dev/issues/9840).
- **Analysis trust:** effective Maven or Gradle evaluation can execute
  repository-controlled build extensions and requires explicit trust or
  isolation. Mentioned by [#9965](https://github.com/Azure/azure-dev/issues/9965),
  but no issue directly owns the trust or isolation design.

### Java and Spring

- **General project analysis:** Maven and Gradle inheritance, profiles,
  multi-module layouts, generated configuration, and custom build logic make
  reliable service and dependency discovery difficult. Tracked by
  [#9965](https://github.com/Azure/azure-dev/issues/9965) and
  [#9969](https://github.com/Azure/azure-dev/issues/9969).
- **Inference accuracy:** a dependency can be optional, unused, test-only,
  externally supplied, or configured for several possible services. False
  positives create costly infrastructure; false negatives create broken apps.
  Tracked by [#9965](https://github.com/Azure/azure-dev/issues/9965),
  [#9966](https://github.com/Azure/azure-dev/issues/9966), and
  [#9979](https://github.com/Azure/azure-dev/issues/9979).
- **Configuration diversity:** custom properties, multiple datasources,
  profile-specific values, configuration servers, and framework version
  differences prevent one universal environment-variable mapping. Tracked by
  [#9964](https://github.com/Azure/azure-dev/issues/9964).
- **Packaging diversity:** executable JARs, WARs, native images, layered JARs,
  custom Maven/Gradle tasks, and unsupported Java versions require different
  build and runtime strategies. Tracked across
  [#9963](https://github.com/Azure/azure-dev/issues/9963),
  [#9969](https://github.com/Azure/azure-dev/issues/9969),
  [#9983](https://github.com/Azure/azure-dev/issues/9983), and
  [#9976](https://github.com/Azure/azure-dev/issues/9976).
- **Database initialization:** automatically enabling schema initialization is
  unsafe for applications using Flyway, Liquibase, Hibernate DDL, or custom
  migration workflows. Partly covered by
  [#9964](https://github.com/Azure/azure-dev/issues/9964) and
  [#9968](https://github.com/Azure/azure-dev/issues/9968), but no issue defines
  the migration-preservation policy.

## Issue tracking coverage

All Spring epic sub-issues are currently open. They mix critical dependencies,
preview-quality work, and later scope expansion.

### Critical for the initial existing-application experience

| Area | Issues | Assessment |
| --- | --- | --- |
| azd and extension framework | #9837, #9838, #9839, #9840 | Critical core contracts for defaults, detection, external-project translation, deterministic materialization, and bare `azd up`. |
| azd and extension framework | #9771, #9967 | Critical once detected backing resources must be composed and their outputs bound without collisions. |
| Java and Spring | #9961 | Critical extension foundation, packaging, release, and ownership boundary. |
| Java and Spring | #9965, #9966 | Critical detection and translation path. These own effective Maven analysis, service identity, relationships, provenance, and deterministic project modeling. |
| Java and Spring | #9963 | Critical Container Apps build and deployment path for the first supported host. |
| Java and Spring | #9964 | Critical application-binding path. It explicitly covers PostgreSQL, Service Bus, Storage, Key Vault, configuration precedence, and local/deployed identity behavior. |
| Java and Spring | #9978 | Critical headline experience: start without `azure.yaml`, present the inferred plan, select a compatible region, and materialize deterministic configuration. |
| Java and Spring | #9979 | Critical preflight validation, including resource-region availability, dependency compatibility, identity permissions, and stopping before resources change. |
| Java and Spring | #9968 | Critical release gate for idempotency, failure paths, diagnostics, cleanup, secrets, and live end-to-end validation. |

### Important, but not a blocker for the narrow Maven vertical slice

| Issues | Assessment |
| --- | --- |
| #9962 | Useful bootstrap templates and stable end-to-end fixtures. Important for development sequencing, but not the final existing-repository experience. |
| #9977 | Important for `azd add` and creating new Spring services; not required to analyze and deploy an existing application. |
| #9981 | Important preview-quality observability beyond the baseline Container Apps logs and health probes. |
| #9974 | Publication and preview-readiness gate rather than an implementation capability. |
| #9969, #9983 | Important compatibility expansion for Gradle and existing container builds, but separable from the first Maven executable-JAR scope. |

### Required later epic scope, not part of the initial vertical-slice path

#9971, #9973, #9984, #9970, #9976, #9982, #9972, and #9985 cover
managed Spring components, additional integrations, Spring AI, other packaging
or hosting targets, migration, and future scenario discovery. They are part of
the epic and should not be treated as optional overall, but they do not block
proving the first reliable Maven, Container Apps, and PostgreSQL vertical
slice.

### Risks that are only partially tracked

- **Least-privilege relational database data-plane binding:** #9964, #9967, and
  #9968 require passwordless managed identity and least privilege, but they do
  not define or own the missing generic azd phase that creates database-local
  principals and grants after ARM provisioning.
- **Trusted build-model evaluation:** #9965 permits Maven execution when
  necessary, but no issue defines the trust UX or isolation boundary for
  repository-controlled Maven or Gradle extensions.
- **Partial-provision rollback:** #9968 requires `azd down` and failure-path
  coverage, while #9979 aims to fail before resource changes. Neither directly
  defines automatic cleanup or resumable recovery after a layered deployment
  creates only part of the resource graph.
- **Credential rotation consistency:** the current prototype can rotate the
  generated database administrator password during provisioning before the
  application update succeeds. The passwordless requirements avoid the final
  design problem, but no issue explicitly owns safe transition behavior.
- **Database migration policy:** the existing issues cover configuration and
  connectivity, but do not define how azd should detect or preserve Flyway,
  Liquibase, Hibernate DDL, or custom schema-management behavior.

## Conclusion

The prototype proves that the existing-application vertical slice is feasible:

```text
untouched Spring Boot repository -> azd up -> generated azd project -> Azure Container Apps
```

The pre-project extension point, safe file materialization, and the deployed
Container Apps architecture are useful foundations for a formal feature.
The current iteration also proves that effective Maven analysis can drive
conditional PostgreSQL generation, while dependency and configuration evidence
can drive Foundry Models generation. It still supports controlled application
shapes and is not yet general Spring Boot support.

## What the prototype proves

The prototype successfully demonstrated that:

- an extension can start before `azure.yaml` exists;
- azd can ask installed extensions to inspect a repository;
- an extension can return deterministic project-relative files;
- azd can validate and safely materialize those files;
- bare `azd up` can continue using the newly generated project;
- a generated Dockerfile can use Azure remote build without local Docker;
- generated Bicep can provision a passwordless Container Apps deployment; and
- evidence for a PostgreSQL runtime dependency generates only the required
  PostgreSQL metadata, infrastructure, and application binding; and
- Spring AI plus Azure Identity and `AZURE_INFERENCE_*` configuration generates
  a Foundry model deployment with managed-identity access; and
- a Maven reactor can classify shared libraries separately from deployable
  services and generate an internal service binding; and
- the PostgreSQL REST, Foundry tool-calling, and multi-service applications
  work in live deployments.

This validates one major product direction within the epic. It does not
validate broad compatibility with existing Spring Boot repositories or the
other epic tracks listed above.

### Live PostgreSQL validation

A clean copy of the acceptance application, with no `azure.yaml`, Dockerfile,
infrastructure, or `.azure` directory, completed bare `azd up` in West US 3.
The generated deployment created:

- Azure Container Registry;
- Log Analytics;
- a Container Apps environment and application;
- a user-assigned managed identity;
- PostgreSQL Flexible Server 16 and the `todos` database; and
- Key Vault with a Container Apps managed-identity secret reference.

Health, liveness, readiness, create, list, and complete requests succeeded. A
Todo remained after restarting the active Container Apps revision, confirming
that the application used PostgreSQL rather than process-local storage.

The live test also exposed important formal-feature risks:

- **Location selection must consider every inferred resource.** The same
  subscription supported Container Apps in East US 2 but was restricted from
  creating PostgreSQL there. Validation occurred only during deployment, after
  other resources had already been created.
- **Deployment diagnostics can hide the actionable cause.** azd first reported
  a possible existing or soft-deleted resource, while the nested PostgreSQL
  error showed an empty supported-version set caused by the regional
  subscription restriction.
- **Failed provisioning leaves partial resources.** The East US 2 attempt left
  the hosting resources and Key Vault for later cleanup.
- **Database policy is hard-coded.** Version 16, `Standard_B1ms`, a 32-GB disk,
  database and administrator names, public networking, and the Azure-services
  firewall rule are generator policy rather than application analysis.
- **Credential rotation is coupled to provisioning.** The administrator
  password is a generated Bicep parameter default. Reprovisioning can rotate
  it and create a new Key Vault secret version before application deployment
  succeeds.
- **Automatic schema initialization is not generally safe.**
  `SPRING_SQL_INIT_MODE=always` works for this idempotent fixture but cannot be
  assumed for applications using Flyway, Liquibase, Hibernate DDL, or custom
  migration processes.

### Live Foundry Models validation

The Spring AI Todo Agent completed `azd up` in East US 2. The generated
deployment created a Foundry `AIServices` account, `gpt-4.1-mini`, Container
Apps hosting, and a user-assigned application identity. The application used
Microsoft Entra authentication, called the model, invoked its create-Todo tool,
and returned the updated Todo list.

The live test changed the prototype in three ways:

- Central US Container Apps provisioning failed from temporary regional
  capacity, so retries must distinguish platform capacity from generator bugs.
- Corporate policy rejected API-key authentication. The final design disables
  local auth and grants the application identity Cognitive Services User.
- A transitive JSON-schema dependency mismatch appeared only when Spring AI
  generated tool definitions. The fixture now tests tool-schema generation.

### Live multi-service validation

A Maven reactor with a shared contracts library, internal catalog service, and
public orders service completed `azd up` in West US 3. The extension generated
two Container Apps, kept catalog internal on port 8081, exposed orders on port
8080, deployed catalog first, and supplied its URL to orders. Creating and
listing an order proved the orders-to-catalog call and PostgreSQL persistence.

Core `--from-code` instead classified all three Maven modules as services,
assigned port 8080 to each, and produced no catalog dependency or URL binding.
This demonstrates that the extension's incremental value is Spring-aware
topology, not the basic existing-application workflow.

## Good foundations for a formal feature

### Pre-project provider lifecycle

The beta `InitService` establishes the missing extension lifecycle:

1. An extension declares the `init-provider` capability.
2. The extension registers a named provider over the gRPC broker.
3. azd asks registered providers to inspect the project directory.
4. No match falls back to existing azd detection.
5. Multiple matches fail instead of selecting an arbitrary provider.

Provider ordering is deterministic, registrations are removed when the stream
closes, and capability checks prevent undeclared access. These behaviors should
remain in a production design.

### Core-owned filesystem writes

The extension returns file paths and content, but azd owns repository writes.
The core implementation:

- rejects absolute paths and parent traversal;
- rejects duplicate paths, including case-insensitive duplicates on Windows;
- refuses to overwrite existing files;
- requires a generated `azure.yaml`;
- validates the generated project;
- sorts files for deterministic materialization; and
- removes created files when writing or validation fails.

Keeping this trust boundary in core is a strong production design. It prevents
every init provider from implementing its own path-safety and rollback logic.

### Deterministic generation

For the supported fixture, the extension generates stable service names,
ordered files, infrastructure, parameters, and health probes. Deterministic,
reviewable output should remain a formal feature requirement.

### Secure Azure architecture

The generated infrastructure uses:

- Azure Container Registry with the admin account disabled;
- a user-assigned managed identity;
- a scoped `AcrPull` role assignment;
- Azure Container Apps;
- Log Analytics; and
- no registry password in generated configuration.

This is a reasonable secure default for a first Container Apps scenario.

### Explicit analysis trust boundary

The Spring-specific pass reads project files without loading the application.
Resource analysis now invokes Maven to obtain the effective model. This does
not start the Spring application, but Maven can load repository-controlled core
extensions.

A formal feature should distinguish safe file inspection from trusted build
model resolution. Maven and Gradle execution should require a clear trust
decision or an isolation boundary.

## Prototype-only areas that need a better design

### The init contract returns final files too early

The contract currently combines detection, project modeling, hosting choice,
and rendering into one response containing file bytes. That is simple for a
prototype but limits future composition and user control.

A formal contract should separate:

1. **Detection**: what applications and modules exist, with evidence and
   confidence.
2. **Translation**: the proposed azd services, dependencies, ports, and hosting
   requirements.
3. **Resolution**: user choices, conflicts, defaults, and explicit overrides.
4. **Rendering**: materialization of `azure.yaml`, infrastructure, and other
   artifacts.

This would let azd show and modify a proposal before files are written. It
would also allow Spring detection to compose with resource providers instead
of owning an entire fixed infrastructure template.

### Bare `azd up` integration is command-specific

The prototype performs pre-project initialization inside
`ExtensionsMiddleware` only for `up`, then replaces lazy and concrete
command-scoped dependencies so later middleware can continue. This proves the
flow but tightly couples extension startup, project creation, middleware order,
and dependency refresh.

A formal feature should have a dedicated pre-project resolution phase shared by
`init`, `up`, and any future command that can start without `azure.yaml`.
Project creation should produce a new command context through a supported
container lifecycle rather than replacing selected registrations in middleware.

### Provider selection needs product semantics

Failing on multiple matches is safer than silently choosing one, but a formal
feature still needs:

- provider priority or specificity rules;
- a user-visible provider selection mechanism;
- non-interactive selection behavior;
- diagnostics explaining why each provider matched;
- version and compatibility negotiation; and
- rules for one extension contributing more than one detected service.

### The generated files are all-or-nothing

The current implementation refuses any existing target file. Formal behavior
must define how inferred projects interact with:

- an existing Dockerfile;
- partial infrastructure;
- an existing but incomplete `azure.yaml`;
- repeated detection after the source project changes; and
- files previously generated by an older extension version.

Generated metadata should record ownership and provenance. Regeneration should
produce a preview or merge plan rather than blindly overwrite user files.

### Core existing-application initialization is the baseline

Core `azd init --from-code` already detects Maven Java services and common
database dependencies, generates a Container Apps resource model, and
synthesizes most of the same Azure architecture as the extension prototype.

The Spring Todo comparison found a concrete Spring-specific gap: core emits
generic `POSTGRES_*` variables while the application consumes
`SPRING_DATASOURCE_*`. Core packaging also encountered a `pack` and Docker API
compatibility blocker before reaching the Maven build.

See
[Spring Boot core azd initialization baseline](SPRING-BOOT-CORE-AZD-INIT-BASELINE.md)
for the experiment, generated model, infrastructure comparison, packaging
result, and product implications.

### Current azd relational database binding gap

The built-in azd resource scaffold currently treats PostgreSQL, MySQL, Redis,
and MongoDB as credential-based resources with an implicit Key Vault
dependency. For PostgreSQL, generated application variables use the server
administrator login and a password stored as `vault.postgres-password`.

This is different from Azure services such as Storage, Service Bus, Event Hubs,
and Key Vault, where access can be granted through ARM-managed Azure RBAC role
assignments. A least-privilege PostgreSQL identity requires database data-plane
operations after the server exists:

1. Configure a Microsoft Entra administrator.
2. Connect to the database as that administrator.
3. Create a PostgreSQL principal for the application's managed identity.
4. Apply idempotent database and schema grants.
5. Configure the application to acquire and refresh PostgreSQL access tokens.

azd can orchestrate these operations through custom hooks, deployment scripts,
Service Connector, or an extension provisioning provider, but its standard
resource scaffolding does not currently expose a generic post-provision
database-binding phase.

The Spring prototype will therefore retain the current azd-style implementation:

- generate the PostgreSQL administrator credential as a secure deployment
  parameter;
- store it in a conditionally generated Key Vault;
- grant the Container App identity access to that secret; and
- inject the datasource password through a Container Apps Key Vault reference.

This is a deliberate prototype compatibility choice, not the desired final
security model. A formal feature should add a data-plane resource-binding
abstraction and use a least-privilege Microsoft Entra database principal.

### Production readiness work is missing

The prototype needs broker-level and command integration tests, telemetry and
privacy review, UX and error-message design, contract versioning documentation,
upgrade compatibility, and full preflight validation. The extension README also
still describes the earlier two-command flow and must be updated if bare
`azd up` becomes supported.

## Does Spring inspection work generally?

**No. It works for narrow, known single-module applications and controlled
multi-module Maven reactors.**

### Currently supported shape

The repository must have:

- a root `pom.xml`;
- either a root application or Maven reactor leaf modules;
- each deployable module must have a Spring Boot parent or direct dependency;
- each deployable module must declare `spring-boot-maven-plugin` and Actuator;
- a project version directly on the project or parent;
- optional `server.port` as a literal integer in
  `src/main/resources/application.properties`; and
- an executable JAR produced under the module's `target` directory.

For this shape, the detector and generator are deterministic and were validated
end to end.

### Important unsupported or unreliable cases

| Area | Current limitation |
| --- | --- |
| Build system | Gradle is unsupported. |
| Maven layout | Controlled reactors work, but nested aggregators, duplicate service names, optional modules, and ambiguous deployable modules are not fully modeled. |
| Effective POM | Resource dependencies use Maven's effective model, but service metadata and eligibility still rely on limited direct POM inspection. |
| Maven expressions | Values such as `${revision}` and inherited artifact metadata are not evaluated. |
| Application type | The detector does not prove that the module is an executable web application. |
| Configuration | `application.yml`, profile files, environment placeholders, command-line overrides, and management-server settings are ignored. |
| Ports | Only a literal `server.port` in `application.properties` is recognized; all other cases become port 8080. |
| Actuator | Actuator must be a direct dependency, and standard liveness/readiness paths are assumed. Exposure and probe configuration are not inspected. |
| Java | The POM Java version is recorded but not used; the Dockerfile always builds and runs Java 21. |
| Maven execution | The generated Dockerfile uses a fixed Maven image and `mvn`, not the repository's Maven wrapper or Maven configuration. |
| Artifact selection | `target/*.jar` can select the wrong file or fail when multiple JARs exist. |
| Source layout | The Dockerfile copies the full reactor and assumes `mvn -pl <module> -am package` is sufficient. |
| Naming | The artifact ID is normalized, but Azure and azd length and collision rules are not fully modeled. |
| Dependencies | PostgreSQL is inferred from the effective Maven model. Service Bus, Event Hubs, Storage, Key Vault usage, and other requirements are not yet inferred. |
| Runtime settings | Context path, JVM options, memory requirements, environment variables, secrets, scaling, ingress policy, and private networking are not inferred. |

## Existing tools that should replace custom heuristics

The repository and Java ecosystem already contain most of the building blocks.
The formal feature should integrate them instead of expanding the prototype's
handwritten POM parser.

### Existing azd Java application detector

`cli/azd/internal/appdetect/java.go` already:

- invokes Maven's `help:effective-pom`;
- resolves inherited and profile-adjusted Maven model values;
- walks multi-module repositories;
- records the root project for child modules; and
- recognizes PostgreSQL, MySQL, Redis, and MongoDB dependencies.

It specifically recognizes `org.postgresql:postgresql` and
`spring-cloud-azure-starter-jdbc-postgresql`. Existing tests cover PostgreSQL
and MySQL detection in a multi-module Maven repository.

This detector is a much better starting point than
`extensions/azure.springboot/internal/springboot/detect.go`. It is still not a
complete Spring detector:

- every leaf Maven module can be returned as a Java service, including
  libraries;
- database inference is a coordinate lookup without confidence or provenance;
- dependency scope is not considered before proposing a database;
- Spring Boot executability, ports, configuration, and artifacts are not
  modeled; and
- Gradle is unsupported.

The model should be promoted or exposed through a generic detection contract so
extensions do not create competing build-system implementations.

### Maven Model Builder and effective POM

Maven's model builder is authoritative for parent inheritance, profiles,
interpolation, dependency management, plugin management, and imported BOMs.
The existing azd implementation obtains this through
`mvn help:effective-pom`.

There is an important trust distinction:

- embedding Maven Model Builder can construct the declarative model under
  controlled resolution rules; but
- starting `mvn` or `mvnw` trusts the repository and can load
  repository-controlled core extensions from `.mvn/extensions.xml`.

Therefore, deep Maven resolution must either be an explicit trusted operation
or run inside an isolation boundary. A safe first pass can parse files without
execution, then request permission or use a sandbox for authoritative
resolution.

### Gradle Tooling API

The Gradle Tooling API is the appropriate source for project hierarchy,
subprojects, tasks, source directories, and resolved dependencies. It is
wrapper-aware and is used by IDEs.

It configures the Gradle build and runs a Gradle daemon, so build scripts and
plugins must be treated as repository-controlled code. It is not a safe
read-only parser for untrusted repositories.

### AppCAT for Java and Konveyor

Azure Migrate Application and Code Assessment for Java 7.x is the strongest
existing candidate for technology and Azure-resource evidence. It:

- analyzes Java source or binaries;
- reports Java version, framework, build tool, dependencies, and technologies;
- supports Azure Container Apps as an assessment target;
- emits YAML or JSON analysis output;
- supports custom YAML rules; and
- is built on the open-source Konveyor analysis engine.

The Azure AppCAT rulesets already contain rules for databases, message queues,
Spring Boot to Key Vault, local file-system use, Azure SDK migration, and other
cloud-readiness concerns. Rules can combine dependency coordinates, source API
usage, and configuration-file patterns. This is substantially stronger than
deciding from one dependency alone.

AppCAT should be evaluated as an evidence provider, not used as the complete
init engine. Its output is assessment-oriented and still needs translation
into a stable model such as:

```text
kind: postgresql
consumer: services/orders
evidence:
  - type: dependency
    value: org.postgresql:postgresql
  - type: configuration
    value: jdbc:postgresql
confidence: high
proposedAzureResource: azure.database.postgresql
```

Adoption also needs decisions about binary distribution, startup cost, pinned
ruleset versions, offline behavior, telemetry, privacy of generated reports,
and compatibility between AppCAT findings and azd's support policy.

### Paketo Java buildpacks

Paketo already detects Maven, Gradle, Spring Boot artifacts, executable JARs,
WARs, Java versions, and build-tool conventions. It supports selecting a module
or artifact in multi-module builds and supplies container-aware JVM memory
configuration.

Paketo is a strong replacement for the generated fixed Java 21 Dockerfile. It
does not infer required PostgreSQL, Service Bus, Storage, or Key Vault
resources. Service bindings are inputs supplied at runtime, not resource
requirements discovered by the buildpack.

### Spring Boot configuration metadata

`META-INF/spring-configuration-metadata.json` describes known configuration
property names, types, defaults, and hints. It is useful for interpreting
properties after an artifact is built, but it does not contain the active
values for a deployment.

Spring Boot configuration has profile overlays, imports, external locations,
environment values, and last-wins precedence. There is no simple static file
parse that exactly reproduces every runtime value. Detection must preserve
unresolved and profile-dependent values rather than silently selecting a
default.

## Recommended detection stack

No single existing tool supplies the complete model. A credible implementation
should combine them:

| Layer | Preferred tool | Result |
| --- | --- | --- |
| Safe discovery | File scan and declarative parsers | Candidate Maven/Gradle roots, config files, wrappers, Dockerfiles |
| Build model | Existing azd Maven detector, Maven Model Builder, Gradle Tooling API | Modules, dependencies, plugins, Java version, build outputs |
| Technology evidence | AppCAT/Konveyor rules | Databases, messaging, storage, secret stores, filesystem and migration concerns |
| Spring configuration | Properties/YAML parser plus metadata | Ports, management settings, profiles, placeholders, unresolved values |
| Packaging | Paketo Java buildpacks | Reproducible executable image and JVM configuration |
| Azure policy | azd translation layer | Proposed resources, identity roles, bindings, hosting, and confidence |

The output between layers must be a normalized evidence model, not generated
Bicep. Resource creation should occur only after azd applies support policy and
resolves ambiguity with the user or explicit non-interactive configuration.

### What general Spring detection should do

A production detector should build a normalized model with provenance rather
than immediately emit files. At minimum it should:

- support Maven and Gradle through separate resolvers;
- discover modules and distinguish applications from libraries and parents;
- resolve effective build metadata without running application code;
- identify executable artifacts and their owning modules;
- detect Java and Spring Boot compatibility;
- resolve application and management ports across supported configuration
  sources;
- inspect Actuator availability and configured probe paths;
- identify dependency-backed Azure integrations;
- report unknown, conflicting, and dynamic values instead of silently assuming;
- allow explicit configuration to override every inference; and
- explain the source of each inferred value.

Detection should return confidence and evidence, for example:

```text
service: orders
module: services/orders
port: 8081
source: services/orders/src/main/resources/application.yml
confidence: exact
```

## Recommended path to formalization

1. Replace the custom POM parser with an adapter over the existing azd Maven
   detector.
2. Define a generic evidence model with source, confidence, module, and scope.
3. Run an AppCAT spike that emits resource evidence for PostgreSQL, Service
   Bus, Storage, Key Vault, and local filesystem use.
4. Use Paketo for packaging instead of generating a fixed Java 21 Dockerfile.
5. Stabilize a generic pre-project detection and translation model in azd core.
6. Keep safe materialization and rollback in core.
7. Replace command-specific `up` handling with a shared pre-project phase.
8. Add Gradle through the Tooling API with an explicit execution trust policy.
9. Separate hosting and infrastructure policy from Spring application
   detection.
10. Add preview, provenance, conflict resolution, and regeneration semantics.
11. Validate against a compatibility suite of representative public Spring
    Boot project shapes before describing the extension as general.

The existing prototype should be retained as an end-to-end reference and
acceptance fixture while these contracts are redesigned. Its PostgreSQL path
now demonstrates the intended boundary: analysis emits a resource requirement,
then generation conditionally emits `azure.yaml`, Bicep, parameters, secrets,
and application bindings for that requirement.

## Adoption research plan

Before investing in the full epic, validate whether Spring developers would
adopt a separate `azd` workflow instead of remaining in Maven, Gradle, their
IDE, or CI.

### Maven and Gradle alternatives

Treat build-tool-native deployment as a primary competitor. Compare:

- Azure Container Apps, App Service, Functions, and Spring Apps Maven plugins;
- the Azure App Service Gradle plugin;
- Jib for Maven and Gradle;
- Spring Boot build-image tasks;
- Eclipse JKube for Kubernetes and OpenShift; and
- comparable cloud-provider Maven and Gradle deployment plugins.

For each tool, record:

- whether it modifies `pom.xml` or Gradle configuration;
- how much of the existing project it detects;
- whether it builds images without a Dockerfile;
- which infrastructure it creates;
- whether it infers or binds databases, messaging, storage, and identity;
- support for local development, CI, preview, cleanup, and repeated deployment;
- generated-file ownership and synchronization behavior; and
- public adoption and pain signals.

The product question is not only whether azd can perform more steps. It is
whether resource composition, identity, configuration mapping, and lifecycle
management justify asking a Java developer to install and invoke a separate
CLI.

### Initial findings

Maven and Gradle tooling provides two simultaneous signals:

1. **Requirement validation:** Java developers already use build-tool-native
   commands to configure, package, and deploy applications.
2. **Substitution risk:** a separate CLI has an installation and workflow
   disadvantage when a build plugin already completes the required job.

Examples include:

- the
  [Azure Container Apps Maven plugin](https://github.com/microsoft/azure-maven-plugins/tree/develop/azure-container-apps-maven-plugin);
- the
  [Azure App Service Maven plugin](https://github.com/microsoft/azure-maven-plugins/tree/develop/azure-webapp-maven-plugin),
  whose `config` goal writes deployment configuration into `pom.xml`;
- the
  [Azure App Service Gradle plugin](https://github.com/microsoft/azure-gradle-plugins/tree/master/azure-webapp-gradle-plugin);
- [Jib](https://github.com/GoogleContainerTools/jib) for daemonless,
  reproducible Maven and Gradle image builds;
- Spring Boot
  [Maven](https://docs.spring.io/spring-boot/maven-plugin/build-image.html) and
  [Gradle](https://docs.spring.io/spring-boot/gradle-plugin/packaging-oci-image.html)
  build-image tasks; and
- [Eclipse JKube](https://eclipse.dev/jkube/) for Java-aware image,
  Kubernetes, and OpenShift generation and deployment.

These tools establish that Dockerfile-free packaging and one-command
application deployment are baseline expectations, not sufficient
differentiation for azd.

The strongest division of responsibility is:

```text
Maven or Gradle
    -> compile, test, package, and build an image

azd
    -> preview and provision the environment
    -> compose resources, identities, and role assignments
    -> map Spring configuration
    -> deploy services and report endpoints
    -> manage environments and teardown
```

#### Where azd is weak

- one Spring service deployed to an existing App Service or Container App;
- application teams that need only a JAR or image upload;
- Kubernetes teams already standardized on JKube, Helm, and GitOps; and
- pipelines where Maven or Gradle plus Terraform already has clear ownership.

#### Where azd may be strong

- applications with databases, messaging, storage, secrets, identities, and
  monitoring;
- multi-service or polyglot applications with shared resources;
- repeatable developer and test environments that need preview and teardown;
- organizations wanting one outer-loop workflow across languages; and
- source-aware migration from Azure Spring Apps to Container Apps.

The current product hypothesis is therefore:

> Maven and Gradle remain the application build authority. azd becomes the
> optional application-and-Azure composition authority.

azd should preserve existing wrappers, Jib configuration, buildpacks,
Dockerfiles, and CI build commands. It should replace a deployment plugin only
when both tools would otherwise own the same hosting configuration or when the
application-level resource graph exceeds the plugin's scope.

### Next evidence collection

Tool existence validates the job but does not prove likely azd adoption. The
first directional census sampled 30 public repositories using:

```text
topic:spring-boot language:Java pushed:>=2026-03-20
archived:false fork:false
```

The sample took the first 10 repositories sorted by most recently updated in
each star band: 20-99, 100-999, and 1,000 or more. It was collected on
2026-09-20 and classified as 15 deployable applications, 10 libraries or
frameworks, and five tutorials.

Within the 15-application subset:

| Signal | Repositories |
|---|---:|
| Maven | 9 |
| Gradle | 6 |
| Multi-module build | 8 |
| Dockerfile | 14 |
| Compose | 13 |
| GitHub Actions | 13 |
| Container-image publication from CI | 7 |
| Kubernetes or Helm configuration | 5 |
| Terraform, Bicep, or `azure.yaml` | 0 |
| Azure Maven plugin or Azure deployment workflow | 0 |
| Cloud or Kubernetes deployment from CI | 0 |
| At least one database or messaging signal | 15 |
| More than one database or messaging signal | 8 |
| Both PostgreSQL and MySQL signals | 5 |

This is directional evidence, not a representative adoption estimate. Topic
tagging, Java-only filtering, star thresholds, and recent-update sorting bias
the sample. Two large repository trees were truncated, file inspection was
limited to 40 relevant files per repository, and dependency matches can come
from tests, samples, optional modules, or support for alternative databases.

The initial interpretation is:

- Maven and Gradle are firmly the build authority.
- Existing Dockerfiles and Compose files are common and must be preserved
  rather than replaced by default.
- Several projects publish images but do not own environment deployment in the
  same repository. This is a plausible azd outer-loop integration point.
- Missing IaC does not prove missing infrastructure. It may be operated
  manually or owned by another repository or platform team.
- Dependency-only inference is unsafe. Competing database signals require
  configuration evidence, module and scope analysis, confidence reporting, and
  user confirmation.
- The sample does not establish Azure demand because it found no explicit Azure
  deployment workflow or Azure build plugin.

### Public Java repositories with `azure.yaml`

An exact GitHub code search on 2026-09-21 for `language: java` in
`azure.yaml` found 151 files across 138 public repositories. Restricting the
analysis to the 113 repositories with `azure.yaml` at the repository root
reduced false positives from generators, test fixtures, and nested examples.

The primary purpose of this census is to establish whether a visible
**Java + azd user base** already exists. The answer is yes: 113 public
root-configured Java repositories, including 70 with Spring evidence, show that
Java applications and azd are already used together. This is a plausible
adjacent audience for a Spring extension that removes the requirement to first
author `azure.yaml`.

The root-project cohort contained:

| Signal | Repositories |
|---|---:|
| Java `azure.yaml` | 113 |
| Spring dependency evidence | 70 |
| Obvious Azure, Microsoft, or Spring sample owners | 14 Spring repositories |
| Other public owners | 56 Spring repositories |
| Active other-owner Spring repositories | 43 |
| Named template metadata | 34 of those 43 |
| `azd-init@...` metadata | 5 of those 43 |
| No template metadata | 4 of those 43 |
| GitHub workflow containing an azd lifecycle command | 11 of those 43 |

Ten of the 11 workflow-enabled repositories retained named template metadata.
The remaining repository used `azd-init` and then added CI. Public workflow
files therefore show that generated azd projects can be operationalized, but
mostly reflect template distribution rather than independently designed azd
adoption.

Across the 70 root Spring configurations:

- Container Apps appeared in 41, App Service in 17, and Azure Spring Apps in
  13;
- 60 declared one Java service and 10 declared multiple Java services;
- 20 selected custom infrastructure and 21 used hooks; and
- only three used inline `resources`, so most Azure topology remained in Bicep
  or template-owned files.

Only 10 Java repositories in the full cohort retained `azd-init@...` metadata;
five had Spring evidence. They were all single-Java-service projects. One
generated a PostgreSQL backing resource, while others contained hosting only or
later added custom Docker, Bicep, hooks, or CI.

This is evidence that Java repositories **can** adopt and retain azd, including
after `--from-code`. It is not evidence of broad external or production
adoption: public code excludes private repositories, GitHub ownership does not
identify customer status, code search is index-dependent, and many repositories
are demos, workshops, experiments, or copies of published templates.

The product implications are:

- there is an existing public Java + azd user base;
- that base demonstrates familiarity with azd and makes extension adoption
  plausible;
- the extension can also attract similar Spring repositories that have not
  adopted azd because they lack `azure.yaml`;
- template creation is the dominant visible Java azd path;
- `--from-code` has a small but concrete public footprint;
- existing projects commonly customize build and infrastructure ownership; and
- Spring-aware import should improve or compose with an existing `azure.yaml`,
  not assume initialization is always starting from an empty repository.

The census does not measure how many existing users would install the
extension or how many non-azd Spring repositories it would convert. That
requires telemetry or an adoption experiment. Template and sample origins
qualify the strength of the demand signal, but do not erase the demonstrated
Java + azd footprint.

The public `azd-init` cases illustrate different post-initialization paths:

| Repository | Observed azd shape | Implication |
|---|---|---|
| `roryp/matrixagents` | Started from `azd-init`, then added Docker, custom Bicep, hooks, frontend build steps, and model infrastructure. | Initialization is a starting point; later Spring analysis must preserve explicit ownership. |
| `puspmish3/MortgageCalculator` | Generated a frontend, Spring backend, PostgreSQL resource, and service dependency. | Core can already generate a useful multi-service graph when its evidence matches. |
| `bbenz/from-commit-to-cloud` | Retained `azd-init` metadata but selected custom infrastructure and added azd CI commands. | Existing custom infrastructure should make extension changes advisory by default. |
| `voddinenideekshith/GitTasker` | Retained a generated hosting model while the application contains several data-service signals. | An existing `azure.yaml` can be incomplete relative to the current application. |
| `allxiao/azd-template-play` | Retained a simple generated Container Apps project. | Some repositories need validation and updates rather than new initialization. |

The four active Spring repositories without template metadata were also weak
evidence of organic production adoption: one was explicitly a reusable shell,
one was a Spring Petclinic AI sample copy, one was an architecture-oriented
FinTech demonstration, and one appeared to be an application but had no public
azd workflow proving deployment. Absence of template metadata is therefore not
enough to classify a repository as independently authored or operational.

This suggests an additional extension experience:

```text
existing azure.yaml
    -> analyze Spring modules and configuration
    -> compare with declared services, resources, ports, and bindings
    -> report missing or conflicting configuration
    -> preserve custom Docker, hooks, and infrastructure ownership
```

Automatic mutation should be limited to clearly core-generated configuration.
Custom Bicep, explicit Docker settings, hooks, and existing resource declarations
should produce a reviewable plan rather than silent replacement.

#### Popular repository compatibility cases

Use a **blind reconstruction test** rather than modifying these repositories'
existing azd configuration. In a disposable copy, hide `azure.yaml`,
`infra/`, and deployment-only Dockerfiles/hooks; retain application and
frontend source and build metadata. Give the implementer only a general
Spring/Azure capability requirement, not the candidate's azd files or this
assessment. Generate a new azd project from the application, then compare it
with the original configuration as the reference answer. Assess functional
parity (services, builds, resources, configuration, endpoint behavior, and
cleanup), not byte-for-byte YAML/Bicep equivalence. Where code cannot establish
resource ownership or credentials, require an explicit input rather than
guessing. Use the azd binary built from the prototype branch, which wires
bare `azd up` to the init provider when `azure.yaml` is absent. The released
azd binary does not include that prototype integration.

Three of the highest-starred non-Microsoft Spring repositories in the census
are initial candidates:

| Repository | Application shape | Prototype result |
|---|---|---|
| `roryp/matrixagents` | One Spring Boot backend, React build hook, WebSocket endpoints, custom Azure OpenAI settings, YAML configuration, and custom Bicep | Outside current scope. Basic YAML is parsed, but the frontend hook and custom AI bindings would be lost. |
| `SandraAhlgrimm/ai-nutrition-planner` | Three Spring Boot modules using different AI frameworks, Buildpack hooks, custom images, scripts, and Bicep | Outside current scope as a complete project. Module topology is discoverable, but build ownership and provider-specific configuration are explicit project choices. |
| `roryp/sora` | Conventional single-module Spring Boot application with Actuator, port 8080, Dockerfile, and Azure OpenAI environment bindings | Bare prototype `azd up` deployed the generated Container App after fixing the root-module Dockerfile path. Home and health endpoints returned 200; video was not tested with a valid key. |

These cases separate application facts from deployment choices.

Reasonably deducible from conventional Maven and Spring configuration:

- deployable Maven modules, artifact-derived service names, and Java versions;
- explicit application ports and Actuator health endpoints;
- conventional executable-JAR packaging;
- recognized database dependencies; and
- environment-variable references in Spring configuration.

Not safe to infer without confirmation or existing project metadata:

- API key versus managed identity authentication;
- Azure OpenAI resource, deployment, model, and API-version selection;
- custom Dockerfile, Buildpack, hook, and frontend packaging ownership;
- custom Bicep resources and role assignments;
- public versus internal ingress when it is only an infrastructure choice; and
- whether existing `azure.yaml` and infrastructure should be merged or
  preserved unchanged.

The original compatibility gaps were:

1. Parse `application.yml` in addition to properties files (basic nested
   mapping now implemented).
2. Inventory generic `${ENV_VAR}` bindings without assuming a resource type
   (only the direct Azure OpenAI contract is now covered).
3. Make Actuator optional for Spring Boot recognition (now implemented for
   web applications).
4. Reconstruct custom build steps where application source supplies evidence;
   do not assume an existing Dockerfile or azd hook will be available.
5. Report uncertainty instead of inventing resource ownership or credentials.
6. Treat Azure OpenAI as an external dependency unless resource ownership and
   authentication are explicit (now implemented for endpoint/API-key clients).

Start with `roryp/sora`: a single-module test of Spring Azure OpenAI
configuration and its required runtime bindings. Keep its repository and
reference configuration hidden from the implementer until the generic feature
is complete. Then test `matrixagents` for YAML/Actuator/frontend coupling and
`ai-nutrition-planner` for multi-service, Buildpack, and framework-specific
configuration. Record inferred facts, missing inputs, deployment results, and
differences from the reference for each case.

**First blind result (`roryp/sora`):** The implementer received only a generic
Spring Boot + external Azure OpenAI requirement, without the candidate
repository or its azd files. The requirement did name representative
`azure.openai.*` property and `AZURE_OPENAI_*` environment conventions, so
this tests implementation against an unseen repository, **not** discovery of
unknown property naming conventions. In an application-only copy of commit
`e78775e`, the resulting analyzer identified one Spring service on port 8080 with Actuator
and external OpenAI configuration; it generated one Container App, secure key
binding, endpoint binding, and a defaultable API-version binding. It did not
invent an Azure OpenAI resource, matching the reference's external-resource
choice. Generated main/service Bicep compiled and the application packaged
with Maven. A detected discrepancy (the Spring API-version default was
initially treated as required) was corrected using azd's optional substitution
syntax. No Azure deployment or API request was performed, so runtime parity is
**not yet established**. Supplied endpoint/key values are still required, and
the current init provider cannot check them before the first infrastructure
phase. The reference's non-root Java 17 image and Container App sizing/scale
are not reconstructed; those are deployment-policy differences, not proven
functional blockers. The candidate was checked through the analyzer and
generator before the live trial below.

**Live bare-`up` result (`roryp/sora`, September 23, 2026):** From an
application-only copy with no `azure.yaml`, the custom azd binary invoked the
installed extension and generated the project. On the corporate subscription
in West US 3, `azd up` created `rg-sora-ext-live-0923`, Container Registry,
Log Analytics, and a Container Apps environment. The first remote image build
failed because the generated root-module Dockerfile tried to copy
`/app/./target/*.jar` even though Maven produced `/app/target/*.jar`. After
fixing the generator and rebuilding the extension, a retry deployed the
Container App. Its home page and Actuator health/readiness endpoints returned
HTTP 200, and the deployed app had endpoint, API-version, and secret-backed
API-key bindings. The test used a deliberately nonfunctional key because the
available Foundry account disables key authentication; **video generation was
not validated**. The extension did not provision an AI resource. This trial
checks the bare-`up` experience, not parity with the repository's former azd
configuration.

### Azure-focused directional cohort

A second search deliberately targeted repositories whose metadata mentioned
Azure and Spring Boot, plus exact `spring-cloud-azure-starter` dependency
matches. Metadata matches were inspected for Spring build files, Azure SDK or
Spring Cloud Azure dependencies, Azure deployment plugins and workflows, and
Azure-specific Terraform, Bicep, or `azure.yaml`.

The metadata cohort contained 49 Spring Boot repositories:

- five were Microsoft or Azure-authored;
- 14 were explicitly samples, tutorials, learning projects, or proofs of
  concept;
- 33 were independent repositories not explicitly classified as samples;
- 24 of those 33 had zero stars, so the cohort is weak evidence of broad
  adoption;
- 16 of the 33 had a concrete Azure integration, deployment, or IaC artifact;
- 10 used Spring Cloud Azure or an Azure SDK;
- six used an Azure deployment plugin or workflow;
- five had Azure-specific Terraform or Bicep;
- none had `azure.yaml`.

The exact dependency search was dominated by Microsoft implementation and
sample repositories, but it also found established independent projects:

- [OWASP WrongSecrets](https://github.com/OWASP/wrongsecrets) uses Spring Cloud
  Azure Key Vault integration and includes Azure Terraform alongside AWS and
  Google Cloud configurations.
- [GreenCity](https://github.com/ita-social-projects/GreenCity) uses Spring
  Cloud Azure identity support and deploys through an existing Helm-oriented
  project structure.
- [OHDSI Perseus](https://github.com/OHDSI/Perseus) uses Spring Cloud Azure for
  identity, Key Vault, Blob Storage, and Redis across multiple modules, while
  retaining its existing Docker-based service structure.
- [Hogwarts Artifacts Online](https://github.com/Washingtonwei/hogwarts-artifacts-online)
  is a Spring sample that uses Key Vault and a direct App Service GitHub
  Actions workflow.

This provides stronger evidence for **Azure service integration demand** than
for **azd adoption**. The observed projects already divide responsibilities
among Maven, Docker, Terraform, Helm, and direct deployment workflows. An azd
extension would need to:

- discover and preserve those existing authorities;
- import or compose with existing infrastructure rather than regenerate it;
- provide clear incremental value for identity, resource binding,
  environment lifecycle, or migration; and
- avoid claiming ownership when the repository already has a working
  deployment path.

The likely greenfield opportunity remains applications that have Azure service
dependencies but no environment deployment model. For established projects,
the likely opportunity is an explicit migration or adoption command with a
previewed ownership boundary.

### Issue-history evidence

An initial issue-history pass reviewed 1,956 issues from Spring Cloud Azure,
Azure Java SDK Spring issues, and the Azure Maven and Gradle plugins. Keyword
classification was used only for triage; representative issues were then read
manually.

The clearest recurring theme is **resource-ownership conflict**, not demand for
more implicit provisioning:

- A Gradle Functions user provisions the Function App, plan, storage, and
  Application Insights through Bicep and requests a deployment-only mode that
  will not recreate them
  ([microsoft/azure-gradle-plugins#153](https://github.com/microsoft/azure-gradle-plugins/issues/153)).
- A Maven Functions user similarly states that Bicep owns the Function App,
  slots, and monitoring, and that Maven should update only the code
  ([microsoft/azure-maven-plugins#2052](https://github.com/microsoft/azure-maven-plugins/issues/2052)).
- An early Web Apps plugin issue reports subscription-limit and cleanup
  failures because each deployment created a new App Service plan rather than
  reusing an existing plan
  ([microsoft/azure-maven-plugins#62](https://github.com/microsoft/azure-maven-plugins/issues/62)).
- One user explicitly asked for Maven to deploy an ARM template, but the answer
  was that the Maven plugin did not provide that capability
  ([microsoft/azure-maven-plugins#1539](https://github.com/microsoft/azure-maven-plugins/issues/1539)).

Identity and binding cross the infrastructure/runtime boundary repeatedly:

- Gradle Functions requires a storage connection string even when the Function
  App and role assignments use managed identity
  ([microsoft/azure-gradle-plugins#193](https://github.com/microsoft/azure-gradle-plugins/issues/193)).
- Maven Functions deployment cannot yet use managed identity for its storage
  operation in some plans
  ([microsoft/azure-maven-plugins#2550](https://github.com/microsoft/azure-maven-plugins/issues/2550)).
- Adding Key Vault access to an existing Function App changed its storage
  configuration and subsequently broke Maven deployment
  ([microsoft/azure-maven-plugins#1848](https://github.com/microsoft/azure-maven-plugins/issues/1848)).
- Spring Cloud Azure users encounter runtime failures when a user-assigned
  identity or its authorization is configured incorrectly
  ([microsoft/spring-cloud-azure#1060](https://github.com/microsoft/spring-cloud-azure/issues/1060)).

Other issues show gaps in multi-module packaging, deployment slots, blue-green
deployment, and Service Bus topic/subscription creation. These validate real
workflow complexity, but each can also be addressed within a build plugin,
runtime library, or existing IaC system.

The evidence therefore supports:

> There is real outer-loop pain, but public users frequently prefer explicit
> separation between infrastructure ownership and application deployment.

This is not direct evidence that users want azd. It is evidence that an azd
extension would need explicit modes:

1. **Generate and provision** a new environment.
2. **Adopt and connect** existing infrastructure without replacing it.
3. **Deploy only** while another system remains the infrastructure authority.

Automatic resource creation should be previewed and opt-in when existing
infrastructure or deployment configuration is detected.

### azd lifecycle ownership tension

azd separates the lifecycle into standalone commands:

```text
azd provision -> create or update infrastructure
azd deploy    -> package and deploy application services
azd up        -> package, provision, publish, and deploy
```

However, `azd up` is not only a convenience wrapper. The current implementation
builds one execution graph across package, provision, publish, and deploy. It
initializes the project's provisioning provider before running either the
default graph or a custom `workflows.up` definition. When no provider is
specified, core azd defaults to Bicep.

`azd deploy` is independently executable and does not run the provisioning
manager. It still requires a configured subscription and enough environment or
project information to resolve the target resource.

This means the proposed extension modes do not map equally to the current
command model:

| Ownership mode | Current command fit |
|---|---|
| azd generates and owns the environment | `azd up` is the intended experience |
| azd runs existing Bicep or Terraform | `azd up` fits only if the user transfers provisioning execution to azd |
| another pipeline owns infrastructure | `azd deploy` fits; `azd up` implies too much ownership |
| azd only connects existing resources | Requires explicit environment import and binding behavior |

A custom `workflows.up` definition cannot fully express deploy-only ownership
because `azd up` initializes infrastructure before dispatching that workflow.
There is also no built-in `infra.provider: none` mode.

The consequence is:

> The extension cannot promise that every adopted Spring application uses
> `azd up` without either taking infrastructure ownership or requiring a core
> lifecycle change.

Possible core designs include an explicit skip-provision mode, a no-op
provisioning provider, or workflow phase selection that occurs before provider
initialization. Until then, an adopted project should be instructed to use
`azd deploy` when another system remains the infrastructure authority.

Teardown must follow the same boundary. `azd down` is appropriate only for
resources azd is authorized to own and delete; an adopted deployment-only
project must not imply that azd can remove shared or externally managed
infrastructure.

### Representative workflow comparison

| Scenario | Existing authority | Best azd role | Assessment |
|---|---|---|---|
| New Spring service with PostgreSQL and no IaC | None | Generate, provision, bind, and deploy through `azd up` | Strongest fit |
| Existing Spring service with Bicep or Terraform pipeline | Existing IaC and CI | Adopt bindings and use `azd deploy`, unless provisioning ownership is explicitly transferred | Useful only with strict ownership preservation |
| One service deploying to an existing App Service | Maven or Gradle plugin | Usually none, or optional deploy-only integration | Build plugin is likely simpler |
| Multi-service application with shared databases, messaging, identity, and monitoring | Mixed or absent | Compose the environment and coordinate deployment | Strong azd differentiation |
| Established Helm or GitOps application | Helm, Kubernetes, and platform tooling | Optional environment bootstrap or migration assistance | Replacing the deployment path is weak |
| Azure Spring Apps migration | Existing platform plus migration tooling | Generate a reviewable target environment and migrate service bindings | Strong transitional use case |

The comparison narrows the likely product:

- `azd up` is the primary greenfield and migration experience.
- `azd deploy` is the safer entry point for externally provisioned
  environments.
- Existing IaC import must not silently become azd ownership.
- A single-command experience across all modes requires additional core azd
  lifecycle semantics.

### Cross-ecosystem ownership evidence

The same ownership pressure appears in non-Azure tools. It is not specific to
Spring or Azure.

#### Jib: intentionally narrow

Jib primarily builds and publishes container images. Its multi-module issue
history concentrates on classpaths, parallel builds, reproducibility, and
registry behavior rather than infrastructure import or deletion. This narrow
boundary makes Jib easy to compose with Terraform, Helm, Kubernetes, or a cloud
deployment system.

#### Eclipse JKube: pressure begins at manifest ownership

JKube extends further into Kubernetes resource and Helm generation. Users then
request:

- packaging and publishing an existing Helm chart without asking JKube to
  generate it
  ([eclipse-jkube/jkube#865](https://github.com/eclipse-jkube/jkube/issues/865));
- generating a dependency-only Helm chart without creating Kubernetes
  resources owned by the Java project
  ([eclipse-jkube/jkube#2112](https://github.com/eclipse-jkube/jkube/issues/2112));
  and
- selectively skipping undeploy for modules whose lifecycle should not be
  controlled by the current invocation
  ([eclipse-jkube/jkube#1358](https://github.com/eclipse-jkube/jkube/issues/1358)).

#### AWS Copilot: full environment ownership creates adoption boundaries

AWS Copilot is closer to azd because it owns application and environment
resources. Its issue history includes requests to:

- combine Copilot-managed services with Terraform, Pulumi, and resources
  Copilot does not support
  ([aws/copilot-cli#3243](https://github.com/aws/copilot-cli/issues/3243));
- run operational commands against services created outside Copilot
  ([aws/copilot-cli#5528](https://github.com/aws/copilot-cli/issues/5528));
- import an existing hosted zone and prevent application deletion from
  deleting it
  ([aws/copilot-cli#3193](https://github.com/aws/copilot-cli/issues/3193)); and
- preview every resource that an application deletion will remove
  ([aws/copilot-cli#5408](https://github.com/aws/copilot-cli/issues/5408)).

#### AWS SAM: convenience resources create cleanup and import problems

AWS SAM users report similar friction:

- retained databases and other persistent resources cannot easily be imported
  into a recreated stack
  ([aws/aws-sam-cli#1699](https://github.com/aws/aws-sam-cli/issues/1699));
- existing resources cannot always be referenced where the underlying
  CloudFormation ownership model requires creation
  ([aws/aws-sam-cli#1061](https://github.com/aws/aws-sam-cli/issues/1061)); and
- automatically created companion stacks simplify first deployment but can
  leave resources behind after deletion
  ([aws/aws-sam-cli#2856](https://github.com/aws/aws-sam-cli/issues/2856)).

In the companion-stack discussion, users explicitly describe keeping Terraform
as the infrastructure authority and using SAM only for the function. This
closely matches the separation observed in Azure Maven and Gradle plugin
issues.

The cross-ecosystem result is:

> Full-stack lifecycle convenience is valuable for greenfield applications,
> but it creates an adoption ceiling when projects already have infrastructure
> ownership, shared resources, or platform automation.

This validates the extension's need for explicit ownership modes, but still
does not validate azd demand specifically. The safest detection sequence is:

```text
detect application and existing deployment authorities
    -> report evidence
    -> ask which lifecycle azd may own
    -> preview generated or adopted state
    -> provision or deploy only within that boundary
```

It should not be:

```text
detect dependency
    -> assume ownership
    -> create resource
```

### Representative repository fit study

The prototype was compared against 10 maintained Spring applications selected
for variation rather than statistical representation. The default credibility
threshold was 100 GitHub stars plus recent maintenance. Lower-popularity
projects were excluded unless they represented a distinct organizational or
ownership case.

The sample intentionally covers:

- single-module and multi-module applications;
- Maven and Gradle;
- no deployment automation, container CI, Helm, and Kubernetes;
- one database and multiple alternative backing services; and
- an application with existing Azure Terraform ownership.

Popularity improves confidence that the repositories represent sustained
projects, but it also biases the sample toward mature applications that already
have deployment conventions. It therefore tests adoption boundaries better
than the greenfield opportunity.

The current prototype has a narrower boundary than the proposed product. It
requires:

- a deployable Maven project at the inspected root;
- a single root module;
- the Spring Boot Maven plugin;
- Spring Boot Actuator; and
- a PostgreSQL runtime dependency for database provisioning.

Gradle, nested applications, multi-module service discovery, MySQL, Redis,
messaging, search services, and existing infrastructure adoption are not
implemented.

| Repository | Public evidence | Current prototype | Hypothetical azd role | Assessment |
|---|---|---|---|---|
| [FlyCms](https://github.com/sunkaifei/FlyCms) | Maven application under `backend`; MySQL; no Docker, IaC, or CI deployment detected | Rejected: nested application, no Actuator, and MySQL is unsupported | Generate a new environment while preserving the application build | Strong greenfield shape, but useful generation requires MySQL and likely Solr analysis |
| [financial-management](https://github.com/LuoDi-Nate/financial-management) | Single-module Maven; MySQL; Docker, Compose, CI, and image publication | Rejected: no Actuator and MySQL is unsupported | Provision an Azure environment around the existing image path | Conditional; azd adds environment lifecycle, but the application already has a complete packaging workflow |
| [aiworkdeck](https://github.com/zeweihan/aiworkdeck) | Multi-module Maven; Docker and Compose; PostgreSQL and MySQL signals | Rejected: multi-module root | Discover services, ask which database is authoritative, and compose the environment | Promising composition case, but dependency inference alone would make an unsafe database choice |
| [MateClaw](https://github.com/mateaix/mateclaw) | Multi-module Maven; Docker, Compose, Kubernetes; PostgreSQL, MySQL, and Redis signals | Rejected: multi-module root | Optional Azure environment or migration path that preserves Kubernetes assets | Conditional; potentially valuable resource graph, but Kubernetes already owns deployment |
| [BookLore](https://github.com/booklore-app/booklore) | Gradle; Docker, Compose, Helm, CI, image publication; MySQL | Rejected: Gradle is unsupported | Adopt existing deployment artifacts or provide an optional Azure environment | Weak; one-service deployment is already well covered |
| [Grimmory](https://github.com/grimmory-tools/grimmory) | Gradle; Docker, Compose, Helm, CI, image publication; MySQL | Rejected: Gradle is unsupported | Adopt or migrate the existing container and Helm workflow | Weak; replacing the existing path offers little visible incremental value |
| [Pig](https://github.com/pig-mesh/pig) | Large multi-module Spring Cloud system; Docker and Compose; PostgreSQL, MySQL, and Redis signals | Rejected: multi-module root | Compose shared Azure resources and multiple deployable services | Potentially strong azd differentiation, but discovery and configuration complexity are far beyond the prototype |
| [metasfresh](https://github.com/metasfresh/metasfresh) | Mature multi-module ERP; image publication; PostgreSQL, Redis, and RabbitMQ signals | Rejected: multi-module root | Optional migration or environment composition with explicit service selection | Weak for automatic adoption; mature operational assumptions and shared services make inference risky |
| [Conductor](https://github.com/conductor-oss/conductor) | Large Gradle multi-module system; Docker and Compose; several alternative databases, queues, and storage systems | Rejected: Gradle and multi-module layouts are unsupported | Explicit deployment profile selection, not dependency-driven generation | Poor automatic-generation fit; dependencies represent supported alternatives rather than requested infrastructure |
| [OWASP WrongSecrets](https://github.com/OWASP/wrongsecrets) | Single-module Maven with Actuator; existing Azure, AWS, and Google Cloud Terraform; Azure Kubernetes and Key Vault scenarios | Structurally accepted, but no PostgreSQL resource would be inferred and existing Terraform ownership would be ignored | Connect to or inspect the existing Terraform-managed environment | Not an adoption target because it is educational, but a strong non-disclosure and ownership-preservation test |

The immediate result is:

- only one of the 10 repositories meets the prototype's structural entry
  requirements;
- that repository is not a useful automatic-provisioning target because its
  infrastructure is already explicitly owned by multi-cloud Terraform; and
- none of the 10 is immediately well served by the prototype without changing
  the application or ignoring an existing deployment authority.

This does not invalidate the live-tested prototype. It shows that the prototype
validated the generation and deployment mechanism, not representative project
compatibility.

The largest applicability gaps are, in order:

1. multi-module and nested-application discovery;
2. Gradle support;
3. removal of Actuator as an eligibility requirement;
4. evidence-backed selection among alternative databases and services; and
5. ownership-aware reuse of Docker, Helm, Kubernetes, Terraform, and CI.

The hypothetical product appears useful in three of the 10 cases:

- FlyCms as a new-environment flow;
- aiworkdeck as an application-and-database composition flow; and
- Pig as a multi-service environment composition flow.

The other cases are conditional or weak because their existing deployment
authority already solves much of the outer-loop problem. This supports a
narrower product target:

> Focus first on credible Spring applications with multiple environment
> dependencies but no established infrastructure authority. Treat mature
> Docker, Helm, Kubernetes, Terraform, and deployment-plugin projects as
> adoption or migration scenarios, not automatic generation scenarios.

### Second representative repository cohort

A second cohort tested 10 additional active applications with approximately
6,600 to 85,000 GitHub stars. It emphasizes popular production-style
applications and platforms rather than repeating the first cohort's smaller
and medium-sized projects.

| Repository | Public evidence | Current prototype | Hypothetical azd role | Assessment |
|---|---|---|---|---|
| [mall](https://github.com/macrozheng/mall) | Multi-module Maven commerce application; Docker and Compose; MySQL, Redis, MongoDB, RabbitMQ, Elasticsearch, and object-storage signals | Rejected: root is a multi-module aggregator | Provision a selected Azure backing-service profile while retaining existing containers | Potentially valuable composition, but automatic creation from the complete dependency set would substantially over-provision |
| [Apollo](https://github.com/apolloconfig/apollo) | Multi-module Maven configuration platform; Docker and CI; MySQL, PostgreSQL, and Redis signals | Rejected: multi-module root and no root Actuator dependency | Deploy an explicitly selected supported topology | Weak automatic fit; this is infrastructure software with its own topology and operational model |
| [eladmin](https://github.com/elunez/eladmin) | Multi-module Maven administration application; MySQL, Redis, and storage signals; no repository deployment authority detected | Rejected: multi-module root without the required root plugin and Actuator declarations | Generate a new environment after identifying the deployable module | Strong candidate if module discovery and MySQL or Redis support exist |
| [litemall](https://github.com/linlinjava/litemall) | Multi-module Maven commerce application; Docker, Compose, and CI; MySQL | Rejected: multi-module aggregator | Compose the application modules with managed MySQL while preserving containers | Conditional; azd could add environment lifecycle, but an existing Compose path already covers local composition |
| [newbee-mall](https://github.com/newbee-ltd/newbee-mall) | Single-module Maven application; MySQL and Redis; no Docker, Compose, or CI deployment detected | Rejected: no Actuator, and MySQL is unsupported | Generate hosting, MySQL, Redis, bindings, and deployment | Strongest second-cohort greenfield candidate |
| [ZFile](https://github.com/zfile-dev/zfile) | Single-module Maven storage application; Docker; MySQL, Redis, and several storage-provider signals | Rejected: no Actuator, and MySQL is unsupported | Deploy the container and connect to an explicitly selected storage provider | Conditional; storage dependencies represent user choices and existing accounts, not permission to create every service |
| [Komga](https://github.com/gotson/komga) | Gradle media server; Docker and CI; no required external backing service detected | Rejected: Gradle is unsupported | Deploy the existing container | Weak; azd adds little over the established single-container workflow |
| [JetLinks Community](https://github.com/jetlinks/jetlinks-community) | Large multi-module Maven IoT platform; Docker, Compose, and CI; PostgreSQL, Redis, MongoDB, and Elasticsearch signals | Rejected: multi-module aggregator | Compose an explicit platform deployment profile | Conditional; potentially valuable environment coordination, but unsafe to infer from dependencies and too complex for generic generation |
| [Apereo CAS](https://github.com/apereo/cas) | Gradle identity platform; Docker, Compose, Kubernetes, and CI; several optional databases, queues, and integrations | Rejected: Gradle is unsupported | Adopt an existing deployment overlay or assist migration | Poor automatic-generation fit; its integrations are configuration choices rather than a single required resource graph |
| [music-website](https://github.com/Yin-Hongwei/music-website) | Maven server nested beside client and administration applications; Docker, Compose, and CI; MySQL, Redis, and storage signals | Rejected: no deployable project at the repository root | Compose the three application components and managed dependencies | Conditional; meaningful multi-component value exists, but the current repository already defines its container topology |

None of these 10 applications can enter the current prototype flow:

- five have multi-module Maven roots;
- two use Gradle;
- one has a nested Maven service rather than a root application; and
- the two root single-module Maven applications lack Actuator and use MySQL.

The dependency distribution also challenges the prototype's PostgreSQL-first
scope:

| Signal | Repositories |
|---|---:|
| MySQL | 7 |
| Redis | 8 |
| PostgreSQL | 3 |
| MongoDB | 3 |
| Elasticsearch | 2 |
| Docker | 8 |
| Compose | 5 |

These are capability signals, not confirmed deployment requirements. In
particular, storage providers, databases, and queues are often supported
alternatives.

Across both cohorts:

- one of 20 repositories meets the prototype's structural eligibility rules;
- zero of 20 is immediately well served by its current generated resource
  graph;
- four repositories are plausible strong targets for a broader greenfield or
  composition product: FlyCms, aiworkdeck, Pig, and newbee-mall;
- eladmin is an additional strong candidate once the deployable module is
  identified; and
- the remaining projects are conditional, migration-oriented, or poor fits
  because their topology or deployment authority is already established.

The second cohort reinforces that ownership-aware behavior is necessary but
not sufficient. Before testing adoption modes deeply, a representative
implementation needs:

1. Maven and Gradle parity;
2. nested and multi-module service discovery;
3. health-check discovery without requiring Actuator;
4. MySQL and Redis analysis before additional specialized services; and
5. explicit deployment-profile selection when dependencies represent
   alternatives.

It does not follow that all five capabilities should be implemented
immediately. The desk analysis should first be used to narrow the initial
supported project shape and identify an adoption hypothesis that can be tested
without claiming general Spring compatibility.

### Concierge trial validation

The next validation should test whether azd creates measurable value for a
developer who clones a cloud-neutral application and wants a disposable Azure
deployment. It should not require implementing generic Gradle, multi-module,
MySQL, Redis, and storage-provider support first.

Use four contrasting applications:

| Application | Validation purpose |
|---|---|
| newbee-mall | Simple Maven application with MySQL and Redis but no detected deployment configuration |
| eladmin | Multi-module application with several dependencies and no detected deployment authority |
| ZFile | Existing Docker packaging with ambiguous storage-provider choices |
| Komga | Single-container control case where azd may add little incremental value |

For each fresh clone:

1. Audit the operational contract: build command, artifact, port, health
   behavior, environment variables, persistence, initialization, and required
   services.
2. Record whether each fact came from build files, application configuration,
   Compose, documentation, or an assumption.
3. Manually create the `azure.yaml`, Bicep, bindings, and deployment
   configuration that the hypothetical extension should generate.
4. Record every material decision that would require a prompt.
5. Deploy into an isolated resource group and exercise meaningful application
   behavior.
6. Restart the application and verify persistent state.
7. Run `azd down` and verify that it removes all azd-owned resources and
   nothing external.
8. Compare the experience with the repository's documented Docker or manual
   deployment path.

The benchmark should compare:

- commands and material decisions;
- elapsed time;
- Azure knowledge required;
- source changes;
- unsupported assumptions;
- resulting functionality; and
- cleanup completeness.

Success criteria:

- at least three of four applications deploy without source changes;
- at least 90% of generated settings are supported by explicit repository
  evidence;
- no service is provisioned solely because a dependency exists;
- no more than three material user decisions are required for a trial;
- persistent data survives an application restart;
- teardown removes every azd-owned resource and no external resource; and
- the azd path requires meaningfully less Azure-specific work than the manual
  baseline.

An independent evaluator should receive only the original repository and the
generated plan, not the research notes. Successful reproduction would show
that the plan is understandable without the prototype author's context.

This benchmark can validate technical usefulness after a user chooses azd. It
cannot establish willingness to discover, install, or adopt azd. That requires
product telemetry, public behavior, or direct user evidence.

### Remaining evidence

Repository inspection cannot reveal infrastructure owned elsewhere or why a
team selected its current workflow. The next evidence should therefore focus
on:

- comparable signals from public support forums;
- Maven Central usage trends as a supporting signal for Spring Cloud Azure and
  Azure deployment plugins;
- task comparisons on representative greenfield, existing-IaC, and
  multi-service applications; and
- explicit replacement, complement, and migration rules for each existing
  workflow shape.

Stars, downloads, and documentation volume should remain supporting signals,
not adoption proof.
