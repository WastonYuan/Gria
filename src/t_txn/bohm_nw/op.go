package bohm_nw

import (
	"t_index"
	"t_txn/bohm_nw/rd"
	"t_log"
	"sync"
	"t_txn"
)


type DEBUG struct {
	keys *sync.Map
}

type BOHM struct {
	// batch_size configure by user
	index *(t_index.Mmap)
	Read_conflict int
}

func New(mmap_c int) *BOHM {
	index := t_index.NewMmap(mmap_c)
	return &(BOHM{index , 0})
}

func (bohm *BOHM) PreparationCost(thread_cnt int, opss [](*t_txn.OPS)) int {
	sum_c := 0
	for i := 0; i < len(opss); i ++ {
		sum_c = sum_c + opss[i].Len()
	}
	return sum_c / thread_cnt
}

// except aria, other no need reset
func (bohm *BOHM) Reset() {
}

type TXN struct {
	txn_id int
	write_map *(map[string]int) // bohm should know the write set in start
	write_version *(map[*(rd.Version)]bool) // true means being writed which will clean when revert
	base *BOHM
}

/*bohm need to know the write map*/
func (bohm *BOHM) Prios(ops *(t_txn.OPS)) *(map[string]int) {
	return ops.ReadWriteMap(true)
}

func (bohm *BOHM) NewTXN(txn_id int, ops *(t_txn.OPS)) t_txn.TxnPtr {
	write_map := bohm.Prios(ops)
	txn := TXN{txn_id, write_map, &(map[*(rd.Version)]bool{}), bohm}
	txn.InstallKeys()
	return &txn
}

func (bohm *BOHM) GetWriteConflict() int {
	return bohm.Read_conflict
}


func quickGetOrInsert(index *(t_index.Mmap), key string) *rd.Record {
	r := index.Search(key)
	if r == nil {
		r = index.GetOrInsert(key, rd.NewRecord())
	}
	return r.(*rd.Record)
}

/*
first phase write (for use instead of InstallKeys)
*/
func (t *TXN) Install(key string)  {
	index := t.base.index
	// find or insert the key to index
	r := quickGetOrInsert(index, key)
	// r.install
	r.Install(t.txn_id, rd.PENDING)
}

/*
Install all keys in the first phase
*/
func (t *TXN) InstallKeys() {
	wm := t.write_map
	for key, count := range (*wm) {
		for i := 0; i < count; i ++ {
			t.Install(key)
		}
	}
}



func (t *TXN) Write(key string) int {
	index := t.base.index
	wv := t.write_version
	r := index.Search(key)
	if r == nil { // impossible run to this
		t_log.Log(t_log.PANIC, "error point in bohm op\n")
	}
	v := r.(*(rd.Record)).Write(t.txn_id)
	if v == nil {
		return t_txn.RE2FIRST 
	} else {
		(*wv)[v] = true // save the writing version for revert
		return t_txn.NEXT
	}
}


func (t *TXN) Read(key string) int {
	index := t.base.index

	r := quickGetOrInsert(index, key)
	v, is_n := r.Read(t.txn_id)
	if is_n {
		v = r.Install(-1, rd.COMMITED)
	}
	if v == nil {
		// t_log.Log(t_log.DEBUG, "txn %v read key %v failed in vl %v\n", t.txn_id, key, r.VersionListString())
		return t_txn.RE2FIRST
	} else {
		return t_txn.NEXT
	}
}

/*
Do nothing
*/
func (t *TXN) Reset() int {
	return t_txn.NEXT
}

func (t *TXN) Commit() int {
	wv := t.write_version
	for version, _ := range (*wv) {
		if version.GetStats() == rd.MODIFIED {
			version.UpdateStats(rd.STAGED)
		} else {
			t_log.Log(t_log.ERROR, "Staged failed Version: %v", version.GetString())
		}
	}
	return t_txn.NEXT
}

func (t *TXN) Init() int {
	return t_txn.NEXT
}
