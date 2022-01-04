#!/bin/sh

mkdir -p bin

echo "build radio skill server"
cd radio
go build -ldflags "-X main.buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.githash=`git rev-parse HEAD`" -o ../bin
exit_status=$?
cd -
if [ $exit_status -eq 0 ]; then
    echo "run radio skill server"
    cd bin
    ./radio
    cd -
else
    echo "can't run skill server!"
fi
