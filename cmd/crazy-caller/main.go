package main

import "fmt"

func main() {
	c := make(chan int)

	go func() {

		defer fmt.Println("defer go routine")

		fmt.Println("go routine start")

		c <- 666
	}()

	v := <-c
	fmt.Println("get value from go routine:", v)

}
