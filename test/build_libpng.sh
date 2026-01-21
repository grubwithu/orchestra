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

cd libpng
git checkout ba980b8 

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -flto -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  ${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
    -I$(pwd)/install/include \
    ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
    ../../../pfuzzer/build/libfuzzer.a \
    .libs/libpng16.a -lz -lm -lstdc++ \
    -o libpng_read_fuzzer
  exit 0
fi

mkdir -p build__HFC_qzmp__
cd build__HFC_qzmp__
rm -rf *

# Make sure CC and CXX is specified version
# Make sure ld, ar, ranlib is corresponding to the compiler

export CXXFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link -fuse-ld=gold $DEFAULT_FLAGS"

../configure --disable-shared --prefix=$(pwd)/install && FUZZ_INTROSPECTOR=1 make -j && make install

FUZZ_INTROSPECTOR=1 ${CC} -g -fsanitize=fuzzer -fuse-ld=gold $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer

export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cd ..
mkdir -p build-runtime
cd build-runtime
rm -rf *

../configure --disable-shared --prefix=$(pwd)/install && make -j && make install

${CC} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer_cov

${CC} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../contrib/oss-fuzz/libpng_read_fuzzer.cc \
  ../../../pfuzzer/build/libfuzzer.a \
  .libs/libpng16.a -lz -lm -lstdc++ \
  -o libpng_read_fuzzer

