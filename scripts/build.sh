#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROGDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly BUILDPACKDIR="$(cd "${PROGDIR}/.." && pwd)"

function main() {
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${BUILDPACKDIR}/bin"

  run::build
  cmd::build
}

function usage() {
  cat <<-USAGE
build.sh [OPTIONS]

Builds the buildpack executables.

OPTIONS
  --help  -h  prints the command usage
USAGE
}

function run::build() {
  if [[ -f "${BUILDPACKDIR}/cmd/main/main.go" ]]; then
    pushd "${BUILDPACKDIR}/bin" > /dev/null || return
      printf "%s" "Building run... "

      GOOS=linux \
      CGO_ENABLED=0 \
        go build \
          -ldflags="-s -w" \
          -o "main" \
            "${BUILDPACKDIR}/cmd/main"

      echo "Success!"

      names=("detect")

      if [ -f "${BUILDPACKDIR}/extension.toml" ]; then
        names+=("generate")
      else
        names+=("build")
      fi

      for name in "${names[@]}"; do
        printf "%s" "Linking ${name}... "

        ln -sf "main" "${name}"

        echo "Success!"
      done
    popd > /dev/null || return
  fi
}

function cmd::build() {
  if [[ -f "${BUILDPACKDIR}/cmd/helper/main.go" ]]; then
    pushd "${BUILDPACKDIR}/bin" > /dev/null || return

      printf "%s" "Building helper... "

      GOOS="linux" \
      CGO_ENABLED=0 \
        go build \
          -ldflags="-s -w" \
          -o "helper" \
            "${BUILDPACKDIR}/cmd/helper"

      echo "Success!"

    popd > /dev/null || return
  fi
}

main "${@:-}"
