## Building V8

First, download `depot_tools`:

    git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git

Add it to your `$PATH`:

    export PATH=<depot_tools-path>:"$PATH"

Ensure `depot_tools` is up to date by running:

    gclient

Grab the `v8` source code:

    mkdir v8
    cd v8
    fetch --nohooks v8
    cd v8

Check out the latest "last known good revision" of the desired branch, e.g. for
version `6.6`, do:

    git checkout 6.6-lkgr
    gclient sync

If you are on `linux`, then you need to install some build dependencies using:

    ./build/install-build-deps.sh

Create the `GN` build directory:

    ./tools/dev/v8gen.py x64.release -- \
        is_component_build=false v8_enable_gdbjit=false \
        v8_enable_i18n_support=false v8_static_library=true \
        v8_use_external_startup_data=false

This will create the build directory in `out.gn/x64.release`. You can see the
full list of GN arguments by running:

    gn args --list out.gn/x64.release

Finally, build `v8`:

    ninja -C out.gn/x64.release

Create the appropriate subdirectories within the `v8worker` directory:

    mkdir -p <v8worker-path>/include/libplatform
    mkdir -p <v8worker-path>/lib/<platform>.x64

Copy over the header and compiled files:

    cp include/*.h <v8worker-path>/include
    cp include/libplatform/*.h <v8worker-path>/include/libplatform/
    cp out.gn/x64.release/obj/libv8_*.a <v8worker-path>/lib/<platform>.x64

Currently only `darwin` and `linux` platforms are supported. But others can be
added fairly easily.

## Building v8worker

You are now all set to build executables that depend on `v8worker`.

One aspect to note on macOS, is the [pending issue] on the slowness of combining
the DWARF debug info. You can speed things up by instructing the linker to omit
the symbol table and debug info during builds:

    go build -ldflags=-s

This will also create significantly smaller binaries.

[pending issue]: https://github.com/golang/go/issues/12259
