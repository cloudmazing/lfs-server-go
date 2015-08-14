#!/bin/bash
set -eu
set -o pipefail

export GO_ENV=test
ldap_pid=""
base=$(dirname $0)/../
cd $base
if [[ ! "`ps -ef | grep '[t]est_ldap_server'`" ]] ; then
  echo "Starting LDAP server"
  cd test_ldap_server
  go build
  ./test_ldap_server >> ../log/ldap_test.log 2>&1 &
  ldap_pid=$!
  cd -
fi

prereqs="cassandra redis"
for p in $prereqs; do
  lf="`echo [$(echo $p | cut -b1)]${p:1}`"
  if [[ "x`ps -ef |grep $lf`" == "x" ]];then
   echo "$p does not look to be running, tests will fail"
  fi
done
go fmt
go test -coverprofile=$base/coverage/cover.out -covermode=count
go tool cover -html=$base/coverage/cover.out
resp=$?

if [[ "x${ldap_pid}" != "x" ]]; then
 echo "Stopping LDAP server"
 kill $ldap_pid
fi

exit $resp
