package test

//go test -bench . -benchmem -cpu=1,2,4
import (
	"goring"
	"testing"
)

func BenchmarkInterfaceChan(b *testing.B) {
	ch := make(chan interface{}, 4096)
	b.ResetTimer()
	go func() {
		for {
			<-ch
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- i
	}
}

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

	b.ResetTimer()
	go func() {
		for {
			v := ring.Dequeue(objs1, num)
			if len(v) > 0 && len(v) != num {
				panic("Dequeue: len(v) != num")
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		// for {
		// 	if ring.Enqueue(i) == 1 {
		// 		break
		// 	}
		// }

		if i&(num-1) == 0 {
			for {
				if ring.EnqueueBulk(objs2, num) == num {
					break
				}
			}
		}
	}
}

/*
obcs-MBP:test obc$ go test -bench . -cpu=2
goos: darwin
goarch: amd64
pkg: goring/test
BenchmarkInterfaceChan-2   	11907808	        95.8 ns/op
BenchmarkInterfaceRing-2   	 1418254	      1541 ns/op
PASS
ok  	goring/test	6.363s
obcs-MBP:test obc$ go test -bench . -cpu=4
goos: darwin
goarch: amd64
pkg: goring/test
BenchmarkInterfaceChan-4   	11603294	        99.6 ns/op
BenchmarkInterfaceRing-4   	 5736200	       440 ns/op
PASS
ok  	goring/test	4.010s

ring 为啥效果这么差.
1. enqueu/dequeue 都是是1个元素
2. dequeue 有分配内存的行为

========改成128个元素批量enqueue、dequeue===============
obcs-MBP:test obc$ go test -bench . -cpu=4
cpu.CacheLinePadSize: 64
goos: darwin
goarch: amd64
pkg: goring/test
BenchmarkInterfaceChan-4   	11289121	        96.2 ns/op
BenchmarkInterfaceRing-4   	12105906	       175 ns/op
PASS
ok  	goring/test	3.426s

========改成128个元素批量enqueue、dequeue, 再把dequeue 有分配内存的行为 去掉===============

obcs-MBP:test obc$ go test -bench . -cpu=4
cpu.CacheLinePadSize: 64
goos: darwin
goarch: amd64
pkg: goring/test
BenchmarkInterfaceChan-4   	11841489	        96.6 ns/op
BenchmarkInterfaceRing-4   	13758882	        88.1 ns/op
PASS
ok  	goring/test	2.567s

性能才相差不多了。

按道理应该ring 的性能应该比较好才对， 而且ring 可以批量入出队
原因是benchmark 的test 方式跟我想象的不一样。直接用example 的方式测试，就可以看出ring 批量入出队的效率比较高了, 单一出入队，还是channel 高。
*/
