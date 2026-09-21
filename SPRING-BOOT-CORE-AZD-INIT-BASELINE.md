# Spring Boot core azd initialization baseline

## Purpose

Establish what core azd already supports for an existing Java application
before evaluating a Spring-specific init-provider extension.

The key product question is:

> Which Spring-specific detection, binding, health, and deployment gaps remain
> after core `azd init --from-code`, and are they large enough to justify an
> extension rather than improvements to core Java initialization?

## Current core Java support

Core `azd init --from-code`:

- detects Maven projects using the effective POM;
- detects leaf modules under multi-module Maven roots;
- treats supported Maven leaf projects as Java services;
- detects MySQL, PostgreSQL, Redis, and MongoDB runtime dependencies;
- excludes test, provided, system, and import-scoped dependencies;
- generates Java services for Azure Container Apps; and
- gives init-provider extensions the first opportunity to handle the project.

It does not support Gradle and is not Spring-specific. A Maven leaf module can
be treated as deployable even when it is a library.

## Spring Todo experiment

The built-in flow was tested against:

```text
cli/azd/extensions/azure.springboot/testdata/spring-todo
```

The fixture had no `azure.yaml` or generated infrastructure before the test.
The command detected:

```text
Java service at .
PostgreSQL
```

The generated project model was:

```yaml
services:
  spring-todo:
    project: .
    host: containerapp
    language: java

resources:
  spring-todo:
    type: host.containerapp
    uses:
      - todos
    port: 8080

  todos:
    type: db.postgres
```

## Generated Azure architecture

`azd infra gen` produced:

- Azure Container Apps;
- Azure Container Registry;
- a Container Apps environment;
- Log Analytics and Application Insights;
- a user-assigned managed identity;
- Key Vault;
- PostgreSQL Flexible Server; and
- the `todos` database.

This substantially overlaps the Spring extension prototype.

## Spring binding gap

Core generated these generic application variables:

```text
POSTGRES_HOST
POSTGRES_USERNAME
POSTGRES_DATABASE
POSTGRES_PASSWORD
POSTGRES_URL
```

The fixture reads:

```text
SPRING_DATASOURCE_URL
SPRING_DATASOURCE_USERNAME
SPRING_DATASOURCE_PASSWORD
SPRING_SQL_INIT_MODE
```

Spring Boot does not translate the generic `POSTGRES_*` variables into
`spring.datasource.*`. The application would therefore retain its fallback
connection:

```text
jdbc:postgresql://localhost:5432/todos
```

Inside the Container App, that address refers to the application container
rather than Azure PostgreSQL. The generated resource graph is broadly correct,
but its application binding does not satisfy the Spring configuration
contract.

The extension prototype closes this gap by generating the Spring datasource
variables and SQL initialization setting directly.

## Packaging result

Core packaging selected the expected Java Container Apps build path:

```text
Packaging service spring-todo
Packaging service spring-todo (Building Docker image from source)
```

It failed before Maven packaging while fetching the Oryx builder because the
acquired `pack` client used Docker API 1.38 and Docker Desktop required at
least 1.40:

```text
client version 1.38 is too old
```

Setting `DOCKER_API_VERSION=1.40` did not change the client version used by
`pack`. This is a core packaging-tool compatibility blocker, not evidence of a
Spring or Maven build failure.

## Multi-service Maven experiment

Core `azd init --from-code` was tested against a Maven reactor containing:

- `order-contracts`, a shared library;
- `catalog-service`, an internal Spring Boot service on port 8081; and
- `orders-service`, a public Spring Boot service using PostgreSQL and catalog.

Core detected all three modules as Container Apps, including the library,
assigned port 8080 to each, and did not generate the orders-to-catalog
dependency or `CATALOG_SERVICE_URL`. It required interactive confirmation and
a database name, so the initial `--no-prompt` run could not complete.

The Spring extension generated only the two deployable services, preserved the
correct ports, kept catalog internal, ordered catalog before orders, and bound
catalog and PostgreSQL only where required. The generated project completed a
live West US 3 request through orders, catalog, and PostgreSQL.

## Spring AI Todo Agent experiment

The built-in flow was also tested against:

```text
cli/azd/extensions/azure.springboot/testdata/spring-ai-todo-agent
```

This fixture is a Spring Boot 4 application using Spring AI 2.0 tool calling
with an in-memory Todo service. It has no database, `azure.yaml`, Dockerfile, or
infrastructure.

Core `azd init --from-code` detected one generic Java service and proposed
Azure Container Apps. It did not detect the Spring AI capability or an Azure
OpenAI dependency. The generated model was:

```yaml
services:
  spring-ai-todo-agent:
    project: .
    host: containerapp
    language: java

resources:
  spring-ai-todo-agent:
    type: host.containerapp
    port: 8080
```

`azd infra gen` produced a resource group plus seven resources:

- Container App and Container Apps environment;
- Container Registry;
- user-assigned managed identity;
- Log Analytics;
- Application Insights; and
- an Application Insights dashboard.

