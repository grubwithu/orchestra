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

if [ $TARGET == "bloaty" ]; then
  PROGNAME=fuzz_target
elif [ $TARGET == "libjpeg-turbo" ]; then
  PROGNAME=libjpeg_turbo_fuzzer
elif [ $TARGET == "libxml2" ]; then
  PROGNAME=xml
elif [ $TARGET == "proj4" ]; then
  PROGNAME=proj_crs_to_crs_fuzzer
elif [ $TARGET == "libpng" ]; then
  PROGNAME=libpng_read_fuzzer
elif [ $TARGET == "freetype2" ]; then
  PROGNAME=ftfuzzer
elif [ $TARGET == "sqlite3" ]; then
  PROGNAME=ossfuzz
elif [ $TARGET == "harfbuzz" ]; then
  PROGNAME=hb-shape-fuzzer
else
  echo "Unknown target: $TARGET"
  exit 1
fi

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
  -rss_limit_mb=0 -max_len=1048575 -ignore_crashes=1 -entropic=0 \
  -fork=4 -fuzzers=afl,fairfuzz,aflfast,redqueen,entropic $PASS_ARGS
fi