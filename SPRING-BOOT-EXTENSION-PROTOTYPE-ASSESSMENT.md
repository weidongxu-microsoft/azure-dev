# Spring Boot extension prototype assessment

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
existing single-module Maven application
    -> dependency-driven PostgreSQL inference
    -> generated azure.yaml and Bicep
    -> Azure Container Apps deployment
```

It does **not** prototype template-based creation, `azd add` for resources or
new Spring services, Gradle, existing container builds, App Service, WAR,
managed Spring components, broader Azure integrations, Spring AI, Spring
Batch, or Azure Spring Apps migration.

## Existing-application flow: critical path

This section covers the flow prototyped in this branch. The other epic tracks
have separate implementation paths and risks that this prototype did not
investigate.

### azd and the extension framework

- Start extensions before project loading so `azd init` and bare `azd up` can
  operate when `azure.yaml` does not exist. Tracked by
  [#9838](https://github.com/Azure/azure-dev/issues/9838),
  [#9840](https://github.com/Azure/azure-dev/issues/9840), and
  [#9978](https://github.com/Azure/azure-dev/issues/9978).
- Select an init provider deterministically, invoke it through a versioned
  extension contract, and return explainable analysis provenance. Tracked by
  [#9838](https://github.com/Azure/azure-dev/issues/9838) and
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

- **Generated-project lifecycle:** ownership, preview, merge, regeneration, and
  upgrade behavior are not defined for existing or previously generated files.
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
- **Relational database binding:** azd lacks a standard post-provision
  data-plane phase for creating least-privilege managed-identity database
  principals and grants. Partly tracked by
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
conditional PostgreSQL generation. It still supports one controlled application
shape and is not yet general Spring Boot support.

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
- the deployed application's health and PostgreSQL-backed REST endpoints work.

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

**No. It works for a narrow, known shape and should currently be described as
a single-module Maven acceptance detector.**

### Currently supported shape

The repository must have:

- a root `pom.xml`;
- one Maven module at the repository root;
- a Spring Boot parent or a direct dependency whose group is
  `org.springframework.boot`;
- `spring-boot-maven-plugin` directly under `build.plugins`;
- `spring-boot-starter-actuator` as a direct dependency;
- a project version directly on the project or parent;
- optional `server.port` as a literal integer in
  `src/main/resources/application.properties`; and
- an executable JAR produced under the root `target` directory.

For this shape, the detector and generator are deterministic and were validated
end to end.

### Important unsupported or unreliable cases

| Area | Current limitation |
| --- | --- |
| Build system | Gradle is unsupported. |
| Maven layout | Multi-module and nested-module repositories are unsupported. |
| Effective POM | Parent inheritance, imported BOMs, profiles, properties, plugin management, and dependency management are not resolved. |
| Maven expressions | Values such as `${revision}` and inherited artifact metadata are not evaluated. |
| Application type | The detector does not prove that the module is an executable web application. |
| Configuration | `application.yml`, profile files, environment placeholders, command-line overrides, and management-server settings are ignored. |
| Ports | Only a literal `server.port` in `application.properties` is recognized; all other cases become port 8080. |
| Actuator | Actuator must be a direct dependency, and standard liveness/readiness paths are assumed. Exposure and probe configuration are not inspected. |
| Java | The POM Java version is recorded but not used; the Dockerfile always builds and runs Java 21. |
| Maven execution | The generated Dockerfile uses a fixed Maven image and `mvn`, not the repository's Maven wrapper or Maven configuration. |
| Artifact selection | `target/*.jar` can select the wrong file or fail when multiple JARs exist. |
| Source layout | The Dockerfile copies only the root `pom.xml` and `src` directory. |
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
