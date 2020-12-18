package workqueue

import (
	"sync"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
)

const (
	DefaultWorkerCount = 6
	DefaultBacklog     = 10
)

type threadCtx struct {
	ff  *WorkQueue
	job func(key uint64) er.R
}

type WorkQueue struct {
	lock        sync.Mutex
	nextNum     uint64
	maxNum      uint64
	resultCache map[uint64]er.R
	resultChan  chan struct{}

	threads []threadCtx
}

func work(tctx threadCtx, nextNum uint64) {
	err := tctx.job(nextNum)
	tctx.ff.lock.Lock()
	defer tctx.ff.lock.Unlock()
	tctx.ff.resultCache[nextNum] = err
}

func task(tctx threadCtx) (bool, uint64) {
	tctx.ff.resultChan <- struct{}{}
	tctx.ff.lock.Lock()
	defer tctx.ff.lock.Unlock()
	num := tctx.ff.nextNum
	if num >= tctx.ff.maxNum {
		return true, num
	}
	tctx.ff.nextNum++
	return false, num
}

func threadLoop(tctx threadCtx) {
	for {
		if done, t := task(tctx); done {
			return
		} else {
			work(tctx, t)
		}
	}
}

func (ff *WorkQueue) Get(
	num uint64,
) er.R {
	var res er.R
	for {
		found := false
		ff.lock.Lock()
		if r, ok := ff.resultCache[num]; ok {
			delete(ff.resultCache, num)
			<-ff.resultChan
			res = r
			found = true
		}
		ff.lock.Unlock()
		if found {
			return res
		}
		time.Sleep(time.Millisecond)
	}
}

func New(workerCount,
	maxResults int,
	rangeMin uint64,
	rangeMax uint64,
	job func(key uint64) er.R,
) *WorkQueue {
	out := WorkQueue{
		resultCache: make(map[uint64]er.R),
		resultChan:  make(chan struct{}, maxResults),
		threads:     make([]threadCtx, workerCount),
		nextNum:     rangeMin,
		maxNum:      rangeMax,
	}
	for i := 0; i < workerCount; i++ {
		out.threads[i].ff = &out
		out.threads[i].job = job
		go threadLoop(out.threads[i])
	}
	return &out
}
