#!/bin/bash

set -eu
set -o pipefail
if [[ ! -f "config.ini" ]];then
  cp ./config.ini.example config.ini
fi
cd $(dirname $0)/../
GO_ENV=${GO_ENV:-dev}
export GO_ENV
godep > /dev/null 2>&1 || go get github.com/tools/godep
godep restore
godep go install ./...

if [[ ("${GO_ENV}" == "test") && (! "`ps -ef | grep '[t]est_ldap'`") ]] ; then
  ./test_ldap_server/test_ldap_server > log/ldap.log 2>&1 &
fi
test ! -f ./lfs-server-go && godep go build ./...
goreman > /dev/null 2>&1 || go get github.com/mattn/goreman
goreman check
nohup goreman start > /dev/null 2>&1 &
echo "Goreman in control"
if [[ "X`ps -ef | grep [l]fs-server-go`" == "X" ]] ;then
	goreman run start lfs-server-go
	echo "Started"
else
	echo "started"
fi
