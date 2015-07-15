#!/bin/bash

set -eu
set -o pipefail

cd $(dirname $0)/../
#LFS_LISTEN=${LFS_LISTEN:-tcp://:9999}
#LFS_HOST=${LFS_HOST:-127.0.0.1:9999}
#LFS_CONTENTPATH=${LFS_CONTENTPATH:-lfs-content}
#LFS_ADMINUSER=${LFS_ADMINUSER:-admin}
#LFS_ADMINPASS=${LFS_ADMINPASS:-admin}
#LFS_CERT=${LFS_CERT:-mine.crt}
#LFS_KEY=${LFS_KEY:-mine.key}
#LFS_SCHEME=${LFS_SCHEME:-https}
#LFS_REDIS_URL=${LFS_REDIS_URL:-localhost:6379/0}
#
#export LFS_LISTEN LFS_HOST LFS_CONTENTPATH LFS_ADMINUSER LFS_ADMINPASS LFS_CERT LFS_KEY LFS_SCHEME

[[ ! -f "lfs-server-go" ]] && go build
./lfs-server-go
