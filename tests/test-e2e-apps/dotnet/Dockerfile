# Stage 1: Build the application
FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/sdk:10.0-noble@sha256:8c6edfb26c85153f236d26efebd9ad349aec403583f343f201ee67838b23658d AS build
ARG TARGETARCH
WORKDIR /source

# Copy the project file first and restore dependencies
# This improves Docker layer caching
COPY DiceRoller/*.csproj ./DiceRoller/
RUN dotnet restore ./DiceRoller/DiceRoller.csproj --arch $TARGETARCH

# Copy the rest of the application code
COPY DiceRoller/. ./DiceRoller/

# Publish the application
# Update WORKDIR to the project directory before publishing
WORKDIR /source/DiceRoller
# Publish for the target architecture, in Release configuration, to the /app folder
# --no-restore is used because we already restored dependencies
RUN dotnet publish -a $TARGETARCH -c Release -o /app --no-restore

# Stage 2: Create the runtime image
FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/aspnet:10.0-noble@sha256:aac50ceef10391e78a899e3d2ccd00bbafa2dcb56ccf4586cbccf00edb84fc21

# Environment variables for .NET runtime behavior
ENV DOTNET_ROLL_FORWARD=Major
# Consider setting DOTNET_ROLL_FORWARD_PRE_RELEASE to 0 (false) for production
ENV DOTNET_ROLL_FORWARD_PRE_RELEASE=1

# Set the working directory in the runtime image
WORKDIR /app

# Copy the published application artifacts from the build stage
COPY --from=build /app/ .

# Set the entry point to run the application
ENTRYPOINT ["./DiceRoller"]