#!/bin/bash

set -euo pipefail

JOBS=4
STATUS_DIR=""

usage() {
  echo "Usage: $0 [-j JOBS|--jobs JOBS]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -j|--jobs)
      if [[ $# -lt 2 ]]; then
        usage
        exit 1
      fi
      JOBS="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if ! [[ "$JOBS" =~ ^[1-9][0-9]*$ ]]; then
  echo "Invalid jobs count: $JOBS" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd "${SCRIPT_DIR}/.." && pwd)
TEST_DIR="${ROOT_DIR}/test"

cleanup() {
  if [[ -n "${STATUS_DIR}" && -d "${STATUS_DIR}" ]]; then
    rm -rf "${STATUS_DIR}"
  fi
}

record_status() {
  local image_tag="$1"
  local status="$2"
  local details="$3"

  printf '%s|%s|%s\n' "${image_tag}" "${status}" "${details}" > "${STATUS_DIR}/${image_tag//[:\/]/_}.status"
}

run_build() {
  local image_tag="$1"
  local rc
  shift

  echo "Building ${image_tag}"
  if "$@"; then
    record_status "${image_tag}" "SUCCESS" "build completed"
    return 0
  fi

  rc=$?
  record_status "${image_tag}" "FAILED" "build command exited with code ${rc}"
  return "${rc}"
}

summarize_results() {
  local status_file
  local image_tag
  local status
  local details
  local failed=0

  echo
  echo "Build summary:"
  for status_file in "${STATUS_DIR}"/*.status; do
    [[ -e "${status_file}" ]] || continue
    IFS='|' read -r image_tag status details < "${status_file}"
    printf '  [%s] %s - %s\n' "${status}" "${image_tag}" "${details}"
    if [[ "${status}" != "SUCCESS" ]]; then
      failed=1
    fi
  done

  return "${failed}"
}

STATUS_DIR=$(mktemp -d)
trap cleanup EXIT

serial_failed=0
if ! run_build "hfc-base:latest" docker build -t hfc-base:latest "${ROOT_DIR}"; then
  serial_failed=1
fi

if ! run_build "hfc-test:latest" docker build -t hfc-test:latest -f "${TEST_DIR}/base.Dockerfile" "${TEST_DIR}"; then
  serial_failed=1
fi

mapfile -t DOCKERFILES < <(find "${TEST_DIR}/dockerfile" -maxdepth 1 -type f \( -name '*.Dockerfile' -o -name '*.dockerfile' \) | sort)

if [[ ${#DOCKERFILES[@]} -eq 0 ]]; then
  echo "No Dockerfiles found under ${TEST_DIR}/dockerfile"
  exit 0
fi

build_target_image() {
  local dockerfile="$1"
  local filename
  local image_name
  local image_tag

  filename=$(basename "${dockerfile}")
  image_name="${filename%.Dockerfile}"
  image_name="${image_name%.dockerfile}"
  image_tag="hfc-${image_name}:latest"

  echo "Building ${image_tag} from ${filename}"
  if docker build -t "${image_tag}" -f "${dockerfile}" "${TEST_DIR}"; then
    record_status "${image_tag}" "SUCCESS" "built from ${filename}"
    return 0
  fi

  record_status "${image_tag}" "FAILED" "failed while building ${filename}"
  return 1
}

export TEST_DIR
export STATUS_DIR
export -f record_status
export -f build_target_image

parallel_failed=0
if ! printf '%s\n' "${DOCKERFILES[@]}" | xargs -r -n 1 -P "${JOBS}" -I {} bash -c 'build_target_image "$1"' _ {}; then
  parallel_failed=1
fi

summary_failed=0
if ! summarize_results; then
  summary_failed=1
fi

if [[ "${serial_failed}" -ne 0 || "${parallel_failed}" -ne 0 || "${summary_failed}" -ne 0 ]]; then
  exit 1
fi
