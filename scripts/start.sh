#!/bin/bash

set -eu
set -o pipefail

if [[ ! -f "config.ini" ]];then
  cp ./config.ini.example config.ini
fi
cd $(dirname $0)/../
GO_ENV=${GO_ENV:-dev}
export GO_ENV

if [[ ("${GO_ENV}" == "dev") && (! "`ps -ef | grep '[t]est_ldap'`") ]] ; then
  ./test_ldap_server/test_ldap_server > log/ldap.log 2>&1 &
fi

[[ ! -f "lfs-server-go" ]] && go build
./lfs-server-go
