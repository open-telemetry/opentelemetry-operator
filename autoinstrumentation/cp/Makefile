VERSION?=latest
REPOSITORY?=ghcr.io/open-telemetry/opentelemetry-operator
GO_BUILD_LDFLAGS ?= '-w -s -extldflags "-static"'

.PHONY := build
build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o ./bin/cp_$(GOOS)_$(GOARCH)$(EXTRA)$(EXTENSION) -ldflags $(GO_BUILD_LDFLAGS)

bin/cp_linux_amd64: main.go go.mod
	GOOS=linux GOARCH=amd64 EXTENSION="" EXTRA="" make build

bin/cp_linux_arm64: main.go go.mod
	GOOS=linux GOARCH=arm64 EXTENSION="" EXTRA="" make build

bin/cp_linux_ppc64le: main.go go.mod
	GOOS=linux GOARCH=ppc64le EXTENSION="" EXTRA="" make build

bin/cp_linux_s390x: main.go go.mod
	GOOS=linux GOARCH=s390x EXTENSION="" EXTRA="" make build

bin/cp_windows_arm64.exe: main.go go.mod
	GOOS=windows GOARCH=arm64 EXTENSION=".exe" EXTRA="" make build

bin/cp_windows_amd64.exe: main.go go.mod
	GOOS=windows GOARCH=amd64 EXTENSION=".exe" EXTRA="" make build

bin/cp_linux_amd64_fips: main.go go.mod
	GOEXPERIMENT=boringcrypto GOOS=linux GOARCH=amd64 EXTENSION="" EXTRA="_fips" make build

bin/cp_linux_arm64_fips: main.go go.mod
	GOEXPERIMENT=boringcrypto GOOS=linux GOARCH=arm64 EXTENSION="" EXTRA="_fips" make build

bin/cp_windows_arm64_fips.exe: main.go go.mod
	GOEXPERIMENT=boringcrypto GOOS=windows GOARCH=arm64 EXTENSION=".exe" EXTRA="_fips" make build

bin/cp_windows_amd64_fips.exe: main.go go.mod
	GOEXPERIMENT=boringcrypto GOOS=windows GOARCH=amd64 EXTENSION=".exe" EXTRA="_fips" make build

## Docker build

.PHONY := docker_linux_amd64
docker_linux_amd64: bin/cp_linux_amd64
	docker buildx build --platform="linux/amd64" --push -t $(REPOSITORY)/cp_linux_amd64:$(VERSION) --build-arg cp=bin/cp_linux_amd64 .

.PHONY := docker_linux_arm64
docker_linux_arm64: bin/cp_linux_arm64
	docker buildx build --platform="linux/arm64" --push -t $(REPOSITORY)/cp_linux_arm64:$(VERSION) --build-arg cp=bin/cp_linux_arm64 .

.PHONY := docker_linux_ppc64le
docker_linux_ppc64le: bin/cp_linux_ppc64le
	docker buildx build --platform="linux/ppc64le" --push -t $(REPOSITORY)/cp_linux_ppc64le:$(VERSION) --build-arg cp=bin/cp_linux_ppc64le .

.PHONY := docker_linux_s390x
docker_linux_s390x: bin/cp_linux_s390x
	docker buildx build --platform="linux/s390x" --push -t $(REPOSITORY)/cp_linux_s390x:$(VERSION) --build-arg cp=bin/cp_linux_s390x .

.PHONY := docker_windows_arm64
docker_windows_arm64: bin/cp_windows_arm64.exe
	docker buildx build --platform="windows/arm64" --push -f Dockerfile.windows -t $(REPOSITORY)/cp_windows_arm64:$(VERSION) --build-arg cp=bin/cp_windows_arm64.exe .

.PHONY := docker_windows_amd64
docker_windows_amd64: bin/cp_windows_amd64.exe
	docker buildx build --platform="windows/amd64" --push -f Dockerfile.windows -t $(REPOSITORY)/cp_windows_amd64:$(VERSION) --build-arg cp=bin/cp_windows_amd64.exe .

.PHONY := docker_linux_amd64_fips
docker_linux_amd64_fips: bin/cp_linux_amd64_fips
	docker buildx build --platform="linux/amd64" --push -t $(REPOSITORY)/cp_linux_amd64_fips:$(VERSION) --build-arg cp=bin/cp_linux_amd64_fips .

.PHONY := docker_linux_arm64_fips
docker_linux_arm64_fips: bin/cp_linux_arm64_fips
	docker buildx build --platform="linux/arm64" --push -t $(REPOSITORY)/cp_linux_arm64_fips:$(VERSION) --build-arg cp=bin/cp_linux_arm64_fips .

.PHONY := docker_windows_amd64_fips
docker_windows_amd64_fips: bin/cp_windows_amd64_fips.exe
	docker buildx build --platform="windows/amd64" --push -f Dockerfile.windows -t $(REPOSITORY)/cp_windows_amd64_fips:$(VERSION) --build-arg cp=bin/cp_windows_amd64_fips.exe .

.PHONY := docker_windows_arm64_fips
docker_windows_arm64_fips: bin/cp_windows_arm64_fips.exe
	docker buildx build --platform="windows/arm64" --push -f Dockerfile.windows -t $(REPOSITORY)/cp_windows_arm64_fips:$(VERSION) --build-arg cp=bin/cp_windows_arm64_fips.exe .

.PHONY := docker
docker: docker_linux_amd64 docker_linux_arm64 docker_linux_ppc64le docker_linux_s390x docker_windows_amd64 docker_windows_arm64 docker_linux_amd64_fips docker_linux_arm64_fips docker_windows_amd64_fips docker_windows_arm64_fips
	docker buildx imagetools create -t $(REPOSITORY)/cp:$(VERSION) \
		$(REPOSITORY)/cp_linux_amd64:$(VERSION) \
		$(REPOSITORY)/cp_linux_arm64:$(VERSION) \
		$(REPOSITORY)/cp_linux_ppc64le:$(VERSION) \
		$(REPOSITORY)/cp_linux_s390x:$(VERSION) \
		$(REPOSITORY)/cp_windows_amd64:$(VERSION) \
		$(REPOSITORY)/cp_windows_arm64:$(VERSION)

	docker buildx imagetools create \
	    -t $(REPOSITORY)/cp:$(VERSION)-fips \
		$(REPOSITORY)/cp_linux_amd64_fips:$(VERSION) \
		$(REPOSITORY)/cp_linux_arm64_fips:$(VERSION) \
		$(REPOSITORY)/cp_windows_amd64_fips:$(VERSION) \
		$(REPOSITORY)/cp_windows_arm64_fips:$(VERSION)
