#!/bin/bash
set -eu
set -o pipefail

export GO_ENV=test
ldap_pid=""
base=$(dirname $0)/../
cd $base
godep > /dev/null 2>&1 || go get github.com/tools/godep
if [[ ! "`ps -ef | grep '[t]est_ldap_server'`" ]] ; then
  echo "Starting LDAP server"
  cd test_ldap_server
  godep go build ./...
  ./test_ldap_server >> ../log/ldap_test.log 2>&1 &
  ldap_pid=$!
  cd -
fi
# install godep dependencies
godep restore
# space delimiter
prereqs=("cassandra", "mysqld")
for p in ${prereqs[@]}; do
  lf="`echo [$(echo $p | cut -b1)]${p:1}`"
  if [[ "x`ps -ef |grep $lf`" == "x" ]];then
   echo "$p does not look to be running, tests will fail"
  fi
done
go fmt ./...
mkdir -p $base/coverage
godep go test -coverprofile=$base/coverage/cover.out -covermode=count $*
godep go tool cover -html=$base/coverage/cover.out
resp=$?

if [[ "x${ldap_pid}" != "x" ]]; then
 echo "Stopping LDAP server"
 kill $ldap_pid
fi

exit $resp
