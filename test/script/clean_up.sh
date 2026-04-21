#!/bin/bash

echo "Cleaning up intermediate build files to save space..."

WORKSPACE_DIR=$(pwd)

find "${WORKSPACE_DIR}" -name ".git" -type d -exec rm -rf {} +

find "${WORKSPACE_DIR}" -name "CMakeFiles" -type d -exec rm -rf {} +
find "${WORKSPACE_DIR}" -name "CMakeCache.txt" -type f -delete

find "${WORKSPACE_DIR}" -name ".deps" -type d -exec rm -rf {} +
find "${WORKSPACE_DIR}" -name "*.la" -type f -delete
find "${WORKSPACE_DIR}" -name "*.lo" -type f -delete

find "${WORKSPACE_DIR}" -name "*.o" -type f \
  ! -name "xml.o" ! -name "fuzz.o" \
  -delete

echo "Cleanup finished. Target files and source code retained."
