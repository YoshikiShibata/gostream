#!/bin/bash

source ~/bin/switch_go2.bash

gofmt -w *.go2
if [ $? != 0 ]; then
	exit 1
fi
