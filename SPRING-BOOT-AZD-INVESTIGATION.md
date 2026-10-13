# Spring Boot and Azure Developer CLI investigation

> Current direction, October 13, 2026: generate deployment files explicitly
> through `azd spring init`, then use ordinary `azd up`. No prototype core
> patches remain. Start with the
> [implementation handoff](./SPRING-BOOT-AZD-EXTENSION-IMPLEMENTATION-INVESTIGATION.md)
> and [current prototype/test matrix](./cli/azd/extensions/azure.springboot/README.md).

## Goal

[Azure/azure-dev#9960](https://github.com/Azure/azure-dev/issues/9960) proposes an
`azure.springboot` extension that can take an existing Spring Boot application
from source to Azure Container Apps. The first milestone is `azd spring init`
generating `azure.yaml` and infrastructure, followed by standard `azd up`.
Spring remains the application source of truth; generated deployment files
are reviewable artifacts, not a replacement for its build/configuration.

## Existing extension capabilities

The current extension framework already supports much of the implementation:

| Requirement | Extension capability |
| --- | --- |
| Maven restore, build, and package | `FrameworkServiceProvider` |
| Image publication and Container Apps deployment | `ServiceTargetProvider` |
| Infrastructure preview, deployment, state, and teardown | `ProvisioningProvider` |
| Pre-provision compatibility checks | `ValidationCheckProvider` |
| Project and service workflow integration | Lifecycle event handlers |
| Spring-owned metadata | Project and service `AdditionalProperties` |
| azd environment values | Environment service |

These capabilities are sufficient for a vertical slice that starts with an
explicit `azure.yaml`: build an executable JAR, provision Container Apps
infrastructure, deploy it, validate prerequisites, and return its endpoint.

The current prototype needs only the custom-command capability for init;
ordinary azd handles the generated Java/Container Apps project afterward.
YAML generation reuses public `project.ProjectConfig`/`project.Save`; Bicep
uses extension-owned templates. Future provider capabilities below must not
be mistaken for required registrations in the current implementation.

Relevant documentation:

- [Extension framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extensions/extension-framework.md)
- [Framework service providers](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extensions/extension-framework-services.md)
- [Extension SDK reference](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extensions/extension-sdk-reference.md)
- [End-to-end extension walkthrough](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extensions/extension-e2e-walkthrough.md)

## Historical core dependencies for automatic initialization

The earlier zero-file/bare-`up` investigation depended on shared upstream
contracts and a custom azd build. Those prototype changes have been removed.
The tracked capabilities below concern possible future standard-init or
automatic-up integration; they do not block explicit `azd spring init`.

| Required behavior | Dependency |
| --- | --- |
| Let extensions detect a Spring Boot directory | [Azure/azure-dev#9838](https://github.com/Azure/azure-dev/issues/9838) |
| Translate Maven modules into the azd project model | [Azure/azure-dev#9839](https://github.com/Azure/azure-dev/issues/9839) |
| Run `azd up` without an existing `azure.yaml` | [Azure/azure-dev#9840](https://github.com/Azure/azure-dev/issues/9840) |
| Supply environment, subscription, and region defaults | [Azure/azure-dev#9837](https://github.com/Azure/azure-dev/issues/9837) |
| Safely compose and remap resource inputs and outputs | [Azure/azure-dev#9771](https://github.com/Azure/azure-dev/issues/9771) |

The documented `ProjectService.AddService` method operates after an azd project
exists. It is not a directory detector or an external-project translation
entry point. The beta `ComposeService` can manipulate composed resources, but
does not yet resolve the input/output and determinism requirements on its own.

## Recommended investigation order

1. **Maven detection**
   - Resolve the effective Maven model.
   - Identify deployable Spring Boot modules without treating parent, library,
     test, or build-only modules as services.
   - Detect Java and Spring Boot versions, packaging, executable artifacts,
     ports, Actuator endpoints, and relevant dependencies.
   - Preserve the source of every inference.

2. **Deterministic project translation**
   - Define stable service names and ordering.
   - Store Spring-specific metadata in extension-owned configuration.
   - Make explicit user configuration override inferred values.
   - Materialize and pin a reviewable `azure.yaml` after initial inference.

3. **Spring configuration and identity mapping**
   - Define dependency-to-property mappings for Spring Boot and Spring Cloud
     Azure.
   - Use Spring relaxed-binding environment-variable names.
   - Use `DefaultAzureCredential` locally and managed identity after deployment.
   - Prefer endpoints and resource names over keys and connection strings.
   - Define least-privilege roles for each supported Azure integration.

4. **Container Apps deployment**
   - Validate the buildpack image path and supported Java versions.
   - Handle multi-module artifact selection.
   - Infer ingress, ports, health probes, JVM memory settings, and scaling.
   - Return the deployed endpoint through the normal azd service experience.

5. **Validation and diagnostics**
   - Separate blocking errors from warnings.
   - Detect ambiguous modules, unsupported Java versions, missing wrappers,
     invalid artifacts, configuration conflicts, unavailable regions, and
     missing roles before deployment.
   - Keep customer content and secrets out of telemetry.

## Historical first-slice proposal

Use a multi-module Maven repository containing one Spring Boot web service with
Actuator, PostgreSQL, and Service Bus dependencies. Begin with an explicit
`azure.yaml` and validate this pipeline:

```text
Maven project
  -> FrameworkServiceProvider builds the executable JAR
  -> ProvisioningProvider creates minimal Container Apps infrastructure
  -> ServiceTargetProvider publishes and deploys the image
  -> ValidationCheckProvider validates Java, artifact, and port configuration
  -> Endpoints returns the application URL
```

The spike should produce deterministic output and explain every inference. In
parallel, the azd team should settle the detection, external-project, and
project-inference contracts because those contracts determine whether the
extension can deliver the broader automatic-init promise of Azure/azure-dev#9960.
The explicit-init implementation does not wait for those contracts.

## Risks observed in the current documentation

- The framework overview and SDK reference show slightly different provider
  signatures.
- The extension schema and demo provisioning-provider manifest are not fully
  aligned.
- Compose, Copilot, and telemetry services remain on the `v1beta` contract.

Implementations should compile against the current Go interfaces and protobuf
contracts rather than relying only on prose examples.
