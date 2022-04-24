package rd

import (
	"sync"
	// "container/list"
	// "t_txn"
)




// by other txn
// R Then W R Then R is allowed
// W Then R W Then W? is not allowed

type Record struct {
	min_wid int // init with -1 means no be writed yet
	rwlock *(sync.RWMutex)
}

func (r *Record) Get_min_wid() int {
	r.rwlock.RLock()
	defer r.rwlock.RUnlock()
	return (r.min_wid)
}

func NewRecord() *Record {
	return &(Record{-1, &(sync.RWMutex{})})
}

/*
Write/Read in first phase (execution phase)
this may be interupt when exec
*/


func (r *Record) Write(txn_id int) bool {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()
	if r.min_wid == -1 { // not write this record yet
		r.min_wid = txn_id
		return true
	}

	if txn_id > r.min_wid { // write after operation will all be abort
		return false
	} else { // write before operation all be ok
		r.min_wid = txn_id
		return true
	}


}

func (r *Record) Read(txn_id int) bool {
	r.rwlock.RLock()
	defer r.rwlock.RUnlock()
	// add to w_txns in order
	if r.min_wid == -1 { // not write this record yet
		return true
	}

	if txn_id > r.min_wid { // write after operation will all be abort
		return false
	} else { // write before operation all be ok
		return true
	}
}


func (r *Record) Validate(txn_id int) bool {

	if txn_id > r.min_wid {
		return false
	} else {
		return true
	}
}