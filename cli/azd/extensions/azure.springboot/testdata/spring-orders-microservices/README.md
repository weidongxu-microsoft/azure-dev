# Spring Orders Microservices

Multi-module Spring Boot fixture for testing service discovery and generation.

- `order-contracts` is a shared library.
- `catalog-service` is an internal HTTP service.
- `orders-service` is a public HTTP service that uses PostgreSQL and calls the
  catalog service.
