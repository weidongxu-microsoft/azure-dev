param environmentName string
param location string = resourceGroup().location
param identityPrincipalId string

@secure()
param administratorPassword string = newGuid()

var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))
var databaseName = 'todos'
var administratorLogin = 'azdadmin'

resource keyVault 'Microsoft.KeyVault/vaults@2024-11-01' = {
  name: 'kv${resourceToken}'
  location: location
  tags: {
    'azd-env-name': environmentName
  }
  properties: {
    tenantId: tenant().tenantId
    sku: {
      family: 'A'
      name: 'standard'
    }
    accessPolicies: [
      {
        tenantId: tenant().tenantId
        objectId: identityPrincipalId
        permissions: {
          secrets: [
            'get'
          ]
        }
      }
    ]
    enableRbacAuthorization: false
    enableSoftDelete: true
    softDeleteRetentionInDays: 7
    publicNetworkAccess: 'Enabled'
  }
}

resource passwordSecret 'Microsoft.KeyVault/vaults/secrets@2024-11-01' = {
  parent: keyVault
  name: 'postgres-administrator-password'
  properties: {
    value: administratorPassword
  }
}

resource server 'Microsoft.DBforPostgreSQL/flexibleServers@2024-08-01' = {
  name: 'psql-${resourceToken}'
  location: location
  tags: {
    'azd-env-name': environmentName
  }
  sku: {
    name: 'Standard_B1ms'
    tier: 'Burstable'
  }
  properties: {
    administratorLogin: administratorLogin
    administratorLoginPassword: administratorPassword
    authConfig: {
      activeDirectoryAuth: 'Disabled'
      passwordAuth: 'Enabled'
    }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
    network: {
      publicNetworkAccess: 'Enabled'
    }
    storage: {
      autoGrow: 'Enabled'
      storageSizeGB: 32
    }
    version: '16'
  }
}

resource database 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2024-08-01' = {
  parent: server
  name: databaseName
  properties: {}
}

resource allowAzureServices 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2024-08-01' = {
  parent: server
  name: 'AllowAllAzureServicesAndResourcesWithinAzureIps'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

output jdbcUrl string = 'jdbc:postgresql://${server.properties.fullyQualifiedDomainName}:5432/${database.name}?sslmode=require'
output administratorLogin string = administratorLogin
output credentialReference string = passwordSecret.properties.secretUriWithVersion
