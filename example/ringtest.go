package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/jursonmo/goring"
)

func runFuncName() string {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}

var (
	queueSize = 4096
	num       = 32 //goring enqueue/dequeue batch
	dataCount = 2000 * num * num
)

func main() {
	fmt.Printf("GOMAXPROCS:%d, queueSize:%d, dataCount:%d, batch num:%d\n", runtime.GOMAXPROCS(0), queueSize, dataCount, num)
	//single producer, single consumer
	channelSPSC()
	goringSPSC()

	//single producer, two consumer
	channelSPMC()
	goringSPMC()

	//TODO: two producer, single consumer

}

func channelSPSC() {
	wg := sync.WaitGroup{}
	ch := make(chan interface{}, queueSize)

	wg.Add(1)
	start := time.Now()
	go func() {
		for i := 0; i < dataCount; i++ {
			a := <-ch
			_ = a
		}
		wg.Done()
	}()
	for i := 0; i < dataCount; i++ {
		ch <- i
	}
	wg.Wait()
	fmt.Printf("%s: enqueue/dequeue data entries:%d, time consuming:%v\n", runFuncName(), dataCount, time.Since(start))
}

func channelSPMC() {
	wg := sync.WaitGroup{}
	ch := make(chan interface{}, queueSize)

	wg.Add(2)
	start := time.Now()
	go func() {
		for i := 0; i < dataCount/2; i++ {
			a := <-ch
			_ = a
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < dataCount/2; i++ {
			a := <-ch
			_ = a
		}
		wg.Done()
	}()

	for i := 0; i < dataCount; i++ {
		ch <- i
	}
	wg.Wait()
	fmt.Printf("%s: enqueue/dequeue data entries:%d, time consuming:%v\n", runFuncName(), dataCount, time.Since(start))
}

func goringSPSC() {
	var wg sync.WaitGroup
	var v []interface{}
	mask := num - 1

	objs1 := make([]interface{}, num)
	enqueuObjs := make([]interface{}, num)
	for i := 0; i < num; i++ {
		enqueuObjs[i] = i
	}

	ring, err := goring.NewRing("ring_1", queueSize, true, true)
	if err != nil {
		return
	}

	wg.Add(1)
	start := time.Now()
	go func() {
		for i := 0; i < dataCount; i++ {
			if i&mask == 0 {
				//loop util dequeue succe
				for {
					v = ring.Dequeue(objs1, num)
					if len(v) == 0 {
						continue
					}
					if len(v) != num {
						panic("Dequeue: len(v) != num")
					}
					break
				}
			}
		}
		wg.Done()
	}()

	for i := 0; i < dataCount; i++ {
		if i&mask == 0 {
			//loop util enqueue succe
			for {
				if ring.EnqueueBulk(enqueuObjs, num) == num {
					break
				}
			}
		}
	}

	wg.Wait()
	fmt.Printf("%s: enqueue/dequeue data entries:%d, time consuming:%v\n", runFuncName(), dataCount, time.Since(start))
}

func goringSPMC() {
	var wg sync.WaitGroup
	var v1 []interface{}
	var v2 []interface{}
	mask := num - 1
	objs1 := make([]interface{}, num)
	objs2 := make([]interface{}, num)

	enqueuObjs := make([]interface{}, num)
	for i := 0; i < num; i++ {
		enqueuObjs[i] = i
	}

	ring, err := goring.NewRing("ring_2", queueSize, true, false) //not singel consumer
	if err != nil {
		return
	}

	wg.Add(2)
	start := time.Now()
	go func() {
		for i := 0; i < dataCount/2; i++ {
			if i&mask == 0 {
				//loop util dequeue succe
				for {
					v1 = ring.Dequeue(objs1, num)
					if len(v1) == 0 {
						continue
					}
					if len(v1) != num {
						panic("Dequeue: len(v) != num")
					}
					break
				}
			}
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < dataCount/2; i++ {
			if i&mask == 0 {
				//loop util dequeue succe
				for {
					v2 = ring.Dequeue(objs2, num)
					if len(v2) == 0 {
						continue
					}
					if len(v2) != num {
						panic("Dequeue: len(v) != num")
					}
					break
				}
			}
		}
		wg.Done()
	}()

	for i := 0; i < dataCount; i++ {
		if i&mask == 0 {
			//loop util enqueue succe
			for {
				if ring.EnqueueBulk(enqueuObjs, num) == num {
					break
				}
			}
		}
	}

	wg.Wait()
	fmt.Printf("%s: enqueue/dequeue data entries:%d, time consuming:%v\n", runFuncName(), dataCount, time.Since(start))
}

/*
output:
./ringtest
cpu.CacheLinePadSize: 64
main.channelSPSC: enqueue/dequeue data entries:2048000, time consuming:213.764252ms
main.goringSPSC: enqueue/dequeue data entries:2048000, time consuming:11.205793ms
main.channelSPMC: enqueue/dequeue data entries:2048000, time consuming:255.494695ms
main.goringSPMC: enqueue/dequeue data entries:2048000, time consuming:14.466823ms

*/
