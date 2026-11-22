#!/bin/sh
set -eu
cd "$(dirname "$0")"
if [ ! -x ./L2EStudyLink ]; then
    echo 'Compiled server missing. Run: sh scripts/build.sh' >&2
    exit 1
fi
exec ./L2EStudyLink
