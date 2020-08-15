package test

import (
	"sync"
	"testing"

	"github.com/jursonmo/goring"
)

func BenchmarkInterfaceChan(b *testing.B) {
	ch := make(chan interface{}, 4096)

	wg := sync.WaitGroup{}
	wg.Add(1)
	b.ResetTimer()
	go func() {
		for {
			a, ok := <-ch
			_ = a
			if !ok {
				break
			}
		}
		wg.Done()
	}()
	for i := 0; i < b.N; i++ {
		ch <- i
	}
	close(ch) //make recvicer quit
	wg.Wait()
}

/*
go test -bench=BenchmarkInterfaceChan -benchmem ring_test.go
cpu.CacheLinePadSize:64, Sizeof(ringAttr{}):12
goos: darwin
goarch: amd64
BenchmarkInterfaceChan-4   	20000000	       100 ns/op	       8 B/op	       1 allocs/op
PASS
ok  	command-line-arguments	2.132s
*/

func BenchmarkInterfaceRing(b *testing.B) {
	ring, err := goring.NewRing("ring_1", 4096, true, true)
	if err != nil {
		return
	}
	num := 32
	objs1 := make([]interface{}, num)
	objs2 := make([]interface{}, num)
	for i := 0; i < num; i++ {
		objs2[i] = i
	}

	dequeueCount := 0
	enqueueCount := 0
	final := (b.N + num - 1) / num

	wg := sync.WaitGroup{}
	wg.Add(1)
	b.ResetTimer()
	go func() {
		for {
			v := ring.Dequeue(objs1, num)
			if len(v) == 0 {
				continue
			}
			if len(v) != num {
				panic("Dequeue: len(v) != num")
			}
			dequeueCount++
			if dequeueCount == final {
				break
			}
		}
		wg.Done()
	}()

	for i := 0; i < b.N; i++ {
		if i&(num-1) == 0 {
			for {
				if ring.EnqueueBulk(objs2, num) == num {
					enqueueCount++
					break
				}
			}
		}
	}
	wg.Wait()
	if enqueueCount != dequeueCount {
		b.Fatalf("enqueueCount:%d != dequeueCount:%d\n", enqueueCount, dequeueCount)
	}
}

/*

go test -bench=BenchmarkInterfaceRing -benchmem ring_test.go
cpu.CacheLinePadSize:64, Sizeof(ringAttr{}):12
goos: darwin
goarch: amd64
BenchmarkInterfaceRing-4   	200000000	         6.07 ns/op	       1 B/op	       0 allocs/op
PASS
ok  	command-line-arguments	1.831s
*/
