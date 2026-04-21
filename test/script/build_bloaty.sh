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

cd bloaty

if ! grep -q "INJECTED FOR TARGET-SPECIFIC FUZZING" CMakeLists.txt; then
cat << 'EOF' >> CMakeLists.txt

# ====== INJECTED FOR TARGET-SPECIFIC FUZZING ======
separate_arguments(FUZZ_CFLAGS_LIST UNIX_COMMAND "$ENV{BLOATY_FUZZ_CFLAGS}")

if(TARGET libbloaty)
    target_compile_options(libbloaty PRIVATE ${FUZZ_CFLAGS_LIST})
endif()
if(TARGET fuzz_target)
    target_compile_options(fuzz_target PRIVATE ${FUZZ_CFLAGS_LIST})
    set_target_properties(fuzz_target PROPERTIES LINK_FLAGS "$ENV{BLOATY_FUZZ_LDFLAGS}")
endif()
# ==================================================
EOF
fi

export CFLAGS="-O1 -fno-omit-frame-pointer -g"
export CXXFLAGS="-O1 -fno-omit-frame-pointer -g"

if [ $UPDATE_PFUZZER -eq 1 ]; then
  cd build-runtime
  export CC="clang"
  export CXX="clang++"
  export LIB_FUZZING_ENGINE="${PFUZZER_LIB}"
  export BLOATY_FUZZ_CFLAGS="-fsanitize=fuzzer-no-link -fsanitize-coverage=trace-cmp"
  export BLOATY_FUZZ_LDFLAGS="-fsanitize-coverage=trace-cmp"
  
  touch ../CMakeLists.txt
  cmake .. -DBUILD_TESTING=off
  make -j$JOBS fuzz_target
  exit 0
fi

mkdir -p build-runtime
pushd build-runtime
# rm -rf *
export CC="clang"
export CXX="clang++"

export LIB_FUZZING_ENGINE="-fsanitize=fuzzer"
export BLOATY_FUZZ_CFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize=fuzzer-no-link -fsanitize-coverage=trace-cmp"
export BLOATY_FUZZ_LDFLAGS="-fprofile-instr-generate -fcoverage-mapping -fsanitize-coverage=trace-cmp"

cmake .. -DBUILD_TESTING=off
make -j$JOBS fuzz_target
cp fuzz_target fuzz_target_cov

export LIB_FUZZING_ENGINE="${PFUZZER_LIB}"
export BLOATY_FUZZ_CFLAGS="-fsanitize=fuzzer-no-link -fsanitize-coverage=trace-cmp"
export BLOATY_FUZZ_LDFLAGS="-fsanitize-coverage=trace-cmp"

touch ../CMakeLists.txt
cmake .. -DBUILD_TESTING=off
make -j$JOBS fuzz_target

popd


mkdir -p build__HFC_qzmp__
pushd build__HFC_qzmp__
rm -rf *
export CC="gclang"
export CXX="gclang++"

export BLOATY_FUZZ_CFLAGS="-fsanitize=fuzzer-no-link"
export BLOATY_FUZZ_LDFLAGS="-fsanitize=fuzzer-no-link"
export LIB_FUZZING_ENGINE="-fsanitize=fuzzer"

cmake .. -DBUILD_TESTING=off
make -j$JOBS fuzz_target


get-bc -o fuzz_target.bc fuzz_target
opt -load-pass-plugin=${FUZZ_INTRO} -passes="fuzz-introspector" fuzz_target.bc

popd