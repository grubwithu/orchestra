#!/bin/bash
set -e

UPDATE_PFUZZER=0
while [[ $# -gt 0 ]]; do
  case $1 in
    --update-pfuzzer)
      UPDATE_PFUZZER=1
      shift
      ;;
    *)
      shift
      ;;
  esac
done

JOBS=$(($(grep -c ^processor /proc/cpuinfo) < 16 ? $(grep -c ^processor /proc/cpuinfo) : 16))

WORKSPACE_DIR=$(pwd)
PFUZZER_LIB="${PFUZZER_LIB:-${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a}"

if [ ! -d "systemd" ]; then
  echo "Cloning systemd repository..."
  git clone --depth 1 https://github.com/systemd/systemd.git
fi

cd systemd

if [ ! -f "${PFUZZER_LIB}" ]; then
  echo "PFUZZER_LIB not found: ${PFUZZER_LIB}" >&2
  exit 1
fi

rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip install -r .github/workflows/requirements.txt --require-hashes
pip install jinja2

mkdir -p libfuzzer/lib
cp "${PFUZZER_LIB}" libfuzzer/lib/libFuzzingEngine.a

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  mkdir -p build-runtime-pfuzzer
  pushd build-runtime-pfuzzer
  rm -rf ./*

  export CC=clang
  export CXX=clang++
  export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  export LDFLAGS="-L$(pwd)/../libfuzzer/lib"
  export LIBRARY_PATH="$(pwd)/../libfuzzer/lib${LIBRARY_PATH:+:$LIBRARY_PATH}"

  meson setup .. \
    --prefix=$(pwd)/install \
    --auto-features=disabled \
    -Dbuildtype=debugoptimized \
    -Doptimization=1 \
    -Doss-fuzz=true \
    -Dlibmount=disabled \
    -Dnspawn=enabled \
    -Dresolve=true \
    -Dtests=false \
    -Dman=false \
    -Dimportd=false \
    -Db_lundef=false \
    -Dstatic-libsystemd=pic

  ninja -j$JOBS fuzz-link-parser
  mkdir -p ../build-runtime
  cp fuzz-link-parser ../build-runtime/fuzz-link-parser

  popd
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
rm -rf ./*

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

meson setup .. \
  --prefix=$(pwd)/install \
  -Dbuildtype=debugoptimized \
  -Doptimization=1 \
  -Dllvm-fuzz=true \
  -Dlibmount=disabled \
  -Dtests=false \
  -Dman=false \
  -Dimportd=false \
  -Db_lundef=false \
  -Dstatic-libsystemd=pic

ninja -j$JOBS fuzz-link-parser
mv fuzz-link-parser ./fuzz-link-parser_cov

popd

mkdir -p build-runtime-pfuzzer
pushd build-runtime-pfuzzer
rm -rf ./*

export CC=clang
export CXX=clang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export LDFLAGS="-L$(pwd)/../libfuzzer/lib"
export LIBRARY_PATH="$(pwd)/../libfuzzer/lib${LIBRARY_PATH:+:$LIBRARY_PATH}"

meson setup .. \
  --prefix=$(pwd)/install \
  --auto-features=disabled \
  -Dbuildtype=debugoptimized \
  -Doptimization=1 \
  -Doss-fuzz=true \
  -Dlibmount=disabled \
  -Dnspawn=enabled \
  -Dresolve=true \
  -Dtests=false \
  -Dman=false \
  -Dimportd=false \
  -Db_lundef=false \
  -Dstatic-libsystemd=pic

ninja -j$JOBS fuzz-link-parser
cp fuzz-link-parser ../build-runtime/fuzz-link-parser

popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf ./*

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

meson setup .. \
  --prefix=$(pwd)/install \
  -Dbuildtype=debugoptimized \
  -Doptimization=1 \
  -Dllvm-fuzz=true \
  -Dlibmount=disabled \
  -Dtests=false \
  -Dman=false \
  -Dimportd=false \
  -Db_lundef=false \
  -Dstatic-libsystemd=pic

ninja -j$JOBS fuzz-link-parser

SHARED_STATIC_LIB=$(find src/shared -maxdepth 1 -name "libsystemd-shared-*.a" | head -n 1)

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o fuzz-link-parser \
  udevadm.p/src_udev_net_link-config.c.o \
  udevadm.p/src_udev_udev-builtin.c.o \
  udevadm.p/src_udev_udev-builtin-btrfs.c.o \
  udevadm.p/src_udev_udev-builtin-dissect_image.c.o \
  udevadm.p/src_udev_udev-builtin-factory_reset.c.o \
  udevadm.p/src_udev_udev-builtin-hwdb.c.o \
  udevadm.p/src_udev_udev-builtin-input_id.c.o \
  udevadm.p/src_udev_udev-builtin-keyboard.c.o \
  udevadm.p/src_udev_udev-builtin-net_driver.c.o \
  udevadm.p/src_udev_udev-builtin-net_id.c.o \
  udevadm.p/src_udev_udev-builtin-net_setup_link.c.o \
  udevadm.p/src_udev_udev-builtin-path_id.c.o \
  udevadm.p/src_udev_udev-builtin-uaccess.c.o \
  udevadm.p/src_udev_udev-builtin-usb_id.c.o \
  udevadm.p/src_udev_udev-config.c.o \
  udevadm.p/src_udev_udev-ctrl.c.o \
  udevadm.p/src_udev_udev-dump.c.o \
  udevadm.p/src_udev_udev-error.c.o \
  udevadm.p/src_udev_udev-event.c.o \
  udevadm.p/src_udev_udev-format.c.o \
  udevadm.p/src_udev_udev-manager.c.o \
  udevadm.p/src_udev_udev-manager-ctrl.c.o \
  udevadm.p/src_udev_udev-node.c.o \
  udevadm.p/src_udev_udev-rules.c.o \
  udevadm.p/src_udev_udev-spawn.c.o \
  udevadm.p/src_udev_udev-varlink.c.o \
  udevadm.p/src_udev_udev-watch.c.o \
  udevadm.p/src_udev_udev-worker.c.o \
  udevadm.p/meson-generated_.._src_udev_link-config-gperf.c.o \
  fuzz-link-parser.p/src_udev_net_fuzz-link-parser.c.o \
  "${SHARED_STATIC_LIB}" \
  src/libsystemd/libsystemd_static.a \
  src/basic/libbasic.a \
  src/libc/libc-wrapper.a \
  -ldl -lssl -lcrypto -lrt -lm -pthread

get-bc -o fuzz-link-parser.bc fuzz-link-parser
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzz-link-parser.bc

popd
