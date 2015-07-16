#!/bin/bash

set -eu
set -o pipefail

cd $(dirname $0)/../
GO_ENV=${GO_ENV:-dev}
export GO_ENV
[[ ! -f "lfs-server-go" ]] && go build
./lfs-server-go
