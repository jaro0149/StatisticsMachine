#!/bin/sh
absolutePath=`readlink -f ../`
export GOOS=linux
export GOARCH=arm
export GOARM=7
export CGO_ENABLED=1
export CC=arm-linux-gnueabi-gcc
export GOPATH="$GOPATH:$absolutePath"
/usr/local/go/bin/go build -o StatisticsMachineApp
