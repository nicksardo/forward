package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	for x := 0; x < 10; x++ {
		fmt.Printf("%d", x)
		time.Sleep(time.Second)

		if x == 5 {
			log.Panic("Pretend to hit a fatal error")
		}
	}
	fmt.Println()
}
