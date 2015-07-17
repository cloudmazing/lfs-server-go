#!/bin/bash

set -eu
set -o pipefail

cd $(dirname $0)/../
GO_ENV=${GO_ENV:-dev}
export GO_ENV

[[ "${GO_ENV}" == "dev" ]] && [[ ! "`ps -ef | grep '[t]est_ldap'`" ]] && ./test_ldap/test_ldap

[[ ! -f "lfs-server-go" ]] && go build
./lfs-server-go
