#!/bin/bash
set -e

banner() {
echo "================================================================"
echo "$1"
echo "================================================================"
}

PROJECT="kind"

banner "Building docker image "$PROJECT" ..."
docker build -t $PROJECT .
banner "Copying the binary..."
docker run -v ${PWD}/bin:/out:rw $PROJECT
