#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
CGO_ENABLED=0 go build -trimpath -o L2EStudyLink .
