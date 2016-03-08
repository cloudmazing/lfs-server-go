#!/bin/bash

set -eu
set -o pipefail

export logfile=log/lfs-server-go.log

cd $(dirname $0)/../

GO_ENV=${GO_ENV:-dev}

export GO_ENV
godep > /dev/null 2>&1 || go get github.com/tools/godep
godep restore

if [[ ("${GO_ENV}" == "test") && (! "`ps -ef | grep '[t]est_ldap'`") ]] ; then
  ./test_ldap_server/test_ldap_server > log/ldap.log 2>&1 &
fi

test ! -f ./lfs-server-go && godep go build

goreman > /dev/null 2>&1 || go get github.com/mattn/goreman
goreman check

echo "Using goreman to start lfs-server-go"

# set the config, if not set
: ${LFS_SERVER_GO_CONFIG:=config.ini}

if [[ ! -f "${LFS_SERVER_GO_CONFIG}" ]];then
  echo "LFS_SERVER_GO_CONFIG is not set, copying in example"
  # copy over a config if one does not exist
  cp ./config.ini.example config.ini
fi

if [[ "X`ps -ef | grep [g]oreman | grep -v grep`" == "X" ]] ;then
	nohup goreman start >> ${logfile} 2>&1 &
fi

if [[ "X`ps -ef | grep [l]fs-server-go | grep -v grep`" == "X" ]] ;then
	goreman run start lfs-server-go >> ${logfile} 2>&1 &
	echo "Started"
else
	echo "Already running"
fi
