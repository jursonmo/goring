package goring

/*
implement: from dpdk ring
atomic cas,
*/

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/jursonmo/goring/internel/cpu" //copy from golang runtime
)

type pad struct {
	x [cpu.CacheLinePadSize]byte
}

type ringheadtail struct {
	head uint32 /**< Prod/consumer head. */
	//pad   [cpu.CacheLinePadSize - unsafe.Sizeof(uint32(0))%cpu.CacheLinePadSize]byte
	tail   uint32 /**< Prod/consumer tail. */
	single bool   /**< True if single prod/cons */
}

type ringAttr struct {
	size uint32 // PowerOfTwo
	mask uint32 // for &
	cap  uint32 //
}

type Ring struct {
	ringAttr
	//pad0 pad
	_    [cpu.CacheLinePadSize - unsafe.Sizeof(ringAttr{})%cpu.CacheLinePadSize]byte
	prod ringheadtail
	_    [cpu.CacheLinePadSize - unsafe.Sizeof(ringheadtail{})%cpu.CacheLinePadSize]byte
	cons ringheadtail
	_    [cpu.CacheLinePadSize - unsafe.Sizeof(ringheadtail{})%cpu.CacheLinePadSize]byte

	cache []interface{}

	// 常用的字段放结构体前面, 不常用的字段放在结构体的后面
	name string
}

type RingQueueBehavior byte

const (
	ISENQUEUE bool = true

	RING_QUEUE_FIXED    RingQueueBehavior = 0 /* Enq/Deq a fixed number of items from a ring */
	RING_QUEUE_VARIABLE RingQueueBehavior = 1 /* Enq/Deq as many items as possible from ring */
)

func init() {
	fmt.Printf("goring env: GOMAXPROCS:%d, cpu.CacheLinePadSize:%d, Sizeof(ringAttr{}):%d\n", runtime.GOMAXPROCS(0),
		cpu.CacheLinePadSize, unsafe.Sizeof(ringAttr{}))
}

func IsPowerOfTwo(n int) bool {
	return n&(n-1) == 0
}

func NewRing(name string, n int, isSP, isSC bool) (*Ring, error) {
	if !IsPowerOfTwo(n) {
		return nil, fmt.Errorf("n:%d is not PowerOfTwo", n)
	}
	nn := uint32(n)
	r := &Ring{}
	r.cap = nn
	r.size = nn
	r.mask = nn - 1
	r.prod.single = isSP
	r.cons.single = isSC
	r.cache = make([]interface{}, n)
	r.name = name
	return r, nil
}

func (r *Ring) String() string {
	return fmt.Sprintf("name:%s, size:%d, prod:[%s], cons:[%s]", r.name, r.size, &r.prod, &r.cons)
}

func (ht *ringheadtail) String() string {
	return fmt.Sprintf("h-%d, t-%d, s-%v", ht.head, ht.tail, ht.single)
}

func (r *Ring) EnqueueBulk(obj interface{}, n int) int {
	return int(r.enqueue(obj, uint32(n), r.prod.single, RING_QUEUE_FIXED))
}

func (r *Ring) EnqueueOne(obj interface{}) int {
	return int(r.enqueue(obj, 1, r.prod.single, RING_QUEUE_FIXED))
}

func (r *Ring) Enqueue(obj interface{}) int {
	return int(r.enqueue(obj, 1, r.prod.single, RING_QUEUE_FIXED))
}

func (r *Ring) enqueue(obj interface{}, n uint32, isSP bool, bbehavior RingQueueBehavior) uint32 {
	var prodHead, prodNext uint32
	var objs []interface{}
	var isbulk bool
	if n > 1 {
		objs, isbulk = obj.([]interface{})
		if !isbulk {
			return 0
		}
	}

	n = r.ringMoveProdHead(&prodHead, &prodNext, n, isSP, bbehavior)
	if n == 0 {
		return 0
	}

	//copy to ring
	if isbulk {
		start := prodHead & r.mask
		if start+n < r.size {
			copyn := copy(r.cache[start:], objs[:n])
			if copyn != int(n) {
				panic("copyn != int(n)")
			}
		} else {
			copy1 := copy(r.cache[start:r.size], objs)
			copy2 := copy(r.cache[:start+n-r.size], objs[copy1:]) //objs[r.size-start:]
			if copy1+copy2 != int(n) {
				panic("copy1+copy2 != int(n) ")
			}
		}
	} else {
		r.cache[prodHead&r.mask] = obj
	}

	updateTail(&r.prod, prodHead, prodNext, isSP, ISENQUEUE)
	return n
}

