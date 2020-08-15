package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	ch := make(chan interface{}, 4096)
	num := 32
	testTime := 10000 * 100 * num
	start := time.Now()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for i := 0; i < testTime; i++ {
			a := <-ch
			if a.(int) != i {
				panic("")
			}
		}
		wg.Done()
	}()
	for i := 0; i < testTime; i++ {
		ch <- i
	}
	wg.Wait()
	fmt.Printf("enqueue/dequeue data entries:%d, time consuming:%v\n", testTime, time.Since(start))
}
