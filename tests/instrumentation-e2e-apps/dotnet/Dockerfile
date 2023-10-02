FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/sdk:7.0.400-bookworm-slim-amd64 AS build
ARG TARGETARCH
WORKDIR /source

RUN dotnet new webapp -o MyWebApp --no-https -f net7.0

WORKDIR /source/MyWebApp
RUN dotnet publish -a $TARGETARCH -o /app

FROM mcr.microsoft.com/dotnet/aspnet:7.0.10-bullseye-slim

ENV DOTNET_ROLL_FORWARD=Major
ENV DOTNET_ROLL_FORWARD_PRE_RELEASE=1

WORKDIR /app
COPY --from=build /app/ .

ENTRYPOINT ["./MyWebApp"]