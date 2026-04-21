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
PFUZZER_LIB="${WORKSPACE_DIR}/../pfuzzer/build/libfuzzer.a"

cd openthread
git checkout 1798f3b61c6145ddb62679e3dd4c6fea8416ae2e

DEFAULT_FLAGS="-O1 -g -fno-omit-frame-pointer -fsanitize-coverage=trace-cmp"

# 更新 PFuzzer 版本
if [ $UPDATE_PFUZZER -eq 1 ]; then
  mkdir -p build-runtime
  cd build-runtime
  export CC=clang
  export CXX=clang++
  export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"

  ${CXX} $CXXFLAGS \
    -I../include \
    $FUZZ_TARGET \
    -o ot-ip6-send-fuzzer \
    ../src/libopenthread.a \
    -lm ${PFUZZER_LIB}
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *

export CC=clang
export CXX=clang++
export CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export LIB_FUZZING_ENGINE=${PFUZZER_LIB}

cmake \
  -DCMAKE_C_FLAGS="${CFLAGS}" \
  -DCMAKE_CXX_FLAGS="${CXXFLAGS}" \
  -DBUILD_TESTING=OFF -DOT_BUILD_EXECUTABLES=OFF -DOT_FUZZ_TARGETS=ON \
  -DOT_MTD=OFF -DOT_PLATFORM=external -DOT_RCP=OFF -DOT_BORDER_AGENT=ON \
  -DOT_BORDER_ROUTER=ON -DOT_BORDER_ROUTING=ON -DOT_CHANNEL_MANAGER=ON \
  -DOT_CHANNEL_MONITOR=ON -DOT_COAP=ON -DOT_COAPS=ON -DOT_COAP_BLOCK=ON \
  -DOT_COAP_OBSERVE=ON -DOT_COMMISSIONER=ON -DOT_DATASET_UPDATER=ON \
  -DOT_DHCP6_CLIENT=ON -DOT_DHCP6_SERVER=ON -DOT_DNS_CLIENT=ON \
  -DOT_DNSSD_SERVER=ON -DOT_ECDSA=ON -DOT_HISTORY_TRACKER=ON \
  -DOT_IP6_FRAGM=ON -DOT_JAM_DETECTION=ON -DOT_JOINER=ON -DOT_LINK_RAW=ON \
  -DOT_LOG_OUTPUT=APP -DOT_MAC_FILTER=ON -DOT_MDNS=ON \
  -DOT_NETDATA_PUBLISHER=ON -DOT_NETDIAG_CLIENT=ON -DOT_PING_SENDER=ON \
  -DOT_SERVICE=ON -DOT_SLAAC=ON -DOT_SNTP_CLIENT=ON \
  -DOT_SRP_ADV_PROXY=ON -DOT_SRP_CLIENT=ON -DOT_SRP_SERVER=ON \
  -DOT_THREAD_VERSION=1.3 -DOT_UPTIME=ON \
  ..
make -j$JOBS
mv tests/fuzz/ot-ip6-send-fuzzer ./ot-ip6-send-fuzzer_cov

cmake \
  -DCMAKE_C_FLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS" \
  -DCMAKE_CXX_FLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS" \
  -DBUILD_TESTING=OFF -DOT_BUILD_EXECUTABLES=OFF -DOT_FUZZ_TARGETS=ON \
  -DOT_MTD=OFF -DOT_PLATFORM=external -DOT_RCP=OFF -DOT_BORDER_AGENT=ON \
  -DOT_BORDER_ROUTER=ON -DOT_BORDER_ROUTING=ON -DOT_CHANNEL_MANAGER=ON \
  -DOT_CHANNEL_MONITOR=ON -DOT_COAP=ON -DOT_COAPS=ON -DOT_COAP_BLOCK=ON \
  -DOT_COAP_OBSERVE=ON -DOT_COMMISSIONER=ON -DOT_DATASET_UPDATER=ON \
  -DOT_DHCP6_CLIENT=ON -DOT_DHCP6_SERVER=ON -DOT_DNS_CLIENT=ON \
  -DOT_DNSSD_SERVER=ON -DOT_ECDSA=ON -DOT_HISTORY_TRACKER=ON \
  -DOT_IP6_FRAGM=ON -DOT_JAM_DETECTION=ON -DOT_JOINER=ON -DOT_LINK_RAW=ON \
  -DOT_LOG_OUTPUT=APP -DOT_MAC_FILTER=ON -DOT_MDNS=ON \
  -DOT_NETDATA_PUBLISHER=ON -DOT_NETDIAG_CLIENT=ON -DOT_PING_SENDER=ON \
  -DOT_SERVICE=ON -DOT_SLAAC=ON -DOT_SNTP_CLIENT=ON \
  -DOT_SRP_ADV_PROXY=ON -DOT_SRP_CLIENT=ON -DOT_SRP_SERVER=ON \
  -DOT_THREAD_VERSION=1.3 -DOT_UPTIME=ON \
  ..
make -j$JOBS
mv tests/fuzz/ot-ip6-send-fuzzer ./ot-ip6-send-fuzzer

popd

mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *

export CC=gclang
export CXX=gclang++
export CFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export CXXFLAGS="-fsanitize=fuzzer-no-link $DEFAULT_FLAGS"
export LIB_FUZZING_ENGINE=${PFUZZER_LIB}

cmake \
  -DCMAKE_C_FLAGS="${CFLAGS}" \
  -DCMAKE_CXX_FLAGS="${CXXFLAGS}" \
  -DBUILD_TESTING=OFF -DOT_BUILD_EXECUTABLES=OFF -DOT_FUZZ_TARGETS=ON \
  -DOT_MTD=OFF -DOT_PLATFORM=external -DOT_RCP=OFF -DOT_BORDER_AGENT=ON \
  -DOT_BORDER_ROUTER=ON -DOT_BORDER_ROUTING=ON -DOT_CHANNEL_MANAGER=ON \
  -DOT_CHANNEL_MONITOR=ON -DOT_COAP=ON -DOT_COAPS=ON -DOT_COAP_BLOCK=ON \
  -DOT_COAP_OBSERVE=ON -DOT_COMMISSIONER=ON -DOT_DATASET_UPDATER=ON \
  -DOT_DHCP6_CLIENT=ON -DOT_DHCP6_SERVER=ON -DOT_DNS_CLIENT=ON \
  -DOT_DNSSD_SERVER=ON -DOT_ECDSA=ON -DOT_HISTORY_TRACKER=ON \
  -DOT_IP6_FRAGM=ON -DOT_JAM_DETECTION=ON -DOT_JOINER=ON -DOT_LINK_RAW=ON \
  -DOT_LOG_OUTPUT=APP -DOT_MAC_FILTER=ON -DOT_MDNS=ON \
  -DOT_NETDATA_PUBLISHER=ON -DOT_NETDIAG_CLIENT=ON -DOT_PING_SENDER=ON \
  -DOT_SERVICE=ON -DOT_SLAAC=ON -DOT_SNTP_CLIENT=ON \
  -DOT_SRP_ADV_PROXY=ON -DOT_SRP_CLIENT=ON -DOT_SRP_SERVER=ON \
  -DOT_THREAD_VERSION=1.3 -DOT_UPTIME=ON \
  ..
make -j$JOBS
get-bc -o ot-ip6-send-fuzzer.bc tests/fuzz/ot-ip6-send-fuzzer
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" ot-ip6-send-fuzzer.bc

popd