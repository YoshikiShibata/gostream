#!/bin/bash

source ~/bin/switch_go2.bash

gofmt -w *.go2
if [ $? != 0 ]; then
	exit 1
fi


for maxprocs in 1 4 8 16 32 64 128 256;
do
    export GOMAXPROCS=$maxprocs
	echo "GOMAXPROCS=$GOMAXPROCS"
    go tool go2go test 
	if [ $? != 0 ]; then
		exit 1
	fi
	echo ""
done;

rm *.go
