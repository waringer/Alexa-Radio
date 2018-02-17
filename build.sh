#!/bin/sh

p=`pwd`

#export GOPATH=$p/lib/
export GIT_SSL_NO_VERIFY=1

echo "get libs"
go get -u -d github.com/waringer/go-alexa/skillserver
go get -u -d github.com/go-sql-driver/mysql
go get -u -d github.com/vmware/go-nfs-client/nfs
go get -u -d github.com/dhowden/tag

mkdir -p bin
cd bin

echo "build radio skill server"
go build -ldflags "-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD`" ../radio
echo "build scanner tool"
go build -ldflags "-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD`" ../scanner

cd -
