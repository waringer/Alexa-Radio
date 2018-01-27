#!/bin/sh

p=`pwd`

export GOPATH=$p/lib/
export GIT_SSL_NO_VERIFY=1

echo "get alexa skillserver lib"
go get -u -d github.com/waringer/go-alexa/skillserver
go get -u -d github.com/go-sql-driver/mysql
#go get -u -d github.com/fhs/gompd/mpd

echo "build radio"
go build -ldflags "-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD`" radio.go

