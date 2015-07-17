#!/bin/bash

set -eu
set -o pipefail

cd $(dirname $0)/../
GO_ENV=${GO_ENV:-dev}
export GO_ENV

if [[ ("${GO_ENV}" == "dev") && (! "`ps -ef | grep '[t]est_ldap'`") ]] ; then
  ./test_ldap/test_ldap > log/ldap.log 2>&1 &
fi

[[ ! -f "lfs-server-go" ]] && go build
./lfs-server-go
