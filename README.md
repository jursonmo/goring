#### goring

implement from dpdk ring

#### need to known:
1. atomic cas,
2. [What’s false sharing and how to solve it](https://medium.com/@genchilu/whats-false-sharing-and-how-to-solve-it-using-golang-as-example-ef978a305e10)
3. [Go memory model](http://golang.org/ref/mem).


#### dpdk ring 
(类似disruptor，无锁,通过原子操作compare and swap:cas 来决绝竞争)

为了实现高效的队列，避免锁的竞争，往往需要实现一个环形队列 
##### 1. 如果模型是单生产者/单消费者的话，
只需要一个head 和一个tail 就行了，head 指向生产者存放数据的槽位，tail 指向消费者提取数据的槽位。
生产者存放数据完毕后，才能移动head，不然消费者可能拿不到数据。也就是head 控制消费者哪些数据可以取。

##### 2. 但是如果是多生产者、单消费者 模型下，
多个生产者的情况，会遇到“如何防止多个线程重复写同一个元素”的问题。
那么需要两个变量来指定生产者存放数据的槽位， 所以需要另一个变量head1，用于多生产之间抢哪些槽位可以存放数据, 原来的head变量还是用于控制消费者哪些数据可以取。

多生产者先抢到head1的位置后，才可以把数据存入head1的槽位，通过 cas 来抢位置。生产者存放数据后，需要判断ring.head是否是oldhead,  是才移动ring.head到head1 位置，然后消费者就可以取了。

```
   0      1       2     3      4
+------+------+------+------+------+---
|      |      |      |      |      |
+------+------+------+------+------+----
               head1 
ring:          head 

producer1:    oldhead  head1
producer2:            oldhead  head1

oldhead 是各个线程的自己临时变量， head 和 head1 表示 ring 的两个变量。
ring的变量head1，用于多生产之间抢哪些槽位可以存放数据, 原来的head变量还是用于控制消费者哪些数据可以取(即生产者已经存放好的数据)。
```
如图，一开始ring的head、head1都指向2号槽位，producer1 线程和producer2 线程 同时生产数据，producer1 抢先cas 拿到3号位置，并把ring.head1设置成3，producer1 的临时变量oldhead 设置为2 ，producer2只能再次cas，拿到4号位置，并且把ring.head1 设置为4，producer2 的临时变量oldhead 设置为3，这时如果producer2率先完成数据的存放工作，也不能把ring.head 设置为4 ，因为producer1 可能还没有把数据存放工作做完，如果producer2把ring.head 设置4，那么消费者就以为3、4号位的数据都可以提取，就可能读到没有被producer1存放完成的3号位数据。所以producer2只能等producer1存放完成的3号位数据，并且等producer1 把ring.head 改成 3 后，producer2才能把ring.head 改成 4.

##### 3. 同理如果是多消费者，也需要两个变量来控制，原理跟多生产者一样。