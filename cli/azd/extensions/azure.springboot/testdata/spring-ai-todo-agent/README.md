# Spring AI Todo Agent

This minimal Spring Boot application exposes `POST /api/chat` and lets a Spring AI model call in-memory tools to list,
create, and complete todo items.

## Run locally

Sign in with Azure CLI, then set a Microsoft Foundry Models inference endpoint
and deployment name:

```powershell
$null = az login
$env:AZURE_INFERENCE_ENDPOINT = "https://<resource>.openai.azure.com/openai/v1"
$env:AZURE_INFERENCE_MODEL = "<deployment>"
.\mvnw.cmd spring-boot:run
```

The application uses `DefaultAzureCredential`. In Azure Container Apps, it uses
the generated user-assigned managed identity.

Then send a request:

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/chat `
    -ContentType application/json `
    -Body '{"message":"Create a todo to test azd, then list my todos."}'
```

The source fixture intentionally contains no `azure.yaml`, Dockerfile, or infrastructure files. It is used to compare
core `azd init --from-code` with Spring-specific initialization.
