package qlock

/* mainly used by calvin */
import (
	"container/list"
	"sync"
	"fmt"
	"t_log"
)

type lock struct {
	Txn_id int
	Is_write bool
}

type QueueLock struct {
	queue *(list.List) // save *lock
	rwlock *sync.RWMutex
}

func NewQueueLock() *QueueLock {
	l := list.New()
	return &QueueLock{l, &(sync.RWMutex{})}
}

/*
the add lock sequence should be the same order with run sequence
*/
func (q *QueueLock) AddLock(txn_id int, is_write bool) {
	l := lock{txn_id, is_write}
	q.rwlock.Lock()
	defer q.rwlock.Unlock()
	q.queue.PushBack(&l)
}


/*
for test
*/
func (q *QueueLock) LockListString() string {
	q.rwlock.RLock()
	defer q.rwlock.RUnlock()
	var res string
	for e := q.queue.Front(); e != nil; e = e.Next() {
		// do something with e.Value
		res = res + fmt.Sprintf("%v ", e.Value)
	}
	return res
}

/*
unsync method
*/
func (q *QueueLock) ReadTicket(txn_id int) bool {
	// read is also need the write lock since need to update the queue
	q.rwlock.Lock()
	defer q.rwlock.Unlock()
	for e := q.queue.Front(); e != nil; e = e.Next() {
		cur := e.Value.(*lock)
		if cur.Is_write == false && cur.Txn_id == txn_id { // get the lock
			// delete the lock
			q.queue.Remove(e)
			return true
		} else if cur.Is_write == true { // until write there is no read so no get lock
			return false
		}
	}
	return false
}


func (q *QueueLock) WriteTicket(txn_id int) bool {
	// read is also need the write lock since need to update the queue
	q.rwlock.Lock()
	defer q.rwlock.Unlock()
	e := q.queue.Front()
	if e == nil {
		t_log.Log(t_log.PANIC, "error point in queue lock\n")
		return false
	}
	cur := e.Value.(*lock)
	if cur.Is_write == true && cur.Txn_id == txn_id { // get the lock
		q.queue.Remove(e)
		return true
	} else {
		return false
	}

}

/*
onece you can write and the op behind all can exec (and can validate ok)
but once you can read the op behind may not can write !!!
can write or not see the first ! if the first is my operation then behind all can do
*/
func (q *QueueLock) WriteValidate(txn_id int) bool {

	q.rwlock.Lock()
	defer q.rwlock.Unlock()
	e := q.queue.Front()
	if e == nil {
		t_log.Log(t_log.PANIC, "error point in queue lock\n")
		return false
	}
	cur := e.Value.(*lock)
	if cur.Txn_id == txn_id { // get the lock
		return true
	} else {
		return false
	}
}

/*
once you can read and the op behind to this write also can read
*/
func (q *QueueLock) ReadValidate(txn_id int) bool {
	// read is also need the write lock since need to update the queue
	q.rwlock.Lock()
	defer q.rwlock.Unlock()
	for e := q.queue.Front(); e != nil; e = e.Next() {
		cur := e.Value.(*lock)

		if cur.Txn_id == txn_id { // until write there is no read so no get lock
			return true
		} else if cur.Is_write == false {
			continue
		} else {
			break
		}
	}
	return false
}