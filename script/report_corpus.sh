#!/bin/bash

CORPUS=$1
# make sure CORPUS is abs path
if [ ! -d "$CORPUS" ]; then
  echo "Error: $CORPUS is not a directory"
  exit 1
fi
CORPUS=$(realpath $CORPUS)

PERIOD=$2
if [ -z "$PERIOD" ]; then
  PERIOD="begin"
fi

curl -X POST -H "Content-Type: application/json" \
  -d '{"fuzzer": "AFL", "period": "'$PERIOD'", "identity": "master_1", "corpus": ["'$CORPUS'"]}' \
  http://localhost:8080/reportCorpus

