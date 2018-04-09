#! /usr/bin/env bash

# Public Domain (-) 2018-present, The Espian Source Authors.
# See the Espian Source UNLICENSE file for details.

set -e -o pipefail

function stage {
  printf "\n\033[34;1m➡  $1\033[0m\n\n"
}

cd "$(dirname "$0")"
source version.sh

TARBALL="v8.linux.x64-${V8WORKER_V8_VERSION}.tar"

stage "Tar up the build scripts for the docker image"
tar cf scripts.tar Dockerfile *.sh

stage "Build the docker image"
docker build -t v8 - < scripts.tar

exists=`docker ps -a -f name=v8 | grep v8 || :`
if [[ ! -z "$exists" ]]; then
    stage "Remove existing container"
    docker rm v8
fi

stage "Run the docker image"
docker run -it --name v8 v8

stage "Copy tarball to the host distfile directory"
mkdir -p distfile
docker cp "v8:/distfile/${TARBALL}" "distfile/${TARBALL}"

stage "Cleanup after ourselves"
docker rm v8
rm scripts.tar
