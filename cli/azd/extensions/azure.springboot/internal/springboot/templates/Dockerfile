FROM maven:3.9.11-eclipse-temurin-21 AS build
WORKDIR /app

COPY . .
RUN mvn --batch-mode --no-transfer-progress -pl {{MAVEN_MODULE}} -am package -DskipTests

FROM eclipse-temurin:21-jre
WORKDIR /app

COPY --from=build {{ARTIFACT_DIR}}/target/*.jar app.jar
EXPOSE {{TARGET_PORT}}

ENTRYPOINT ["java", "-jar", "app.jar"]
