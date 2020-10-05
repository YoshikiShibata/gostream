// Copyright © 2020 Yoshiki Shibata. All rights reserved.

package gostream

import (
	"bufio"
	"os"
)

func FileLines(filepath string) (Stream[string], error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	input := bufio.NewScanner(f)
	input.Split(bufio.ScanLines)

	nextReq := make(chan struct{})
	nextData := make(chan orderedData[string])
	prevDone := make(chan struct{})

	go func() {
		i := 0
		for range nextReq {
			if !input.Scan() {
				close(nextData)
				close(prevDone)
				f.Close()
				go func() {
					for range nextReq {
					}
				}()
				return
			}
			nextData <- orderedData[string]{
				order: uint64(i),
				data:  input.Text(),
			}
		}
		close(nextData)
		close(prevDone)
		f.Close()
	}()

	return &genericStream[string]{
		parallelCount: 1,
		prevDone:      prevDone,
		nextReq:       nextReq,
		nextData:      nextData,
	}, nil
}
