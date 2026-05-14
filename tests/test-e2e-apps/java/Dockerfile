FROM --platform=$BUILDPLATFORM eclipse-temurin:17-jdk-focal AS builder

RUN apt update
RUN apt install zip unzip -y

RUN mkdir /app

# In some archs, there could be probems setting the executable permissions
RUN mkdir /app/.gradle
RUN chmod -R +x /root/

RUN curl -s "https://get.sdkman.io" | bash
RUN bash -c "source $HOME/.sdkman/bin/sdkman-init.sh && sdk install springboot && spring init /app"

WORKDIR /app
# Pin Gradle wrapper — `spring init` scaffolds a wrapper at whatever version Spring CLI currently
# ships, which can be incompatible with our pinned spring-boot plugin in build.gradle.
RUN sed -i 's|gradle-[0-9.]\+-bin.zip|gradle-8.10-bin.zip|' /app/gradle/wrapper/gradle-wrapper.properties
COPY DemoApplication.java /app/src/main/java/com/example/app/
COPY build.gradle .
# Remove the Application.java scaffolded by `spring init` so the plugin doesn't see two main classes.
RUN find /app/src/main/java -name 'Application.java' -delete
RUN ./gradlew bootJar --no-daemon

FROM docker.io/library/eclipse-temurin:17.0.8.1_1-jre

COPY --from=builder /app/build/libs/app-0.0.1-SNAPSHOT.jar .
ENTRYPOINT ["java", "-jar", "app-0.0.1-SNAPSHOT.jar"]
