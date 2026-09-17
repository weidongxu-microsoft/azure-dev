# Azure Spring Boot extension prototype

This prototype targets the following experience:

```text
existing Spring Boot repository -> azd init --from-code -> azd up -> Azure Container Apps
```

The repository does not need an `azure.yaml`, Dockerfile, or infrastructure
files before the first `azd` command.

## How it works

- `testdata/spring-todo` is the initial single-module Maven acceptance
  application.
- `internal/springboot.Detect` reads `pom.xml` and
  `application.properties` without executing application code.
- The `spring-boot` init provider registers with azd before a project exists.
- `azd init --from-code` asks the provider to inspect the repository, then azd
  validates and writes the returned project-relative files.
- Generated files are deterministic and never overwrite existing files.
- Generated infrastructure includes a resource group, Container Registry,
  Log Analytics workspace, Container Apps environment, managed identity,
  AcrPull role assignment, and Container App.

Bare `azd up` inference is not implemented yet. Initialize the repository first:

```text
azd init --from-code
azd up
```
