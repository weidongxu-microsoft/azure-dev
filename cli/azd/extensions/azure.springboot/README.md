# Azure Spring Boot extension prototype

This prototype targets:

```text
existing Spring Boot repository -> azd up -> Azure Container Apps
```

The application repository does not need an `azure.yaml`, Dockerfile, or
infrastructure files.

## Analysis and generation

Analysis and generation are separate:

1. Spring inspection identifies deployable services, ports, health capability,
   and configured service dependencies.
2. azd's existing Java detector resolves the effective Maven model.
3. Runtime dependencies become evidence-backed resource requirements.
4. Generation consumes only the analysis result.

Container Apps, Container Registry, Log Analytics, and managed identity are the
current hosting policy. Other resources are conditional:

| Evidence | Generated resource and binding |
| --- | --- |
| PostgreSQL runtime dependency | PostgreSQL Flexible Server, database, Key Vault credential, datasource environment variables, and `db.postgres` metadata |
| Spring AI OpenAI starter, Azure Identity, and `AZURE_INFERENCE_*` configuration | Microsoft Foundry resource, model deployment, managed identity access, inference environment variables, and `ai.openai.model` metadata |
| `azure.openai.endpoint` and `azure.openai.api-key` properties referencing `AZURE_OPENAI_ENDPOINT` and `AZURE_OPENAI_API_KEY` | External Azure OpenAI endpoint and secure Container App API-key secret; no inferred Azure OpenAI resource or model |
| A service URL property referencing another detected service | Deployment ordering, internal ingress for the dependency, and the configured service URL environment variable |

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

## Acceptance application

`testdata/spring-todo` is a Spring Data JDBC application with a standard
PostgreSQL runtime dependency. It contains no Azure-specific application
dependency or configuration. Tests use H2 in PostgreSQL compatibility mode.

The generated PostgreSQL binding currently uses a generated administrator
credential stored in Key Vault. A formal feature should create a least-privilege
database principal and prefer Microsoft Entra authentication.

`testdata/spring-orders-microservices` is a Maven reactor with a shared library,
an internal catalog service, and a public orders service using PostgreSQL. It
validates that libraries remain build dependencies without becoming deployment
units.

## Current boundaries

- Root or multi-module Maven projects with explicit Spring Boot service modules.
- Maven must be trusted because effective-model resolution can load Maven core
  extensions.
- PostgreSQL and Microsoft Foundry Models are the application resource requirements currently translated;
  direct Azure OpenAI configuration is treated as an external dependency.
- Service dependencies require an explicit `*_SERVICE_URL` property contract.
- Gradle, profile resolution, and existing-resource selection remain future
  work.
