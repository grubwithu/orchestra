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
# PFUZZER_LIB="${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a"

cd jsoncpp
git checkout 941802d466ff6117508e326025720b74d67636f0

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CXX} $CXXFLAGS \
    -I$(pwd)/install/include \
    ../src/test_lib_json/fuzz.cpp \
    -o jsoncpp_fuzzer \
    $(pwd)/install/lib/libjsoncpp.a \
    ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *
export CC="clang"
export CXX="clang++"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake .. \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
  -DCMAKE_INSTALL_LIBDIR=lib \
  -DBUILD_SHARED_LIBS=OFF \
  -DBUILD_STATIC_LIBS=ON \
  -DJSONCPP_WITH_TESTS=OFF \
  -DJSONCPP_WITH_POST_BUILD_UNITTEST=OFF \
  -DJSONCPP_WITH_PKGCONFIG_SUPPORT=OFF \
  -DJSONCPP_WITH_CMAKE_PACKAGE=OFF \
  -DJSONCPP_WITH_EXAMPLE=OFF

make -j$JOBS
make install

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../src/test_lib_json/fuzz.cpp \
  -o jsoncpp_fuzzer_cov \
  $(pwd)/install/lib/libjsoncpp.a

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../src/test_lib_json/fuzz.cpp \
  -o jsoncpp_fuzzer \
  $(pwd)/install/lib/libjsoncpp.a \
  ${PFUZZER_LIB}

popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

cmake .. \
  -DCMAKE_INSTALL_PREFIX=$(pwd)/install \
  -DCMAKE_INSTALL_LIBDIR=lib \
  -DBUILD_SHARED_LIBS=OFF \
  -DBUILD_STATIC_LIBS=ON \
  -DJSONCPP_WITH_TESTS=OFF \
  -DJSONCPP_WITH_POST_BUILD_UNITTEST=OFF \
  -DJSONCPP_WITH_PKGCONFIG_SUPPORT=OFF \
  -DJSONCPP_WITH_CMAKE_PACKAGE=OFF \
  -DJSONCPP_WITH_EXAMPLE=OFF

make -j$JOBS
make install

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include \
  ../src/test_lib_json/fuzz.cpp \
  -o jsoncpp_fuzzer.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -o jsoncpp_fuzzer \
  jsoncpp_fuzzer.o \
  $(pwd)/install/lib/libjsoncpp.a

get-bc -o jsoncpp_fuzzer.bc jsoncpp_fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" jsoncpp_fuzzer.bc

popd
