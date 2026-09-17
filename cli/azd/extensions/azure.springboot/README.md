# Azure Spring Boot extension prototype

This prototype targets:

```text
existing Spring Boot repository -> azd up -> Azure Container Apps
```

The application repository does not need an `azure.yaml`, Dockerfile, or
infrastructure files.

## Analysis and generation

Analysis and generation are separate:

1. Spring inspection identifies the deployable service, port, and health
   capability.
2. azd's existing Java detector resolves the effective Maven model.
3. Runtime dependencies become evidence-backed resource requirements.
4. Generation consumes only the analysis result.

Container Apps, Container Registry, Log Analytics, and managed identity are the
current hosting policy. Other resources are conditional:

| Evidence | Generated resource and binding |
| --- | --- |
| PostgreSQL runtime dependency | PostgreSQL Flexible Server, database, Key Vault credential, datasource environment variables, and `db.postgres` metadata |

If PostgreSQL is not present in the analysis, none of its Bicep, parameters,
secrets, environment variables, or `azure.yaml` entries are generated.

## Acceptance application

`testdata/spring-todo` is a Spring Data JDBC application with a standard
PostgreSQL runtime dependency. It contains no Azure-specific application
dependency or configuration. Tests use H2 in PostgreSQL compatibility mode.

The generated PostgreSQL binding currently uses a generated administrator
credential stored in Key Vault. A formal feature should create a least-privilege
database principal and prefer Microsoft Entra authentication.

## Current boundaries

- One root Maven Spring Boot service.
- Maven must be trusted because effective-model resolution can load Maven core
  extensions.
- PostgreSQL is the only application resource requirement currently translated.
- Gradle, multi-service translation, profile resolution, and existing-resource
  selection remain future work.
