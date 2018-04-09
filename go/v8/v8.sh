#! /usr/bin/env bash

# Public Domain (-) 2018-present, The Espian Source Authors.
# See the Espian Source UNLICENSE file for details.

set -e -o pipefail

function stage {
  printf "\n\033[34;1m➡  $1\033[0m\n\n"
}

cd "$(dirname "$0")"
source version.sh

OS_NAME=$(uname -s | tr 'A-Z' 'a-z')
TARBALL="v8.${OS_NAME}.x64-${V8WORKER_V8_VERSION}.tar"

if [[ ! -d "build" ]]; then
    stage "Create the build directory"
    mkdir build
fi

cd build

if [[ ! -d "depot_tools" ]]; then
    stage "Clone the depot_tools repo"
    git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
fi

PATH="`pwd`/depot_tools":"$PATH"

stage "Ensure gclient is up to date"
gclient

if [[ ! -d "v8" ]]; then
    stage "Fetch V8"
    fetch --nohooks v8
else
    stage "Update V8 checkout"
    cd v8
    gclient sync
    cd ..
fi

stage "Checkout V8 v${V8WORKER_V8_VERSION}"
cd v8
git checkout "${V8WORKER_V8_VERSION_ID}"

stage "Sync third-party dependencies"
gclient sync

case $OS_NAME in
    linux)
        stage "Install additional dependencies for Linux";
        ./build/install-build-deps.sh --no-prompt;;
esac

stage "Create the GN build directory"
./tools/dev/v8gen.py x64.release -- \
    is_component_build=false v8_enable_gdbjit=false \
    v8_enable_i18n_support=false v8_static_library=true \
    v8_use_external_startup_data=false

stage "Build V8"
ninja -C out.gn/x64.release

cd ..
if [[ -d "include" ]] || [[ -d "lib" ]]; then
    stage "Remove stale directories"
    rm -rf include
    rm -rf lib
fi

mkdir -p include/libplatform
mkdir -p "lib/${OS_NAME}.x64"

stage "Copy header and compiled files"
cp v8/include/*.h include/
cp v8/include/libplatform/*.h include/libplatform/
cp v8/out.gn/x64.release/obj/libv8_*.a "lib/${OS_NAME}.x64/"

stage "Create ${TARBALL}"
tar cf "${TARBALL}" include lib
rm -rf include
rm -rf lib
cd ..
mkdir -p distfile
mv "build/${TARBALL}" "distfile/${TARBALL}"
