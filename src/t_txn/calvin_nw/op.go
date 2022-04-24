package calvin_nw

import (
	"t_txn"
	"t_index"
	"t_txn/calvin_nw/rd"
	"fmt"
	"sync"
	"t_log"
)



type DebugData struct {
	keys *(sync.Map)
}

func NewDebugData() *DebugData {
	return &(DebugData{&(sync.Map{})})
}

type Calvin struct {
	// batch_size configure by user
	index *(t_index.Mmap)
}

func (calvin *Calvin) PreparationCost(thread_cnt int, opss [](*t_txn.OPS)) int {
	sum_c := 0
	for i := 0; i < len(opss); i ++ {
		sum_c = sum_c + opss[i].Len()
	}
	return sum_c / thread_cnt
}

// except aria, other no need reset
func (calvin *Calvin) Reset() {
}


func New(mmap_c int) *Calvin {
	index := t_index.NewMmap(mmap_c)
	return &Calvin{index}
}

type TXN struct {
	txn_id int
	base *Calvin
	opv *([](*t_txn.OP))
}

func quickGetOrInsert(index *(t_index.Mmap), key string) *rd.Record {
	r := index.Search(key)
	if r == nil {
		r = index.GetOrInsert(key, rd.NewRecord())
	}
	return r.(*rd.Record)
}

/*
calvin need to know whole seq 
(it should know the read write sequence of the same record and can not be inorder)
*/
func (calvin *Calvin) Prios(ops *(t_txn.OPS)) (opv *([](*t_txn.OP))) {
	return ops.ReadWriteSeq()
}

func (calvin *Calvin) NewTXN(txn_id int, p_ops *t_txn.OPS) t_txn.TxnPtr {
	// input the lock reserve
	opv := calvin.Prios(p_ops)
	index := calvin.index
	for i := 0; i < len(*opv); i++ {
		op := (*opv)[i]
		key := op.Key
		is_w := op.Is_write
		r := quickGetOrInsert(index, key)
		r.Reserve(txn_id, is_w)
	}
	return &TXN{txn_id, calvin, opv}
}


func (calvin *Calvin) GetLockString(key string) string {
	var res string
	index := calvin.index
	r := index.Search(key)
	if r == nil {
		res = res + fmt.Sprintf("there is no key %v", key)
		return res
	} else {
		return r.(*rd.Record).LockListString()
	}
}


func (t *TXN) Write(key string) int {
	index := t.base.index
	r := index.Search(key)
	if r == nil {
		// this can not be happend
		return t_txn.RE2FIRST
	} else {
		if r.(*rd.Record).Write(t.txn_id) {
			return t_txn.NEXT
		} else {
			t_log.Log(t_log.DEBUG, "write validate failed\n")
			return t_txn.RE2FIRST
		}
	}
}

func (t *TXN) Read(key string) int {
	index := t.base.index
	r := index.Search(key)
	if r == nil {
		return t_txn.RE2FIRST
	} else {
		if r.(*rd.Record).Read(t.txn_id) {
			return t_txn.NEXT
		} else {
			t_log.Log(t_log.DEBUG, "read validate failed\n")
			return t_txn.RE2FIRST
		}
	}
}


/* do the txn */
func (t* TXN) Reset() int {
	return t_txn.NEXT	
}

func (t *TXN) Commit() int {
	index := t.base.index
	for i := 0 ; i < len(*(t.opv)); i++ {
		op := (*(t.opv))[i]
		key := op.Key
		is_w := op.Is_write
		r := index.Search(key).(*rd.Record)

		if is_w {
			if r.DoneWrite(t.txn_id) == false {
				t_log.Log(t_log.PANIC, "error point in calvin_nw op\n")
			} 
		} else {
			if r.DoneRead(t.txn_id) == false {
				t_log.Log(t_log.PANIC, "error point in calvin_nw op\n")
			} 
		}
	}
	return t_txn.NEXT
}

func (t *TXN) Init() int {
	return t_txn.NEXT
}