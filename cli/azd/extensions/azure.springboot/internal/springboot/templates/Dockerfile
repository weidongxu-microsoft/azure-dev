FROM maven:3.9.11-eclipse-temurin-21 AS build
WORKDIR /app

COPY pom.xml .
COPY src ./src
RUN mvn --batch-mode --no-transfer-progress package -DskipTests

FROM eclipse-temurin:21-jre
WORKDIR /app

COPY --from=build /app/target/*.jar app.jar
EXPOSE {{TARGET_PORT}}

ENTRYPOINT ["java", "-jar", "app.jar"]
