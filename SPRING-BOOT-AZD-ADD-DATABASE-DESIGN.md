# Spring Boot `azd add` database design questions

This note records the proposed experience discussed for
[#9960](https://github.com/Azure/azure-dev/issues/9960). It is a design
direction, **not** implemented behavior or an agreed extension contract.

## Current direction (October 13, 2026)

The first milestone is explicit `azd spring init`, which generates
`azure.yaml` and infrastructure, followed by standard `azd up`. It does not
edit the application's POM, Spring configuration, business logic, or schema.
The earlier no-`azure.yaml`/`azd add` proposal below is future design research,
not a prerequisite for this milestone.

Managed identity is the intended Azure authentication approach. The prototype
already uses it for Foundry, ACR pulls, and Key Vault secret access, but its
PostgreSQL server/binding still uses an administrator password and disables
Entra authentication. The before/after PostgreSQL cases pass generation and
packaging; they do not prove a passwordless connection or chat persistence.

For a non-admin PostgreSQL app identity, Bicep enables Entra authentication and
configures an Entra administrator; SQL creates the identity-mapped database role
and grants its permissions. A generated, idempotent `postprovision` script is
the proposed bridge, requiring administrator access and network connectivity.
Neither this script nor passwordless PostgreSQL support is implemented.
[Bicep can instead configure a service principal as Entra administrator](https://learn.microsoft.com/en-us/azure/templates/microsoft.dbforpostgresql/2024-08-01/flexibleservers/administrators),
but giving the app administrator privileges is not least-privilege access.

See the [implementation handoff](./SPRING-BOOT-AZD-EXTENSION-IMPLEMENTATION-INVESTIGATION.md)
and [current tests](./cli/azd/extensions/azure.springboot/README.md).

## Historical direction and ownership

- Start with an existing Spring Boot repository **without `azure.yaml`**.
  In the experience discussed here, neither `azd add` nor `azd up` generates
  one. This differs from the current prototype, which materializes
  `azure.yaml`, Dockerfiles, and Bicep through `azd spring init`, and from the
  materialization proposed in #9840. The persistence model needs a decision
  before implementation.
- Spring build and configuration files describe the application. Azure-only
  additions or overrides must not become a second copy of the Spring module
  structure. If `azure.yaml` is used for those additions in another design,
  project referencing, merge ownership, and output remapping matter to *both*
  existing-app deployment and `azd add` (#9839, #9768, #9771).
- `azd add` expresses developer intent and may, with review and consent, edit
  the existing Spring project. `azd up` re-analyzes it, resolves ownership of
  its dependencies, provisions selected Azure resources, and binds their
  outputs to Spring runtime configuration. The application must not depend on
  azd to understand its build or its business logic.

This follows the broader Spring extension principles: preserve Spring as the
application source of truth; use framework-specific discovery with evidence
and conflict reporting; keep Azure additions separate from inferred
application facts; make outputs consumable as Spring settings; use shared azd
extension, discovery, and composition contracts; and validate against real
applications, not just synthetic fixtures.

Two further goals depend on those contracts. An Azure integration selected
through `azd add` must include both resource/binding work and, when necessary,
explicitly approved Spring edits; adding infrastructure alone does not make an
unused dependency useful. A source-based Azure Spring Apps-to-Container Apps
path should reuse Spring analysis and bindings, preserve application builds,
and expose unsupported platform features as migration decisions. It must not
imply automatic production data or traffic cutover. See
[#9972](https://github.com/Azure/azure-dev/issues/9972).

## Example: add PostgreSQL to a database-free chat app

The example is a Spring Boot 3.5.x Maven app that already uses Foundry but has
no datasource. Selecting "add PostgreSQL connection" must not implicitly mean
"store chats" or "add a vector store." A proposed, opt-in application diff is:

```xml
<!-- pom.xml: use an existing compatible BOM if present -->
<dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>com.azure.spring</groupId>
      <artifactId>spring-cloud-azure-dependencies</artifactId>
      <version>6.5.0</version>
      <type>pom</type>
      <scope>import</scope>
    </dependency>
  </dependencies>
</dependencyManagement>

<!-- Inside the existing dependencies element -->
<dependency>
  <groupId>org.springframework.boot</groupId>
  <artifactId>spring-boot-starter-jdbc</artifactId>
</dependency>
<dependency>
  <groupId>com.azure.spring</groupId>
  <artifactId>spring-cloud-azure-starter-jdbc-postgresql</artifactId>
</dependency>
```

The Azure starter includes the PostgreSQL driver. The starter and BOM shown
here are for Spring Cloud Azure's Entra/passwordless JDBC path, **not** a
requirement for connecting to any PostgreSQL database. For ordinary
credential-based JDBC, the PostgreSQL driver and an appropriate datasource
setup suffice; MyBatis, JPA, and Spring Data JDBC are separate application
choices. Resolve dependencies against the *effective* Maven/Gradle model and
the actual Spring Boot version; do not duplicate existing dependencies.

```properties
# src/main/resources/application.properties: illustrative proposed contract
spring.datasource.url=${AZD_POSTGRES_JDBC_URL}
spring.datasource.username=${AZD_POSTGRES_USERNAME}
spring.datasource.azure.passwordless-enabled=true
```

`AZD_POSTGRES_*` names are hypothetical, not azd or Spring standards. Their
presence alone must **not** be treated as proof that azd owns the database.
The extension must define a durable, inspectable, project-associated intent
for "provision new" versus "connect existing" that survives later `azd up`,
works for multiple services/databases, and does not require `azure.yaml` in
this design. The exact format and precedence over other configuration are
unresolved. The extension supplies per-environment URL and username at
runtime; it must not commit credentials or inject a static Azure hostname
into application source. It must also address the local run when these
placeholders are unset, rather than breaking an existing workflow silently.

**Java code and schema: no change for connection-only.** Do not add entities,
repositories, queries, migrations, mock CRUD code, or chat-memory behavior.
The resulting datasource may be usable, but the existing chat logic does not
read or write PostgreSQL. An opt-in chat-memory or pgvector integration would
be a *different* feature and diff.

## Inference, provisioning, and proof

The PostgreSQL driver, MyBatis, an Azure JDBC starter, a JDBC URL, and
`passwordless-enabled=true` each describe different things: data-access API,
driver/authentication, destination, and authentication intent. **None alone
establishes database ownership.** A concrete URL may point to a separately
managed server; a generic `${DATABASE_URL}` could be supplied by another
system. Existing projects without a recorded ownership choice need an
explicit decision; `--no-prompt` must not silently create a database.

For "provision new," `azd up` needs to create the database, wire its outputs
to the selected Spring service, configure network access and Entra identities,
and create the required database principal and grants. Azure RBAC by itself
does not give a JDBC client SQL access. For "connect existing," leave
provisioning to its owner and verify that the app can reach and authenticate
to the selected server. Whether a connection is *passwordless* cannot be
proved by the starter or property alone: check effective configuration and
prove a connection through the app's datasource without a password fallback.
Provisioning success does not demonstrate that application code uses the DB.

## What the issues currently cover

| Issue | Relevant scope or gap |
| --- | --- |
| [#9964](https://github.com/Azure/azure-dev/issues/9964) | Maps outputs to Spring runtime settings; explicitly excludes changes to POM, Gradle, Java, and Spring configuration files. |
| [#9967](https://github.com/Azure/azure-dev/issues/9967) | Adds composable PostgreSQL infrastructure via `azd add`; explicitly excludes existing-application build, source, and Spring configuration edits. |
| [#9977](https://github.com/Azure/azure-dev/issues/9977) | Generates a **new** Spring Boot service with its own POM and code; does not patch an existing service. |
| [#9838](https://github.com/Azure/azure-dev/issues/9838), [#9840](https://github.com/Azure/azure-dev/issues/9840), [#9770](https://github.com/Azure/azure-dev/issues/9770) | Historical pre-project discovery/inference and extension-contributed `azd add` questions. The prototype's no-file hook has been removed; the explicit-init milestone generates `azure.yaml` before standard deployment. |

To deliver this connection-only `azd add` experience, explicitly assign
ownership of **opt-in edits to an existing Spring project**, including preview,
conflict handling, idempotence, and rollback. Do not present #9967 alone as
complete application integration.

Related: [prototype assessment](./SPRING-BOOT-EXTENSION-PROTOTYPE-ASSESSMENT.md).
For the passwordless starter and database-user prerequisites, see
[Spring Cloud Azure PostgreSQL support](https://learn.microsoft.com/azure/developer/java/spring-framework/postgresql-support).
