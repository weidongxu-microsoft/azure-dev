param location string = 'westus3'
param privateRuntime bool = false

var token = uniqueString(resourceGroup().id)

resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: 'acrstg${token}'
  location: location
  sku: { name: 'Basic' }
  properties: {
    adminUserEnabled: false
  }
}

resource environment 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: 'cae-staging-${token}'
  location: location
  properties: {}
}

module bootstrap 'bootstrap.bicep' = {
  name: '${deployment().name}-bootstrap'
  params: {
    location: location
    environmentId: environment.id
  }
}

module permissions 'permissions.bicep' = {
  name: '${deployment().name}-permissions'
  params: {
    registryName: registry.name
    principalId: bootstrap.outputs.principalId
  }
}

module configured 'configured.bicep' = {
  name: '${deployment().name}-configured'
  params: {
    location: location
    environmentId: environment.id
    registryServer: registry.properties.loginServer
    privateRuntime: privateRuntime
  }
  dependsOn: [permissions]
}

output registryName string = registry.name
output registryServer string = registry.properties.loginServer
output appId string = configured.outputs.appId
output principalId string = bootstrap.outputs.principalId
output endpoint string = configured.outputs.endpoint
output roleAssignmentId string = permissions.outputs.roleAssignmentId
