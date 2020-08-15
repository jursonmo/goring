package test

/*
test all:
go test -bench . -benchmem -cpu=1,2,4 ch_test.go //分别用1,2,4 cpu来测试所有benchmark
go test -bench=BenchmarkInterfaceChan -benchmem ch_test.go //正则匹配

only test BenchmarkInterfaceChan1:
go test -bench=BenchmarkInterfaceChan1 -benchmem ch_test.go

only test BenchmarkInterfaceChan2:
go test -bench=BenchmarkInterfaceChan2 -benchmem ch_test.go
*/

import (
	"sync"
	"testing"
)

func BenchmarkInterfaceChan1(b *testing.B) {
	ch := make(chan interface{}, 4096)
	recvSum := 0
	b.ResetTimer()
	go func() {
		for {
			a, ok := <-ch
			_ = a
			if !ok {
				break
			}
			recvSum++
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- i
	}
	close(ch) //make recvicer quit
	b.Logf("recvSum:%d, b.N:%d\n", recvSum, b.N)
}

/*
go test -bench=BenchmarkInterfaceChan1 -benchmem ch_test.go
goos: darwin
goarch: amd64
BenchmarkInterfaceChan1-4   	20000000	       100 ns/op	       8 B/op	       1 allocs/op
--- BENCH: BenchmarkInterfaceChan1-4
    ch_test.go:26: recvSum:0, b.N:1
    ch_test.go:26: recvSum:0, b.N:100
    ch_test.go:26: recvSum:8690, b.N:10000
    ch_test.go:26: recvSum:999689, b.N:1000000
    ch_test.go:26: recvSum:19999718, b.N:20000000
PASS
ok  	command-line-arguments	2.122s
*/

func BenchmarkInterfaceChan2(b *testing.B) {
	ch := make(chan interface{}, 4096)
	recvSum := 0
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
			recvSum++
		}
		wg.Done()
	}()
	for i := 0; i < b.N; i++ {
		ch <- i
	}
	close(ch) //make recvicer quit
	wg.Wait()
	b.Logf("recvSum:%d, b.N:%d\n", recvSum, b.N)
}

/*
go test -bench=BenchmarkInterfaceChan2 -benchmem ch_test.go
goos: darwin
goarch: amd64
BenchmarkInterfaceChan2-4   	20000000	       100 ns/op	       8 B/op	       1 allocs/op
--- BENCH: BenchmarkInterfaceChan2-4
    ch_test.go:66: recvSum:1, b.N:1
    ch_test.go:66: recvSum:100, b.N:100
    ch_test.go:66: recvSum:10000, b.N:10000
    ch_test.go:66: recvSum:1000000, b.N:1000000
    ch_test.go:66: recvSum:20000000, b.N:20000000
PASS
ok  	command-line-arguments	2.119s
*/
