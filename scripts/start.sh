#!/bin/bash

set -eu
set -o pipefail
if [[ ! -f "config.ini" ]];then
  cp ./config.ini.example config.ini
fi
cd $(dirname $0)/../
GO_ENV=${GO_ENV:-dev}
export GO_ENV
go get github.com/tools/godep
godep > /dev/null 2>&1 || godep get
go install ./...

if [[ ("${GO_ENV}" == "test") && (! "`ps -ef | grep '[t]est_ldap'`") ]] ; then
  ./test_ldap_server/test_ldap_server > log/ldap.log 2>&1 &
fi
test ! -f ./lfs-server-go && godep go build
./lfs-server-go
