/*
	Package queues
*/

package queues

import "sync"

// LifoQ is LIFO Queue with lock
type LifoQ struct {
	m    sync.Mutex
	ptr  int
	last int
	q    []interface{}
}

//
func NewLifoQ(initsize int) *LifoQ {
	if initsize < 0 {
		initsize = 0
	}
	return &LifoQ{
		last: -1,
		q:    make([]interface{}, initsize),
	}
}

//
func (q *LifoQ) Len() int {
	return q.last + 1
}

//
func (q *LifoQ) Pop() interface{} {
	q.m.Lock()
	if q.last < 0 {
		q.m.Unlock()
		return nil
	}
	q.ptr = q.last
	q.last--
	q.m.Unlock()
	return q.q[q.ptr]
}

//
func (q *LifoQ) Push(p interface{}) {
	q.m.Lock()
	q.q = append(q.q, p)
	q.last++
	q.m.Unlock()
	return
}

//
func (q *LifoQ) Compact() {
	q.m.Lock()
	for i := len(q.q) - 1; i > q.last; i-- {
		q.q[i] = nil
	}
	q.m.Unlock()
	return
}
