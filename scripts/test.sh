#!/bin/bash

export GO_ENV=test
cd $(dirname $0)/../
if [[ ! "`ps -ef | grep '[t]est_ldap'`" ]] ; then
  echo "Starting LDAP server"
  cd test_ldap
  go build
  ./test_ldap >> ../log/ldap_test.log 2>&1 &
  ldap_pid=$!
  cd -
fi

go test
resp=$?

[[ "x$ldap_pid" != "x" ]] && kill $ldap_pid

exit $resp
