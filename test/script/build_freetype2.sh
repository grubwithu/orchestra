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

cd freetype2
git checkout 94cb3a2eb96b3f17a1a3bd0e6f7da97c0e1d8f57

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  clang++ $DEFAULT_FLAGS -fsanitize=address,fuzzer-no-link -std=c++11 -lstdc++ \
    -I./install/include/ -I../include/ -I../libarchive-3.4.3/building/install/include/ \
    -larchive ../src/tools/ftfuzzer/ftfuzzer.cc  -o ftfuzzer \
    ./install/lib/libfreetyped.a ../../../pfuzzer/build/libfuzzer.a
  exit 0
fi

mkdir -p src/tools/ftfuzzer/
wget -O src/tools/ftfuzzer/ftfuzzer.cc https://raw.githubusercontent.com/freetype/freetype2-testing/refs/heads/master/fuzzing/src/legacy/ftfuzzer.cc

bash autogen.sh # preinstall: libtool

mkdir -p build-runtime
# rm -rf build-runtime/*
export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS -fsanitize=address,fuzzer-no-link"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS -fsanitize=address,fuzzer-no-link"
#../configure --disable-shared --prefix=$(pwd)/install --with-harfbuzz=no --with-bzip2=no --with-png=no --without-zlib --with-brotli=no
cmake -B build-runtime -DBUILD_SHARED_LIBS=false -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_STANDARD=11 -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/build-runtime/install -DFT_DISABLE_BROTLI=TRUE \
  -DFT_DISABLE_ZLIB=TRUE -DFT_DISABLE_BZIP2=TRUE -DFT_DISABLE_PNG=TRUE
pushd build-runtime
make clean && make -j$JOBS && make install
${CC} -fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS -fsanitize=address,fuzzer -std=c++11 \
  -larchive -I./install/include/ -I../include/ ../src/tools/ftfuzzer/ftfuzzer.cc \
  -o ftfuzzer_cov ./install/lib/libfreetyped.a
${CC} $DEFAULT_FLAGS -fsanitize=address,fuzzer-no-link -std=c++11 -lstdc++ \
  -larchive -I./install/include/ -I../include/ ../src/tools/ftfuzzer/ftfuzzer.cc \
  -o ftfuzzer ./install/lib/libfreetyped.a ../../../pfuzzer/build/libfuzzer.a
popd

mkdir -p build__HFC_qzmp__
rm -rf build__HFC_qzmp__/*
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="$DEFAULT_FLAGS -fsanitize=fuzzer-no-link"
export CFLAGS="$DEFAULT_FLAGS -fsanitize=fuzzer-no-link"
# ../configure --disable-shared --prefix=$(pwd)/install --with-harfbuzz=no --with-bzip2 --with-png --with-zlib --with-brotli
cmake -B build__HFC_qzmp__ -DBUILD_SHARED_LIBS=false -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_STANDARD=11 -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/build__HFC_qzmp__/install -DFT_DISABLE_BROTLI=TRUE \
  -DFT_DISABLE_ZLIB=TRUE -DFT_DISABLE_BZIP2=TRUE -DFT_DISABLE_PNG=TRUE
pushd build__HFC_qzmp__
make clean && make -j$JOBS && make install

${CXX} -c $DEFAULT_FLAGS -fsanitize=fuzzer -std=c++11 -larchive \
  -I./install/include/ -I../include/ \
  ../src/tools/ftfuzzer/ftfuzzer.cc -o ftfuzzer.o \
  ./install/lib/libfreetyped.a
${CXX} $DEFAULT_FLAGS -fsanitize=fuzzer -std=c++11 -larchive \
  ftfuzzer.o  -o ftfuzzer \
  ./install/lib/libfreetyped.a

get-bc -o ftfuzzer.bc ftfuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" ftfuzzer.bc

popd

