targetScope = 'subscription'

param environmentName string
param location string
param administratorObjectId string
param administratorName string
param bootstrapIp string
param provisionApp bool = false
param runtimeConfigured bool = false
param appImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'

resource group 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: 'rg-${environmentName}'
  location: location
  tags: {
    'azd-env-name': environmentName
    'purpose': 'disposable-postgres-mi-proposal'
  }
}

module database 'database.bicep' = {
  scope: group
  name: 'database'
  params: {
    location: location
    administratorObjectId: administratorObjectId
    administratorName: administratorName
    bootstrapIp: bootstrapIp
  }
}

module application 'application.bicep' = if (provisionApp) {
  scope: group
  name: 'application'
  params: {
    location: location
    postgresHost: database.outputs.host
    runtimeConfigured: runtimeConfigured
    appImage: appImage
  }
}

output AZURE_RESOURCE_GROUP string = group.name
output POSTGRES_HOST string = database.outputs.host
output POSTGRES_SERVER_NAME string = database.outputs.serverName
output POSTGRES_ADMIN_NAME string = administratorName
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = provisionApp ? application!.outputs.registryEndpoint : ''
output AZURE_RESOURCE_PETCLINIC_ID string = provisionApp ? application!.outputs.appId : ''
output PETCLINIC_PRINCIPAL_ID string = provisionApp ? application!.outputs.principalId : ''
output PETCLINIC_ENDPOINT string = provisionApp ? application!.outputs.endpoint : ''