func (r *Ring) ringMoveProdHead(oldHead, newHead *uint32, n uint32, isSP bool, behavior RingQueueBehavior) uint32 {
	max := n
	cap := r.cap
	free := uint32(0)
	succ := false
	for {
		n = max //reset to initial value every loop

		*oldHead = atomic.LoadUint32(&r.prod.head)
		free = cap + atomic.LoadUint32(&r.cons.tail) - *oldHead
		if free < n {
			if behavior == RING_QUEUE_FIXED {
				n = 0
			} else {
				n = free
			}

		}
		if n == 0 {
			return 0
		}
		*newHead = *oldHead + n
		if isSP {
			r.prod.head = *newHead
			succ = true
		} else {
			succ = atomic.CompareAndSwapUint32(&r.prod.head, *oldHead, *newHead)
		}
		if succ {
			break
		}
	}
	return n
}

func (r *Ring) Dequeue(obj interface{}, n int) []interface{} {
	return r.dequeue(obj, uint32(n), r.cons.single, RING_QUEUE_FIXED)
}

func (r *Ring) DequeueVariable(obj interface{}, n int) []interface{} {
	return r.dequeue(obj, uint32(n), r.cons.single, RING_QUEUE_VARIABLE)
}

//var objs [128]interface{}

func (r *Ring) dequeue(obj interface{}, n uint32, isSC bool,
	behavior RingQueueBehavior) (objs []interface{}) {
	var consHead, consNext uint32
	var isbulk bool

	//if n > 1 {
	objs, isbulk = obj.([]interface{}) //must be []interface{}
	if !isbulk {
		return
	}
	//}
	n = r.ringMoveConsHead(&consHead, &consNext, n, isSC, behavior)
	if n == 0 {
		objs = objs[:0]
		return
	}

	// get ring data to objs
	//objs = make([]interface{}, n) // here make a low performent
	start := consHead & r.mask
	if start+n < r.size {
		copy(objs[:], r.cache[start:start+n])
	} else {
		copyn := copy(objs[:], r.cache[start:r.size])
		copy(objs[copyn:], r.cache[:start+n-r.size])
	}

	updateTail(&r.cons, consHead, consNext, isSC, false)
	objs = objs[:n]
	return
}

func (r *Ring) ringMoveConsHead(oldHead, newHead *uint32, n uint32,
	isSC bool, behavior RingQueueBehavior) uint32 {
	max := n
	entries := uint32(0)
	succ := false
	for {
		n = max
		*oldHead = atomic.LoadUint32(&r.cons.head)
		entries = atomic.LoadUint32(&r.prod.tail) - *oldHead
		if entries < n {
			if behavior == RING_QUEUE_FIXED {
				n = 0
			} else {
				n = entries
			}
		}
		if n == 0 {
			return 0
		}
		*newHead = *oldHead + n
		if isSC {
			atomic.StoreUint32(&r.cons.head, *newHead)
			succ = true
		} else {
			succ = atomic.CompareAndSwapUint32(&r.cons.head, *oldHead, *newHead)
		}
		if succ {
			break
		}
	}
	return n
}

func updateTail(ht *ringheadtail, oldVal, newVal uint32, isSingle bool, isEnqueue bool) {
	if !isSingle {
		for {
			//多个生产者, 需要等前面的生产者按次序更新完tail值后，当前的生产者才能更新tail。
			//这个次序就是多个生产者同时cas去抢ringheadtail.head 的次序
			//同理，多个消费者也一样
			if atomic.LoadUint32(&ht.tail) != oldVal {
				runtime.Gosched()
			} else {
				break
			}
		}
	}
	atomic.StoreUint32(&ht.tail, newVal)
}
