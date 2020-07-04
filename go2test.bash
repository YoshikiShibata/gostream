#!/bin/bash

source ~/bin/switch_go2.bash

for go2 in generic.go2 map.go2 slice.go2 stream_test.go2; 
do
	echo "formatting $go2 ..."
	gofmt -w $go2
	if [ $? != 0 ]; then
		exit 1
	fi
done

go tool go2go test
