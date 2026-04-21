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

cd curl
git checkout 1fbffe7f08f0d551038520b569b817f58084f77b

if [ ! -d "curl-fuzzer" ]; then
  echo "Cloning curl-fuzzer repository into curl/curl-fuzzer..."
  git clone --depth 1 https://github.com/curl/curl-fuzzer.git
fi

DEFAULT_FLAGS="-fsanitize-coverage=trace-cmp -O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
  
  ${CXX} $CXXFLAGS \
    -I$(pwd)/install/include \
    -I../curl-fuzzer \
    ../curl-fuzzer/curl_fuzzer.cc \
    ../curl-fuzzer/curl_fuzzer_tlv.cc \
    ../curl-fuzzer/curl_fuzzer_callback.cc \
    -o curl_fuzzer_http \
    $(pwd)/install/lib/libcurl.a \
    -lpthread ${PFUZZER_LIB}
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
  -DBUILD_TESTING=OFF \
  -DBUILD_CURL_EXE=OFF \
  -DCURL_USE_OPENSSL=OFF \
  -DUSE_NGHTTP2=ON \
  -DUSE_LIBIDN2=ON \
  -DUSE_LIBPSL=ON \
  -DCURL_ZLIB=OFF \
  -DCURL_BROTLI=OFF \
  -DCURL_ZSTD=OFF \
  -DCURL_DISABLE_LDAP=ON \
  -DCURL_DISABLE_LDAPS=ON \
  -DCURL_DISABLE_DICT=ON \
  -DCURL_DISABLE_FILE=ON \
  -DCURL_DISABLE_GOPHER=ON \
  -DCURL_DISABLE_IMAP=ON \
  -DCURL_DISABLE_POP3=ON \
  -DCURL_DISABLE_RTSP=ON \
  -DCURL_DISABLE_SMB=ON \
  -DCURL_DISABLE_SMTP=ON \
  -DCURL_DISABLE_TELNET=ON \
  -DCURL_DISABLE_TFTP=ON

make -j$JOBS
make install

${CXX} -fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer $DEFAULT_FLAGS \
  -lpsl -lidn2 -lnghttp2 \
  -I$(pwd)/install/include \
  -I../curl-fuzzer \
  ../curl-fuzzer/curl_fuzzer.cc \
  ../curl-fuzzer/curl_fuzzer_tlv.cc \
  ../curl-fuzzer/curl_fuzzer_callback.cc \
  -o curl_fuzzer_http_cov \
  $(pwd)/install/lib/libcurl.a \
  -lpthread

${CXX} -fsanitize=fuzzer-no-link $DEFAULT_FLAGS \
  -lpsl -lidn2 -lnghttp2 \
  -I$(pwd)/install/include \
  -I../curl-fuzzer \
  ../curl-fuzzer/curl_fuzzer.cc \
  ../curl-fuzzer/curl_fuzzer_tlv.cc \
  ../curl-fuzzer/curl_fuzzer_callback.cc \
  -o curl_fuzzer_http \
  $(pwd)/install/lib/libcurl.a \
  -lpthread \
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
  -DBUILD_TESTING=OFF \
  -DBUILD_CURL_EXE=OFF \
  -DCURL_USE_OPENSSL=OFF \
  -DUSE_NGHTTP2=ON \
  -DUSE_LIBIDN2=ON \
  -DUSE_LIBPSL=ON \
  -DCURL_ZLIB=OFF \
  -DCURL_BROTLI=OFF \
  -DCURL_ZSTD=OFF \
  -DCURL_DISABLE_LDAP=ON \
  -DCURL_DISABLE_LDAPS=ON \
  -DCURL_DISABLE_DICT=ON \
  -DCURL_DISABLE_FILE=ON \
  -DCURL_DISABLE_GOPHER=ON \
  -DCURL_DISABLE_IMAP=ON \
  -DCURL_DISABLE_POP3=ON \
  -DCURL_DISABLE_RTSP=ON \
  -DCURL_DISABLE_SMB=ON \
  -DCURL_DISABLE_SMTP=ON \
  -DCURL_DISABLE_TELNET=ON \
  -DCURL_DISABLE_TFTP=ON

make -j$JOBS
make install

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include -I../curl-fuzzer \
  ../curl-fuzzer/curl_fuzzer.cc -o curl_fuzzer.o

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include -I../curl-fuzzer \
  ../curl-fuzzer/curl_fuzzer_tlv.cc -o curl_fuzzer_tlv.o

${CXX} -c -fsanitize=fuzzer $DEFAULT_FLAGS \
  -I$(pwd)/install/include -I../curl-fuzzer \
  ../curl-fuzzer/curl_fuzzer_callback.cc -o curl_fuzzer_callback.o

${CXX} -fsanitize=fuzzer $DEFAULT_FLAGS \
  -lpsl -lidn2 -lnghttp2 \
  -o curl_fuzzer_http \
  curl_fuzzer.o curl_fuzzer_tlv.o curl_fuzzer_callback.o \
  $(pwd)/install/lib/libcurl.a \
  -lpthread

get-bc -o curl_fuzzer_http.bc curl_fuzzer_http
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" curl_fuzzer_http.bc

popd
