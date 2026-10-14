param location string
param environmentId string
param registryServer string
param privateRuntime bool

var probe = '''
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({"probe": "aca-bicep-system-identity", "port": 8080}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
HTTPServer(("0.0.0.0", 8080), Handler).serve_forever()
'''

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
        targetPort: privateRuntime ? 8080 : 80
        transport: 'auto'
      }
      registries: [{
        server: registryServer
        identity: 'system'
      }]
    }
    template: {
      containers: [{
        name: 'main'
        image: privateRuntime ? '${registryServer}/identity-probe:v1' : 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
        command: privateRuntime ? ['python', '-u', '-c', probe] : []
        resources: { cpu: json('0.25'), memory: '0.5Gi' }
      }]
      scale: { minReplicas: 1, maxReplicas: 1 }
    }
  }
}

output appId string = app.id
output endpoint string = 'https://${app.properties.configuration.ingress.fqdn}'
