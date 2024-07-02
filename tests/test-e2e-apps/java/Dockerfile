FROM --platform=$BUILDPLATFORM eclipse-temurin:17-jdk-focal as builder

RUN apt update
RUN apt install zip unzip -y

RUN mkdir /app

# In some archs, there could be probems setting the executable permissions
RUN mkdir /app/.gradle
RUN chmod -R +x /root/

RUN curl -s "https://get.sdkman.io" | bash
RUN bash -c "source $HOME/.sdkman/bin/sdkman-init.sh && sdk install springboot && spring init /app"

WORKDIR /app
COPY DemoApplication.java /app/src/main/java/com/example/app/
COPY build.gradle .
RUN ./gradlew bootJar --no-daemon

FROM eclipse-temurin:17.0.8.1_1-jre

COPY --from=builder /app/build/libs/app-0.0.1-SNAPSHOT.jar .
ENTRYPOINT ["java", "-jar", "app-0.0.1-SNAPSHOT.jar"]
