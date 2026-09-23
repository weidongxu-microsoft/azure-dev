param environmentName string
param location string = resourceGroup().location
param identityPrincipalId string

var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))
var modelDeploymentName = 'gpt-4.1-mini'
var cognitiveServicesUserRoleId = subscriptionResourceId(
  'Microsoft.Authorization/roleDefinitions',
  'a97b65f3-24c7-4388-baec-2e87135dc908'
)

resource foundry 'Microsoft.CognitiveServices/accounts@2025-06-01' = {
  name: 'ai${resourceToken}'
  location: location
  tags: {
    'azd-env-name': environmentName
  }
  kind: 'AIServices'
  sku: {
    name: 'S0'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    allowProjectManagement: true
    customSubDomainName: 'ai${resourceToken}'
    disableLocalAuth: true
    publicNetworkAccess: 'Enabled'
  }

  resource modelDeployment 'deployments' = {
    name: modelDeploymentName
    properties: {
      model: {
        format: 'OpenAI'
        name: 'gpt-4.1-mini'
        version: '2025-04-14'
      }
    }
    sku: {
      name: 'GlobalStandard'
      capacity: 1
    }
  }
}

resource applicationModelUser 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(foundry.id, identityPrincipalId, cognitiveServicesUserRoleId)
  scope: foundry
  properties: {
    principalId: identityPrincipalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: cognitiveServicesUserRoleId
  }
}

output endpoint string = 'https://${foundry.name}.openai.azure.com/openai/v1/'
output modelDeploymentName string = foundry::modelDeployment.name
