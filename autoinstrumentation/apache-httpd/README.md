# How to build Apache HTTPD auto-instrumentation docker image

To build image for Apache HTTPD auto instrumentation, use the following commands

```
export REPO_NAME="<your-docker-image-repo-name>"
export IMAGE_NAME_PREFIX="autoinstrumentation-apache-httpd"
export IMAGE_VERSION=`cat version.txt`
export IMAGE_NAME=${REPO_NAME}/${IMAGE_NAME_PREFIX}:${IMAGE_VERSION}
docker build --build-arg version=${IMAGE_VERSION} . -t ${IMAGE_NAME} -t ${REPO_NAME}/${IMAGE_NAME_PREFIX}:latest
docker push ${IMAGE_NAME} 
docker push ${REPO_NAME}/${IMAGE_NAME_PREFIX}:latest
```
