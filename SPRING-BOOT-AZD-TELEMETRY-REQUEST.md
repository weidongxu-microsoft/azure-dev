# Spring Boot azd extension telemetry request

> October 13, 2026 scope: the proposed first milestone is
> `azd spring init` -> generated project -> ordinary `azd up`.
> This remains a request for aggregate evidence, not collected results or
> a request to add telemetry fields. See the
> [implementation handoff](./SPRING-BOOT-AZD-EXTENSION-IMPLEMENTATION-INVESTIGATION.md).

## Goal

Determine whether a meaningful group of Spring Boot developers would use an
azd extension to take an existing application without `azure.yaml`, generate
an Azure environment, and run `azd up`.

Use the latest 12 months of aggregate telemetry. Exclude internal, test, and
automated validation traffic where possible. Do not provide customer names or
raw identifiers.

## Requested results

### 1. Java audience

- Active external Java users, projects, and tenants.
- Java share of total azd usage and its trend.
- Known Spring template usage if Spring cannot be identified directly.

**How this validates the goal:** Establishes whether azd already reaches enough
Java developers for a Spring-specific extension to have a plausible audience.
It does not establish extension demand by itself.

### 2. Existing-application (`azd init --from-code`) demand

- Number of Java users entering the existing-application scan through
  `azd init --from-code`.
- Resulting `init.method`, especially `app` and `project`, compared with
  `template`.

The existing-application process is valid today for supported Java Maven
projects. It can detect Java services and MySQL, PostgreSQL, Redis, or MongoDB,
then generate a Container Apps-oriented azd project.

This is already close to the proposed extension's primary goal: start from an
existing application without `azure.yaml`, infer a deployment model, and make
`azd up` possible. The explicit `spring init` milestone must demonstrate
Spring-aware improvements over this baseline; a new command alone is not
incremental value. Existing `--from-code` usage is a proxy for that audience,
not a measurement of the unreleased Spring command.

`init.method=project` should not be treated as an independent user-selected
workflow. It can be recorded when existing-application scanning finds no
supported service or database and falls back to a minimal `azure.yaml`.

**How this validates the goal:** The proposed extension targets existing Spring
applications without `azure.yaml`. These numbers measure use of the closest
existing workflow and establish the baseline that the extension must improve.
The relevant incremental value is better Spring module classification, ports,
service relationships, configuration bindings, and resource inference.

The count is likely to be small and must not be interpreted as total demand.
Users may not know the workflow exists, may start from templates, or may avoid
azd because their project shape is unsupported. A low count therefore does not
invalidate the extension hypothesis. A meaningful count provides stronger
evidence because users are already seeking this experience.

### 3. Existing-application (`azd init --from-code`) conversion

For the Java users and projects counted in item 2, measure progression through:

```text
init attempted
    -> init completed
    -> azd up attempted
    -> azd up succeeded
```

Include:

- primary abandonment steps and errors;
- later `azd deploy` usage; and
- later `azd down` usage, to identify disposable trials.

**How this validates the goal:** Shows whether existing Java applications
actually reach a working Azure deployment. Abandonment is relevant only when it
is caused by problems the extension could address, such as unsupported project
layouts, missing service detection, resource configuration, or generated
deployment setup. Authentication, permission, quota, and transient Azure
failures do not validate the extension hypothesis and should be reported
separately.

Successful `up` followed by later `deploy` indicates continued development.
Successful `up` followed by `down` can validate the disposable trial scenario.
If existing Java applications already convert successfully without
Spring-specific help, the extension has less incremental value.

### 4. Required extension scope

For Java application initialization:

- most frequently detected and confirmed databases and services;
- resources users add or remove after detection; and
- deployment targets, especially Container Apps, App Service, Azure Spring
  Apps, and AKS.

This should indicate which hosting target and resources, such as PostgreSQL,
MySQL, or Redis, matter for an initial extension.

**How this validates the goal:** Determines whether Java applications share a
small, repeatable resource graph that an extension can support reliably. High
confirmation rates for a few services strengthen the case for generation.
Frequent removal or replacement of detected resources indicates that automatic
inference would be unsafe. Deployment-target concentration identifies the
smallest useful first version.

### 5. Extension installation friction

For existing project-required extensions:

```text
extension requested
    -> extension installed
    -> first extension command
    -> successful lifecycle command
```

Report abandonment and failure rates at each step.

**How this validates the goal:** The proposed experience depends on users
installing `azure.springboot` before it can analyze an application. Existing
extension funnels provide a baseline for whether this additional acquisition
step is acceptable. High install-to-use conversion supports the model; high
abandonment means extension discovery or bootstrap may need a core azd change
before Spring-specific functionality matters.

## Decision question

> Is there a meaningful external Java audience attempting existing-application
> initialization, and would better Spring detection and resource generation
> materially improve successful `azd up` conversion?

The extension hypothesis is weaker if existing-application attempts are rare,
Java usage is almost entirely template-based, or required-extension
installation causes substantial abandonment.

The opportunity is stronger if Java existing-application attempts are
meaningful but conversion is poor because of module detection, database
inference, or deployment configuration.
