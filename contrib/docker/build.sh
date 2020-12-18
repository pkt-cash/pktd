#!/usr/bin/env sh

# This script should be run from the project root
# e.g. ./contrib/docker/build.sh

# Install docker:
# $ sudo apt install docker.io)
# $ sudo systemctl start docker
# install dep:
# $ go get github.com/golang/dep 

DOCKER_FILE_LOCATION=./contrib/docker/Dockerfile
DOCKER_IMAGE_NAME=pktd

./do
echo "Binary built. Building Docker image now."

if which docker; then
	fpm -n $RPM_PACKAGE_NAME -s dir -t rpm $BINARY_FOLDER
	docker build -t DOCKER_IMAGE_NAME . -f DOCKER_FILE_LOCATION
	echo "Docker image built."
	echo ""
	echo "Find this Docker image:"
	echo "$ docker image ls"
	echo ""
	echo "Run this Docker image"
	echo "$ docker run -d -p 8080:8080 ${DOCKER_IMAGE_NAME}"
else
	echo "docker not installed or not reachable"
	exit 1
fi
