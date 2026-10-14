# Bicep-only system-identity registry staging

Separately live-validated ACA/ACR example; no PostgreSQL or Petclinic deployment.

| Module | Responsibility |
| --- | --- |
| `main.bicep` | ACR/environment and ordered nested deployments. |
| `bootstrap.bicep` | Create public-image ACA with system identity and no registry configuration; `@onlyIfNotExists()` prevents rollback later. |
| `permissions.bicep` | Grant registry-scoped `AcrPull` to that principal. |
| `configured.bicep` | Update the same ACA with system-identity registry configuration; optionally select the private probe image and port 8080. |

Requires Bicep 0.38.3+ and ARM support for `@onlyIfNotExists()`.
The live test used Bicep 0.45.15 and ACA API `2024-03-01`.

## Reproduce

Use a **new, disposable** resource group in an authorized subscription:

```powershell
az account set --subscription "<subscription-name-or-id>"
$group = "<new-disposable-resource-group>"
az group create --name $group --location westus3
az deployment group create --resource-group $group --name initial --template-file main.bicep
$outputs = az deployment group show --resource-group $group --name initial --query properties.outputs -o json | ConvertFrom-Json
$registry = $outputs.registryName.value
az acr import --name $registry --source docker.io/library/python:3.12-alpine --image identity-probe:v1
az deployment group create --resource-group $group --name private-runtime --template-file main.bicep --parameters privateRuntime=true
Invoke-RestMethod $outputs.endpoint.value
az deployment group create --resource-group $group --name repeat --template-file main.bicep --parameters privateRuntime=true
Invoke-RestMethod $outputs.endpoint.value
```

These commands illustrate the sequence. Check every operation's exit status.
For long-running work, submit deployment/import with `--no-wait` and monitor
completion before advancing; do not submit dependent steps prematurely.
Initial deployment retains the public placeholder while configuring registry
access. Image import is a separate publishing operation, not a Bicep resource.

Both HTTP responses must be:

```json
{"probe":"aca-bicep-system-identity","port":8080}
```

Compare ACA's principal ID, private image, target port, and latest ready revision
before/after the identical redeployment:

```powershell
az containerapp show --resource-group $group --name identity-staging --query '{principalId:identity.principalId,image:properties.template.containers[0].image,port:properties.configuration.ingress.targetPort,revision:properties.latestReadyRevisionName}' -o json
```

The live test returned HTTP 200 on both checks and preserved the same principal
and ready revision. ACR admin credentials were disabled, ACA had no secrets or
user-assigned identities, and registry authentication used `identity: 'system'`.
No imperative ACA registry or ingress updates were used.

No RBAC propagation failure occurred. Dependency ordering alone does not prove
immediate authorization propagation on every deployment.
Do not reset `privateRuntime=false` after deploying the private image.

## Cleanup

After confirming that `$group` is the run-owned disposable group:

```powershell
az group delete --name $group --yes --no-wait
az group exists --name $group
```

Monitor until `group exists` returns `false`; ACA environment deletion can linger.
The original live experiment verified deletion of the complete resource group.
