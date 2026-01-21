#!/bin/bash
set -e

UPDATE=0
while [[ $# -gt 0 ]]; do
  case $1 in
    --update)
      UPDATE=1
      shift
      ;;
    *)
      shift
      ;;
  esac
done

# build hfc

make

# build pfuzzer

pushd pfuzzer
bash build.sh
popd

# build targets
cd test/



if [ $UPDATE -eq 1 ]; then
  BUILD_ARGS="--update-pfuzzer"
else 
  BUILD_ARGS=""
fi

bash build_sqlite3.sh $BUILD_ARGS
bash build_libpng.sh $BUILD_ARGS
bash build_freetype2.sh $BUILD_ARGS
bash build_harfbuzz.sh $BUILD_ARGS

