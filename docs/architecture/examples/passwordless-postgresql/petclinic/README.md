# Petclinic: passwordless PostgreSQL

Disposable proof-of-capability configuration, not a production template.

## Files

| File | Purpose |
| --- | --- |
| `azure.yaml` | Docker/ACA service, explicit Bicep infrastructure, postprovision bootstrap. |
| `infra/main.bicep`, `main.parameters.json` | Resource group, database/application phases, environment substitutions. |
| `infra/database.bicep` | PostgreSQL 15, Entra administrator, database, exact bootstrap-IP firewall rule; password authentication disabled. |
| `infra/application.bicep` | ACA environment, Log Analytics, ACR, system-assigned ACA, registry-scoped `AcrPull`. |
| `scripts/db.py`, `requirements.txt` | Administrator-authenticated SQL principal mapping, schema/seed initialization, grants, restricted ACA firewall rules, staged registry/ingress configuration. |
| `.azd-spring/application-azd.properties` | JDBC managed-identity plugin, hostname-verified TLS, no runtime DDL. |
| `.azd-spring/Dockerfile`, `.dockerignore` | Java 17, non-root application image and isolated build context. |
| `dependency.xml` | Runtime Maven dependency to add to the application's existing `<dependencies>`. |

## Prepare the application

Use PowerShell, Azure CLI/azd signed in, Docker, and Python. The deployment identity
must provision resources/role assignments and configure a PostgreSQL Entra
administrator. Run bootstrap as that administrator.

Clone https://github.com/spring-projects/spring-petclinic into a **new directory**
outside this configuration checkout and detach at
`500158f732419217507c7656904b8e6aa1bcc0d6`.
Copy `azure.yaml`, `.dockerignore`, `infra`, `scripts`, and `.azd-spring` from this
folder into the application directory. Add the dependency in `dependency.xml`
to the existing POM. No application-source changes are required.

The pinned application uses Spring Boot 4.1.0. The live image included Azure
Identity Extensions 1.2.10, Azure Identity 1.18.7, and PostgreSQL JDBC 42.7.11.
The Dockerfile's JAR filename matches this pinned application revision.

From the application directory:

```powershell
python -m pip install -r scripts\requirements.txt
az account set --subscription "<subscription-name-or-id>"
azd env new "<unique-environment-name>" --subscription "<subscription-id>" --location westus3
azd env set POSTGRES_ADMIN_OID "<administrator-user-object-id>"
azd env set POSTGRES_ADMIN_NAME "<administrator-user-principal-name>"
azd env set BOOTSTRAP_IP "<your-public-IPv4-address>"
azd env set PROVISION_APP false
```

Do not substitute another tenant's object ID. The administrator must belong to
the configured PostgreSQL authentication tenant and be able to acquire the
`oss-rdbms` token used by the script. Supply your actual values locally;
do not commit generated `.azure` files.

## Provision and deploy

1. Build before incurring Azure costs:
   `docker build --platform linux/amd64 -f .azd-spring/Dockerfile -t petclinic-postgres-mi:proposal .`
2. Run `azd provision --no-prompt`. The database-only postprovision hook verifies
   Entra administrator connectivity before creating ACA.
3. Run `azd env set PROVISION_APP true`, then `azd provision --no-prompt`.
   ACA initially uses a public placeholder, no registry configuration, and port 80.
   Bicep grants its system identity `AcrPull`. The hook adds exact outbound-IP
   firewall rules, maps the PostgreSQL role, initializes the application schema,
   grants permissions, then configures registry access and port 8080.
4. Run `azd deploy petclinic --from-package petclinic-postgres-mi:proposal --no-prompt`.
   The private application image uses the passwordless JDBC configuration.
5. Persist the deployed image for later provisioning:

```powershell
$values = azd env get-values --output json | ConvertFrom-Json
$image = az containerapp show --ids $values.AZURE_RESOURCE_PETCLINIC_ID --query 'properties.template.containers[0].image' -o tsv
azd env set PETCLINIC_IMAGE $image
```

The hook sets `APP_RUNTIME_READY=true`; later Bicep provisioning therefore retains
the final registry configuration and port. Preserving `PETCLINIC_IMAGE` separately
prevents a later provision from selecting the public placeholder image.
If bootstrap fails, inspect and correct the error before explicitly resuming it
with `azd hooks run postprovision --no-prompt`.

**Destructive bootstrap:** Petclinic's own `schema.sql`/`data.sql` initialize/reset
the demo schema. Do not rerun this hook against data you want to preserve.
This prototype does not provide production migrations or safe schema reconciliation.

## Permissions and boundaries

The runtime role `petclinic_app` receives `CONNECT`, schema `USAGE`, `SELECT` on
Petclinic tables, `INSERT`/`UPDATE` on `owners`/`pets`/`visits`, and sequence `USAGE`.
It does not own the schema and has no DDL, DELETE, reference-data writes, or
administrative permissions. Not every Petclinic workflow is permitted by this policy.

The empty datasource password overrides the application's development fallback;
it is not a provisioned password. The JDBC plugin explicitly selects managed
identity and uses verified TLS. No `AZURE_CLIENT_ID` is needed for a system identity.

The combined hook uses CLI calls for firewall, registry, and ingress changes.
Only principal mapping and SQL grants require SQL data-plane execution. See the
separately tested [Bicep-only ACA example](../aca-bicep-staging) for registry staging.
The two examples were not tested as a combined deployment.

Exact ACA outbound addresses were sufficient for this disposable test, but are not
a stable production egress contract. Private connectivity or fixed NAT requires
separate design. All bootstrap-IP and ACA firewall rules need cleanup.

## Verification and cleanup

The live trial read seeded vets, created and edited an owner through HTML forms,
independently read the owner back, confirmed its row through administrator SQL,
and verified persistence/fresh app connections after restarting the ready revision.
Negative SQL checks denied DDL, DELETE, and reference-data writes.
Token refresh after expiry was not tested.

For a new run, verify these behaviors before considering the configuration valid
for that subscription. Afterward run `azd down --force --purge --no-prompt`, remove
the run-owned local Docker image, and confirm the disposable resource group is gone.
Never delete a shared resource group.
