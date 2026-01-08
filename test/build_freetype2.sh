#!/bin/bash
set -e

cd freetype2
git checkout 94cb3a2eb96b3f17a1a3bd0e6f7da97c0e1d8f57

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -flto -g"

if [ ! -d "libarchive-3.4.3" ]; then
  wget https://github.com/libarchive/libarchive/releases/download/v3.4.3/libarchive-3.4.3.tar.xz 
  tar -xvf libarchive-3.4.3.tar.xz 
  rm libarchive-3.4.3.tar.xz

  pushd libarchive-3.4.3/
  mkdir -p building
  cd building
  export CXXFLAGS="$DEFAULT_FLAGS -fsanitize=fuzzer-no-link"
  export CFLAGS="$DEFAULT_FLAGS -fsanitize=fuzzer-no-link"
  ../configure --disable-shared --prefix=$(pwd)/install
  make clean && make -j$(nproc) && make install
  popd
fi

mkdir -p src/tools/ftfuzzer/
wget -O src/tools/ftfuzzer/ftfuzzer.cc https://raw.githubusercontent.com/freetype/freetype2-testing/refs/heads/master/fuzzing/src/legacy/ftfuzzer.cc

bash autogen.sh # preinstall: libtool
mkdir -p build
rm -rf build/*
export CXXFLAGS="-fuse-ld=gold $DEFAULT_FLAGS -fsanitize=fuzzer-no-link"
export CFLAGS="-fuse-ld=gold $DEFAULT_FLAGS -fsanitize=fuzzer-no-link"
# ../configure --disable-shared --prefix=$(pwd)/install --with-harfbuzz=no --with-bzip2 --with-png --with-zlib --with-brotli
cmake -B build -DBUILD_SHARED_LIBS=false -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_STANDARD=11 -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/build/install -DFT_DISABLE_BROTLI=TRUE \
  -DFT_DISABLE_ZLIB=TRUE -DFT_DISABLE_BZIP2=TRUE -DFT_DISABLE_PNG=TRUE
pushd build
make clean && FUZZ_INTROSPECTOR=1 make -j && make install
FUZZ_INTROSPECTOR=1 ${CC} $DEFAULT_FLAGS -fsanitize=fuzzer -fuse-ld=gold -std=c++11 \
  -I./install/include/ -I../include/ -I../libarchive-3.4.3/building/install/include/ \
  ../src/tools/ftfuzzer/ftfuzzer.cc  -o ftfuzzer \
  ./install/lib/libfreetyped.a  ../libarchive-3.4.3/building/install/lib/libarchive.a
popd

mkdir -p build-runtime
rm -rf build-runtime/*
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS -fsanitize=address,fuzzer-no-link"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS -fsanitize=address,fuzzer-no-link"
#../configure --disable-shared --prefix=$(pwd)/install --with-harfbuzz=no --with-bzip2=no --with-png=no --without-zlib --with-brotli=no
cmake -B build-runtime -DBUILD_SHARED_LIBS=false -DCMAKE_BUILD_TYPE=Debug -DCMAKE_CXX_STANDARD=11 -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/build-runtime/install -DFT_DISABLE_BROTLI=TRUE \
  -DFT_DISABLE_ZLIB=TRUE -DFT_DISABLE_BZIP2=TRUE -DFT_DISABLE_PNG=TRUE
pushd build-runtime
make clean && make -j && make install
${CC} -fprofile-instr-generate -fcoverage-mapping $DEFAULT_FLAGS -fsanitize=address,fuzzer -std=c++11 \
  -I./install/include/ -I../include/ -I../libarchive-3.4.3/building/install/include/ \
  ../src/tools/ftfuzzer/ftfuzzer.cc  -o ftfuzzer_cov \
  ./install/lib/libfreetyped.a  ../libarchive-3.4.3/building/install/lib/libarchive.a
${CC} $DEFAULT_FLAGS -fsanitize=address,fuzzer -std=c++11 -lstdc++ \
  -I./install/include/ -I../include/ -I../libarchive-3.4.3/building/install/include/ \
  ../src/tools/ftfuzzer/ftfuzzer.cc  -o ftfuzzer \
  ./install/lib/libfreetyped.a  ../libarchive-3.4.3/building/install/lib/libarchive.a \
  ../../../pfuzzer/build/libfuzzer.a
popd
