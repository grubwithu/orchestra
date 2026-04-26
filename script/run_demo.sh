#!/bin/bash
set -e

cleanup() {
  kill $HFC_PID
  # rm -rf tmp_xx223
}
trap cleanup EXIT INT TERM

# Make sure that the script is run from the root directory of the project
## 0. Check argument
TARGET=$1
shift

HFC_ONLY=0
PASS_ARGS=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --hfc-only)
      HFC_ONLY=1
      shift
      ;;
    --default)
      PASS_ARGS+=("-fork=4 -fuzzers=afl,fairfuzz,aflfast,mopt,aflsmart,darwin,lafintel,redqueen,entropic")
    *)
      PASS_ARGS+="$1 "
      shift
      ;;
  esac
done
if [ -z "$TARGET" ]; then
  echo "Usage: $0 [--skip-build] <target>"
  echo "  --skip-build    Skip compilation steps and directly run the demo"
  exit 1
fi
PROGNAME=""

case "$TARGET" in
  bloaty)
    PROGNAME="fuzz_target"
    ;;
  curl)
    PROGNAME="curl_fuzzer_http"
    ;;
  freetype2)
    PROGNAME="ftfuzzer"
    ;;
  harfbuzz)
    PROGNAME="hb-shape-fuzzer"
    ;;
  jsoncpp)
    PROGNAME="jsoncpp_fuzzer"
    ;;
  lcms)
    PROGNAME="cms_transform_fuzzer"
    ;;
  libjpeg|libjpeg-turbo)
    PROGNAME="libjpeg_turbo_fuzzer"
    ;;
  libpcap)
    PROGNAME="fuzz_both"
    ;;
  libpng)
    PROGNAME="libpng_read_fuzzer"
    ;;
  libxml2)
    PROGNAME="xml"
    ;;
  libxslt)
    PROGNAME="xpath"
    ;;
  mbedtls)
    PROGNAME="fuzz_dtlsclient"
    ;;
  openssl)
    PROGNAME="x509"
    ;;
  openthread)
    PROGNAME="ot-ip6-send-fuzzer"
    ;;
  proj4)
    PROGNAME="proj_crs_to_crs_fuzzer"
    ;;
  re2)
    PROGNAME="fuzzer"
    ;;
  sqlite3)
    PROGNAME="ossfuzz"
    ;;
  systemd)
    PROGNAME="fuzz-link-parser"
    ;;
  vorbis)
    PROGNAME="decode_fuzzer"
    ;;
  woff2)
    PROGNAME="convert_woff2ttf_fuzzer"
    ;;
  zlib)
    PROGNAME="zlib_uncompress_fuzzer"
    ;;
  *)
    echo "Unknown target: $TARGET"
    exit 1
    ;;
esac

## 4. Run HFC
pushd test/submodule/$TARGET/
pushd build__HFC_qzmp__/
DATA_FILE=$(ls ./fuzzerLogFile*.data | head -n 1)
DATA_FILE_ABS=$(realpath $DATA_FILE)
popd
pushd build-runtime/
PROG_FILE=$(realpath ${PROGNAME}_cov)
popd
popd
if [ $HFC_ONLY -eq 0 ]; then
  build/hfc -verbose -fuzzintro=$DATA_FILE_ABS -program=$PROG_FILE 1>build/$PROGNAME.log 2>&1 & HFC_PID=$!
elif [ $HFC_ONLY -eq 1 ]; then
  build/hfc -verbose -fuzzintro=$DATA_FILE_ABS -program=$PROG_FILE 1>build/$PROGNAME.log 2>&1
fi

$AFL_SYSTEM_CONFIG || true
export AFL_SKIP_CPUFREQ=1
export AFL_NO_AFFINITY=1
export HFC_URL=http://localhost:8080

## 5. Run pfuzzer
if [ $HFC_ONLY -eq 0 ]; then
  mkdir -p tmp_xx223

  timeout -s INT 24h test/submodule/${TARGET}/build-runtime/${PROGNAME} tmp_xx223/ test/${TARGET}_seeds/ \
  -rss_limit_mb=0 -max_len=1048575 -ignore_crashes=1 -entropic=0 -strategy_threshold=120 $PASS_ARGS
fi
