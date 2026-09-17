# Spring Todo

Minimal Spring Boot and PostgreSQL application used to validate automatic
`azure.springboot` analysis and deployment.

The application intentionally contains no `azure.yaml`, Dockerfile,
infrastructure, or Azure-specific dependency. The extension must infer the
application model and PostgreSQL requirement from the Maven and Spring Boot
configuration.

Run PostgreSQL locally, then set the standard Spring datasource variables:

```shell
export SPRING_DATASOURCE_URL=jdbc:postgresql://localhost:5432/todos
export SPRING_DATASOURCE_USERNAME=postgres
export SPRING_DATASOURCE_PASSWORD=postgres
./mvnw spring-boot:run
```

Create a todo:

```shell
curl -X POST http://localhost:8080/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Try the Spring Boot extension"}'
```
