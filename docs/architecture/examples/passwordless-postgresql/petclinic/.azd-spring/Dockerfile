FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /workspace
COPY . .
RUN chmod +x mvnw && ./mvnw -B -DskipTests package

FROM eclipse-temurin:17-jre
WORKDIR /app
COPY --from=build --chown=10001:10001 /workspace/target/spring-petclinic-4.0.0-SNAPSHOT.jar /app/application.jar
COPY --chown=10001:10001 .azd-spring/application-azd.properties /app/config/application-azd.properties
EXPOSE 8080
USER 10001
ENTRYPOINT ["java", "-jar", "/app/application.jar"]
