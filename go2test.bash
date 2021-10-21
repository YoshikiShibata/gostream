#!/bin/bash


gofmt -w *.go
if [ $? != 0 ]; then
	exit 1
fi


for maxprocs in 1 4 8 16 32 64 128 256;
do
    export GOMAXPROCS=$maxprocs
	echo "GOMAXPROCS=$GOMAXPROCS"
    go test -v -gcflags=all=-G=3
	if [ $? != 0 ]; then
		exit 1
	fi
	echo ""
done;

rm *.go
