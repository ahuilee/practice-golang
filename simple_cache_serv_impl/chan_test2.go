package main

import (
	"fmt"
	"time"
)

type A struct {
	num int
}


func asynRun(que chan A) {


	for i:=0; i<100;i++ {
		fmt.Println("chan put", i)
		que <- A{num: i}

		time.Sleep(1000*time.Millisecond)
	}
	
	fmt.Println("chan done")

}


func main() {

	que := make(chan A)

	go asynRun(que)

	for i:=0; i<1000; i++ {

		n := <-que
		fmt.Println("get", n)

	}

}
