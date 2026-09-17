# Spring Todo

Minimal Spring Boot application used to validate automatic `azure.springboot`
detection and deployment.

The application intentionally contains no `azure.yaml`, Dockerfile,
infrastructure, or Azure-specific dependency. The extension must infer the
application model from the Maven and Spring Boot configuration.

Run locally:

```shell
./mvnw spring-boot:run
```

Create a todo:

```shell
curl -X POST http://localhost:8080/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Try the Spring Boot extension"}'
```
