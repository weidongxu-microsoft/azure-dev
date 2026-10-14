param location string
param postgresHost string
param runtimeConfigured bool = false
param appImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
var token = uniqueString(resourceGroup().id)

resource workspace 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: 'log-mi-${token}'
  location: location
  properties: {
    sku: { name: 'PerGB2018' }
    retentionInDays: 30
  }
}

resource environment 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: 'cae-mi-${token}'
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: workspace.properties.customerId
        sharedKey: workspace.listKeys().primarySharedKey
      }
    }
  }
}

resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: 'acrmi${token}'
  location: location
  sku: { name: 'Basic' }
  properties: {
    adminUserEnabled: false
  }
}

resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'petclinic'
  location: location
  tags: { 'azd-service-name': 'petclinic' }
  identity: { type: 'SystemAssigned' }
  properties: {
    managedEnvironmentId: environment.id
    configuration: {
      activeRevisionsMode: 'Single'
      ingress: {
        external: true
        targetPort: runtimeConfigured ? 8080 : 80
        transport: 'auto'
      }
      registries: runtimeConfigured ? [
        {
          server: registry.properties.loginServer
          identity: 'system'
        }
      ] : []
    }
    template: {
      containers: [
        {
          name: 'main'
          image: appImage
          resources: { cpu: json('0.5'), memory: '1Gi' }
          env: [
            { name: 'POSTGRES_HOST', value: postgresHost }
            { name: 'SPRING_PROFILES_ACTIVE', value: 'postgres' }
            { name: 'SPRING_CONFIG_ADDITIONAL_LOCATION', value: 'file:/app/config/application-azd.properties' }
            { name: 'SERVER_PORT', value: '8080' }
          ]
        }
      ]
      scale: { minReplicas: 1, maxReplicas: 1 }
    }
  }
}

resource pull 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: registry
  name: guid(registry.id, app.id, 'AcrPull')
  properties: {
    principalId: app.identity.principalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7ca-43fe172d538d')
  }
}

output registryEndpoint string = registry.properties.loginServer
output appId string = app.id
output principalId string = app.identity.principalId
output endpoint string = 'https://${app.properties.configuration.ingress.fqdn}'
