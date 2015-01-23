// palloc use sync.Pool for []byte alloc

package palloc

import "sync"

// malloc step base
const ALLOC_BASE int = 16

//
type allocPool struct {
	step  int // step size for new Pool
	pools map[int]*sync.Pool
	m     sync.Mutex
}

//
func poolNewFun(size int) func() interface{} {
	return func() interface{} {
		return make([]byte, size)
	}
}

//
func newAlloc(step int) *allocPool {
	if step <= ALLOC_BASE {
		step = ALLOC_BASE
	}
	step = step + step%ALLOC_BASE
	ap := &allocPool{
		step:  step,
		pools: make(map[int]*sync.Pool),
	}
	ap.pools[step] = &sync.Pool{
		New: poolNewFun(step),
	}
	return ap
}

//
func (ap *allocPool) Get(size int) []byte {
	if size < 0 {
		size = 0
	}
	asize := size + size%ap.step
	ap.m.Lock()
	if p, ok := ap.pools[asize]; ok {
		ap.m.Unlock()
		return p.Get().([]byte)[:size]
	}
	// create new pool
	ap.pools[asize] = &sync.Pool{
		New: poolNewFun(asize),
	}
	ap.m.Unlock()
	return ap.pools[asize].Get().([]byte)[:size]
}

//
func (ap *allocPool) Put(buf []byte) {
	if cap(buf) <= ap.step {
		ap.pools[ap.step].Put(buf)
		return
	}
	asize := cap(buf) + cap(buf)%ap.step
	ap.m.Lock()
	if p, ok := ap.pools[asize]; ok {
		ap.m.Unlock()
		p.Put(buf)
		return
	}
	// create new pool
	ap.pools[asize] = &sync.Pool{
		New: poolNewFun(asize),
	}
	ap.m.Unlock()
	ap.pools[asize].Put(buf)
	return
}

//
var pool = newAlloc(ALLOC_BASE)

//
func NewAlloc(size int) *allocPool {
	return newAlloc(size)
}

//
func Get(size int) []byte {
	return pool.Get(size)
}

//
func Put(buf []byte) {
	pool.Put(buf)
}

//
