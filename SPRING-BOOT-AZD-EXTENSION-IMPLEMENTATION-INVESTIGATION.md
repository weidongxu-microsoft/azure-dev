# Spring Boot azd extension implementation investigation

## Objective

Continue investigating
[Azure/azure-dev#9960](https://github.com/Azure/azure-dev/issues/9960), which
proposes a first-party `azure.springboot` extension. The target experience is:

```text
existing Spring Boot repository -> azd up -> running application in Azure Container Apps
```

The developer should not need to duplicate Maven project structure in
`azure.yaml`, write infrastructure, create a Dockerfile, or learn
Spring-specific azd commands.

The detailed findings are in
[`SPRING-BOOT-AZD-INVESTIGATION.md`](./SPRING-BOOT-AZD-INVESTIGATION.md).

## Current state

- A working `azure.springboot` prototype implements extension-driven
  initialization, Spring/Maven analysis, generated `azure.yaml`, Dockerfiles,
  Bicep, PostgreSQL and Foundry resources, and multi-service topology.
- Controlled PostgreSQL, Foundry, and multi-service fixtures have been tested
  locally and through live Azure deployments using bare `azd up` without
  `azure.yaml` on the custom prototype azd binary.
- Existing extension APIs cover the generated-project vertical slice.
- Upstream integration of the init-provider contract and broader inference
  coverage remain open implementation concerns.

## Epic boundaries

The extension must:

- integrate through standard `azd init`, `azd up`, `azd provision`, and
  `azd deploy` workflows;
- keep Spring-specific behavior inside `cli/azd/extensions/azure.springboot/`;
- use Azure Container Apps as the initial default host;
- support Maven executable JAR applications first;
- preserve Spring Boot configuration conventions;
- prefer passwordless authentication and managed identity;
- produce deterministic and explainable output.

It must not:

- add `azd spring` or other Spring-specific commands;
- copy Spring detection into azd core;
- require Spring Cloud Azure;
- target Azure Spring Apps;
- hide missing core capabilities behind extension-specific workarounds.

## Verified extension capabilities

The current framework provides:

| Capability | Intended Spring Boot use |
| --- | --- |
| `FrameworkServiceProvider` | Maven restore, build, and executable JAR packaging |
| `ServiceTargetProvider` | Image publication, Container Apps deployment, and endpoint discovery |
| `ProvisioningProvider` | Infrastructure preview, deployment, state, outputs, and teardown |
| `ValidationCheckProvider` | Java, Maven, artifact, port, region, and configuration checks |
| Lifecycle event handlers | Standard azd workflow integration |
| Project/service `AdditionalProperties` | Extension-owned Spring metadata |
| Environment service | Read and write azd environment values |
| Beta `ComposeService` | List, inspect, and add composed resources |

Useful local references:

- `cli/azd/docs/extensions/extension-framework.md`
- `cli/azd/docs/extensions/extension-framework-services.md`
- `cli/azd/docs/extensions/extension-sdk-reference.md`
- `cli/azd/docs/extensions/extension-e2e-walkthrough.md`
- `cli/azd/docs/extensions/contract-versioning.md`
- `cli/azd/extensions/extension.schema.json`
- `cli/azd/extensions/microsoft.azd.demo/extension.yaml`
- `cli/azd/extensions/microsoft.azd.demo/internal/cmd/listen.go`
- `cli/azd/extensions/microsoft.azd.demo/internal/project/framework_service_demo.go`
- `cli/azd/extensions/microsoft.azd.demo/internal/project/service_target_demo.go`
- `cli/azd/extensions/microsoft.azd.demo/internal/project/provisioning_demo.go`
- `cli/azd/extensions/microsoft.azd.demo/internal/project/demo_validation_check.go`

## Open core dependencies

All of these issues were open during the investigation:

| Issue | Required behavior |
| --- | --- |
| [Azure/azure-dev#9837](https://github.com/Azure/azure-dev/issues/9837) | Let an extension provide init defaults such as environment, subscription, and region |
| [Azure/azure-dev#9838](https://github.com/Azure/azure-dev/issues/9838) | Let extensions contribute directory detectors |
| [Azure/azure-dev#9839](https://github.com/Azure/azure-dev/issues/9839) | Reference and translate an external project from `azure.yaml` |
| [Azure/azure-dev#9840](https://github.com/Azure/azure-dev/issues/9840) | Infer and materialize a project when `azd up` starts without `azure.yaml` |
| [Azure/azure-dev#9771](https://github.com/Azure/azure-dev/issues/9771) | Remap component and layer inputs and outputs without collisions |

`ProjectService.AddService` works after an azd project exists. It is not a
directory-detection or external-project translation hook.

## Important findings

1. **Build and deployment can be prototyped now.**
   A manually authored `azure.yaml` can select extension-provided framework,
   provisioning, and service-target providers.

2. **The headline experience cannot be implemented only with upstream documented APIs.**
   The custom prototype azd binary adds pre-project extension startup and
   init-provider dispatch for bare `azd up`; upstream integration is still
   required.

3. **Configuration mapping needs its own spike.**
   The environment service stores values, but this does not prove that
   `./mvnw spring-boot:run`, IntelliJ, and VS Code automatically receive them.
   Determine the supported local-development mechanism instead of assuming
   azd environment values become process environment variables.

4. **Determinism is a product requirement.**
   Service names, module ordering, inferred resources, configuration mappings,
   and materialized files must remain stable across machines and repeated runs.
   The inferred extension version should be pinned.

5. **Use source contracts as authoritative.**
   Some prose examples show provider signatures that differ from the current
   SDK reference. The extension schema and demo provisioning-provider manifest
   also appear imperfectly aligned. Compile against current Go interfaces and
   protobuf contracts.

6. **Contract maturity matters.**
   Stable APIs use `azd.extensions.v1`; Compose remains under
   `azd.extensions.v1beta`. Do not assume beta fields are functional without a
   corresponding beta host override.

## Recommended next investigation

### 1. Trace provider selection

Determine whether Spring should register:

- a new framework key such as `spring`, or
- an override for the built-in `java` framework.

Answer these questions from code and tests:

- How is a framework provider selected from `ServiceConfig.language`?
- What happens when core and an extension register the same language?
- Can a provider be selected without manually writing `language` in
  `azure.yaml`?
- Does the framework artifact model preserve everything the service target
  needs, including the executable JAR path and Java version?

Start in:

- `cli/azd/pkg/project/`
- `cli/azd/pkg/azdext/`
- `cli/azd/grpc/proto/azd/extensions/v1/framework_service.proto`

### 2. Trace project creation before `azure.yaml`

Follow `azd init --from-code` and `azd up` when no project exists:

- locate the current `appdetect` call path;
- identify where project-not-found is returned;
- inspect the proposed integration points in #9837, #9838, #9839, and #9840;
- document the minimum contract needed by Spring detection and translation.

The result should distinguish:

- capabilities already merged but not documented;
- capabilities still only proposed in issues;
- Spring-specific logic that belongs entirely in the extension.

### 3. Define the detected Spring project model

Use a multi-module Maven fixture and define deterministic output for:

- deployable modules;
- module paths and stable service names;
- Java and Spring Boot versions;
- packaging and executable artifact paths;
- application and management ports;
- Actuator health endpoints;
- dependencies that imply PostgreSQL, Service Bus, Storage, and Key Vault;
- provenance for every inference;
- unknown or conflicting values.

Prefer effective Maven metadata over matching one parent-POM shape. Detection
must not execute application code or modify the repository.

### 4. Define configuration mappings

Create a table for the initial integrations:

| Integration | Detection evidence | Provisioned outputs | Spring Boot/Spring Cloud Azure properties | Required role |
| --- | --- | --- | --- | --- |
| PostgreSQL | TBD | TBD | TBD | TBD |
| Service Bus | TBD | TBD | TBD | TBD |
| Storage | TBD | TBD | TBD | TBD |
| Key Vault | TBD | TBD | TBD | TBD |

Verify mappings against current Spring Boot relaxed binding and Spring Cloud
Azure property names. Prefer endpoints, namespaces, account names, and database
names over secrets. Local code should use `DefaultAzureCredential`; deployed
code should use managed identity through the same application path.

### 5. Build the first vertical slice

After the contract questions are answered, create a bounded prototype:

```text
explicit azure.yaml
  -> extension framework provider builds one Maven executable JAR
  -> provisioning provider creates minimal Container Apps infrastructure
  -> service target publishes and deploys the image
  -> validation checks Java, Maven, artifact, and port configuration
  -> endpoint appears through standard azd output
```

Use a multi-module fixture with one Spring Boot web application, Actuator,
PostgreSQL, and Service Bus dependencies. The prototype should explain every
inference and produce identical ordered output on repeated runs.

## Suggested issue order

Start with the detection and translation chain:

1. Azure/azure-dev#9838
2. Azure/azure-dev#9965
3. Azure/azure-dev#9839
4. Azure/azure-dev#9966
5. Azure/azure-dev#9840
6. Azure/azure-dev#9978

Then investigate configuration and deployment:

1. Azure/azure-dev#9771
2. Azure/azure-dev#9964
3. Azure/azure-dev#9963
4. Azure/azure-dev#9979
5. Azure/azure-dev#9968

## Investigation checklist

- Re-read issue bodies because the extension contracts are actively evolving.
- Check issue and PR links for implementations merged after this investigation.
- Inspect actual interfaces and protobufs before proposing an architecture.
- Keep findings in `SPRING-BOOT-AZD-INVESTIGATION.md`.
- Do not broaden implementation until provider selection and pre-project
  detection contracts are understood.
- Do not create Spring-specific commands as a workaround.

## Related Java runtime proof of concept

Repositories:

- Upstream: https://github.com/JonathanGiles/java-runtime-for-azure
- Fork branch:
  https://github.com/saragluna/java-runtime-for-azure/tree/poc/todo-java-mongo-aca

The fork branch is 61 commits ahead of upstream `main` and 0 behind. It changes
119 files, with approximately 30,029 additions and 84 deletions. The branch
head is
[`99adca9`](https://github.com/saragluna/java-runtime-for-azure/commit/99adca9c9049f38929d42d0cc8ff332e1d6344b5)
from August 9, 2024.

Despite its name, the branch contains no Todo, MongoDB, or Azure Container Apps
artifacts. It is primarily a proof of concept for running the Java runtime
locally through Aspire's Developer Control Plane (DCP).

Key changes:

- Adds an `azure-runtime-dcp` Java 21 module that starts DCP, translates
  application resources into Kubernetes-style containers, executables,
  services, endpoints, and networks, and manages logs and shutdown.
- Adds local processing to the core runtime, including template generation,
  project introspection, and an `aspire-apps.json` manifest.
- Adds generated Spring Cloud Config Server and Zipkin Server resources.
- Detects Eureka and Zipkin Maven dependencies and injects corresponding
  runtime configuration.
- Updates the Storage Explorer sample to use `DcpAppHost` and Spring Boot
  build-image support.

Treat this branch as exploratory prior art rather than a reusable implementation.
It includes generated sources, a bundled macOS ARM64 binary, hard-coded example
paths, an infinite host loop, duplicate Maven configuration, and no GitHub
check runs for the branch head.

### Implementation relevance

The upstream repository is an MIT-licensed Java proof of concept based on an
explicitly authored App Host. It does not analyze an arbitrary Spring
repository or generate an azd project. Its useful patterns are:

- a deployment-neutral graph of applications, resources, and dependencies;
- deterministic translation from that graph into an intermediate manifest;
- dependency-driven supporting resources and runtime configuration; and
- explicit Spring project, endpoint, and container abstractions.

Relevant upstream areas include:

- `azure-runtime/azure-runtime/src/main/java/com/azure/runtime/host/`
- `azure-runtime/azure-runtime/src/main/java/com/azure/runtime/resources/`
- `azure-runtime/extensions/spring/`
- `azure-runtime/extensions/microservice-common/`

The fork primarily adds local DCP orchestration through
`azure-runtime/azure-runtime-dcp/`. Its Kubernetes-style containers,
executables, services, networks, gRPC protocol, and process lifecycle are not
applicable to the initial azd extension. The generated Eureka, Spring Cloud
Config, and Zipkin resources are examples of dependency-triggered
configuration, but should not be ported into the current Azure deployment
scope.

The existing `azure.springboot` prototype already implements the most relevant
architectural idea through its deployment-neutral `Analysis`, `Service`,
`Resource`, `ServiceDependency`, and `Evidence` types. Direct reuse from the
Java runtime repositories would introduce unrelated Java 21, Kubernetes,
gRPC, protobuf, and DCP runtime dependencies.

Neither repository solves the extension's main implementation requirements:

- effective Maven and multi-module analysis;
- deployable-module and executable-artifact selection;
- Spring properties, YAML, profile, and port resolution;
- Azure resource and role inference;
- `azure.yaml`, Dockerfile, and Bicep generation; or
- integration with the azd init-provider contract.

### Resulting implementation priorities

1. Keep and expand the prototype's deployment-neutral analysis graph.
2. Replace shallow raw-POM interpretation with effective Maven metadata.
3. Model packaging, executable artifacts, Java and Spring versions, confidence,
   conflicts, and structured evidence.
4. Parse Spring properties and YAML, including profiles and placeholders.
5. Make Actuator an optional health-check enhancement, not a Spring detection
   requirement.
6. Add conservative Service Bus, Storage, Key Vault, and additional database
   inference.
7. Add deterministic multi-module tests for deployable modules, artifacts,
   ports, configuration conflicts, and generated bindings.

Use the Java runtime repositories as design validation for the resource-graph
approach, not as an implementation dependency or a source for broad code
transplantation.
