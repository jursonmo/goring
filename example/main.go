package main

import (
	"fmt"

	"github.com/jursonmo/goring"
)

func main() {
	ring, err := goring.NewRing("ring_1", 8, true, true)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", ring)
	n := 0
	//enqueue data
	a := make([]int, 8)
	for i := 0; i < len(a); i++ {
		a[i] = i
		n = ring.Enqueue(a[i])
		if n != 1 {
			panic("n != 1")
		}
	}

	//test dequeue and check value
	for i := 0; i < len(a); i++ {
		objs := make([]interface{}, 1)
		v := ring.Dequeue(objs, 1)
		if len(v) != 1 {
			panic("len(v) != 1")
		}
		if v[0].(int) != i {
			panic("")
		}
		if objs[0].(int) != i {
			panic("")
		}
		fmt.Println(v[0].(int))
	}

	//test empty
	v := ring.Dequeue(make([]interface{}, 1), 1)
	if len(v) > 0 {
		panic("now ring should be empty")
	}

	//=================================================
	//test: enqueue bulk data

	n = ring.EnqueueBulk(genInterface(a[:4]), len(a[:4]))
	if n != len(a[:4]) {
		fmt.Println(n, len(a[:4]))
		panic("n != len(a[:4])")
	}
	n = ring.EnqueueBulk(genInterface(a[4:]), len(a[4:]))
	if n != len(a[4:]) {
		fmt.Println(n, len(a[4:]))
		panic("n != len(a[4:])")
	}

	objs := make([]interface{}, len(a[:4]))
	v = ring.Dequeue(objs, len(objs))
	if len(v) != len(a[:4]) {
		panic("ring should dequeue expect value")
	}
	for i := 0; i < len(a[:4]); i++ {
		if v[i].(int) != a[i] {
			panic("value is not expect")
		}
	}

	//force to dequeue 5 data, but only 4 data in queue, it should fail
	v = ring.Dequeue(make([]interface{}, 5), 5)
	if len(v) > 0 {
		panic("ring should dequeue fail, there not enough data to dequeue")
	}

	//try to dequeue 5 data, but only 4 data be dequeued
	v = ring.DequeueVariable(make([]interface{}, 5), 5)
	//check v len
	if len(v) != len(a[:4]) {
		panic("ring should dequeue expect value")
	}
	//check v value
	for i := 0; i < 4; i++ {
		if v[i].(int) != a[i+4] {
			panic("value is not expect")
		}
	}

	fmt.Println("base test success")
}

func genInterface(a []int) []interface{} {
	b := make([]interface{}, len(a))
	for i := 0; i < len(a); i++ {
		b[i] = a[i]
	}
	return b
}
