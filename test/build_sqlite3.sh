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

cd sqlite3
git checkout 4d9384cba35ce7971431da9b543e0f9d68975947

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -flto -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  ${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -lstdc++ \
    -I$(pwd)/install/include \
    ../test/ossfuzz.c \
    -o ossfuzz \
    $(pwd)/install/lib/libsqlite3.a \
    ../../../pfuzzer/build/libfuzzer.a
  exit 0
fi

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *

# Make sure CC and CXX is specified version
# Make sure ld, ar, ranlib is corresponding to the compiler

export CXXFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"

../configure --shared=0 --prefix=$(pwd)/install --disable-tcl && FUZZ_INTROSPECTOR=1 make -j && make install
FUZZ_INTROSPECTOR=1 ${CC} -fsanitize=fuzzer -fuse-ld=gold $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz \
  $(pwd)/install/lib/libsqlite3.a
popd

mkdir -p build-runtime
pushd build-runtime
rm -rf *
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
../configure --shared=0 --prefix=$(pwd)/install --disable-tcl && make -j && make install
${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz_cov \
  $(pwd)/install/lib/libsqlite3.a
${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -lstdc++ \
  -I$(pwd)/install/include \
  ../test/ossfuzz.c \
  -o ossfuzz \
  $(pwd)/install/lib/libsqlite3.a \
  ../../../pfuzzer/build/libfuzzer.a
popd