It did not produce a Microsoft Foundry resource, model deployment, role
assignment, or either inference variable required by the fixture:

```text
AZURE_INFERENCE_ENDPOINT
AZURE_INFERENCE_MODEL
```

The generated Container App only received
`APPLICATIONINSIGHTS_CONNECTION_STRING`, `AZURE_CLIENT_ID`, and `PORT`.

In comparison, the Spring extension recognizes the application's Spring AI,
Azure Identity, and `AZURE_INFERENCE_*` configuration. It adds the Foundry
model deployment, managed-identity access, and runtime bindings that generic
`--from-code` does not provide. This path completed live tool calling in East
US 2.

## Implications

The extension should not be justified as introducing existing-Java-app
initialization. Core azd already supplies that workflow and much of the
resource generation.

A Spring-specific extension is justified only if it materially improves:

- Spring configuration binding;
- Spring-aware health and readiness behavior;
- service and module classification;
- build and packaging compatibility;
- application-specific database initialization; or
- explicit Spring AI resource and configuration setup; or
- migration and adoption behavior that should not be generic to all Java
  applications.

Before adding extension capabilities, compare them with possible improvements
to core Java detection and resource binding. Generic Maven, Gradle, packaging,
and database-resource improvements likely belong in core; behavior that depends
on Spring conventions may belong in the extension.

## Existing Java azd template architectures

The curated awesome-azd gallery currently contains 19 templates tagged with
Java. Eighteen are Microsoft-tagged and one is community-tagged. Only three are
tagged with the Spring framework.

This is evidence of architectures azd currently demonstrates and supports. It
is not an estimate of customer architecture frequency.

### Hosting patterns

| Hosting target | Java templates |
|---|---:|
| Azure Container Apps | 8 |
| Azure Functions | 7 |
| Azure App Service | 3 |
| Azure Kubernetes Service | 2 |

Templates can use more than one hosting target.

### Data and integration services

| Service | Java templates |
|---|---:|
| Azure OpenAI | 8 |
| PostgreSQL | 3 |
| MySQL | 2 |
| Azure Storage | 2 |
| Blob Storage | 2 |
| Azure AI Foundry | 1 |
| Azure AI Search | 1 |
| Azure AI services | 1 |
| Azure SQL | 1 |
| Cosmos DB | 1 |
| Event Grid | 1 |
| Event Hubs | 1 |

### Cross-cutting services

| Capability | Java templates |
|---|---:|
| Managed identity | 14 |
| Application Insights | 7 |
| Azure Monitor | 6 |
| Virtual networks | 5 |
| Key Vault | 2 |
| Log Analytics | 1 |

### Frameworks

| Framework | Java templates |
|---|---:|
| Spring | 3 |
| Quarkus | 3 |
| Semantic Kernel | 2 |
| Jakarta EE/Open Liberty | 1 |
| LangChain4j | 1 |

Other entries combine Java services with JavaScript frameworks or agent
frameworks.

### Representative architecture groups

#### Web application plus managed data

- Spring PetClinic on App Service with Azure SQL, Key Vault, monitoring, and
  Application Insights.
- Quarkus on Container Apps with PostgreSQL or MySQL, managed identity, and
  monitoring.
- Quarkus Todo API on Container Apps with PostgreSQL.

#### Microservices and Kubernetes

- Spring PetClinic microservices on Container Apps with MySQL, Key Vault,
  managed identity, monitoring, and Azure OpenAI.
- Dapr pub/sub Java services on AKS.
- Jakarta EE Cargo Tracker on AKS with PostgreSQL, managed identity, and Azure
  OpenAI.

#### Serverless and event-driven

Java Functions templates cover:

- HTTP triggers;
- timer triggers;
- Event Grid blob triggers;
- Event Hubs triggers;
- Cosmos DB triggers;
- Durable Functions fan-out/fan-in; and
- remote MCP servers.

These commonly combine managed identity, virtual networks, Application
Insights, Storage, and an event source.

#### AI, RAG, and agents

The current gallery has a strong AI skew:

- Spring or Java applications with Azure OpenAI;
- RAG with Azure AI Search;
- LangChain4j and Quarkus;
- Foundry-based multi-agent applications;
- transaction-review and customer-assistant agents; and
- blob-backed knowledge or document inputs.

### Relevance to the Spring extension

The template inventory suggests these capability layers:

1. **Common baseline:** Container Apps or App Service, managed identity,
   monitoring, and application configuration.
2. **Common stateful services:** MySQL, PostgreSQL, or Azure SQL.
3. **Frequent adjacent services:** Storage and Key Vault.
4. **Explicit workload features:** Event Hubs, Event Grid, Cosmos DB, Azure
   OpenAI, AI Search, and Foundry.
5. **Specialized topologies:** Functions, AKS, Dapr, and multi-agent systems.

The extension should not infer every service represented in the gallery.
Relational databases and standard Spring bindings are plausible convention-
based detection targets. Eventing, AI, storage-account purpose, Kubernetes,
and multi-agent topology usually require explicit application configuration or
user intent.
