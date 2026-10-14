param location string
param environmentId string

@onlyIfNotExists()
resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'identity-staging'
  location: location
  identity: { type: 'SystemAssigned' }
  properties: {
    managedEnvironmentId: environmentId
    configuration: {
      activeRevisionsMode: 'Single'
      ingress: {
        external: true
        targetPort: 80
        transport: 'auto'
      }
      registries: []
    }
    template: {
      containers: [{
        name: 'main'
        image: 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
        resources: { cpu: json('0.25'), memory: '0.5Gi' }
      }]
      scale: { minReplicas: 1, maxReplicas: 1 }
    }
  }
}

output principalId string = app.identity.principalId
