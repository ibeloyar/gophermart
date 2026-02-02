package pg

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
)

type WorkerPool struct {
	jobsQueue  chan model.Order
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	numWorkers int

	pauseMu   sync.Mutex
	pauseCond *sync.Cond
	paused    bool
}

func NewWorkerPool() *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		jobsQueue:  make(chan model.Order, runtime.NumCPU()),
		ctx:        ctx,
		cancel:     cancel,
		wg:         sync.WaitGroup{},
		numWorkers: runtime.NumCPU(),
	}

	wp.pauseCond = sync.NewCond(&wp.pauseMu)

	return wp
}

func (r *Repository) worker(ctx context.Context, order model.Order) {
	accrual, err := r.getAccrual(ctx, order.Number)
	if err != nil {
		r.lg.Errorf("getting accruals error: %v", err)
		return
	}

	if accrual != nil {
		if err := r.updateOrderStatusAndAccrual(ctx,
			order.UserID,
			order.Number,
			accrual.Status,
			accrual.Accrual,
		); err != nil {
			r.lg.Errorf("updating order status error: %v", err)
		}
	}
}

func (wp *WorkerPool) shutdown() {
	wp.pauseMu.Lock()
	defer wp.pauseMu.Unlock()

	if wp.ctx.Err() != nil {
		wp.cancel()
	}

	select {
	case <-wp.jobsQueue:
	default:
		close(wp.jobsQueue) // закрываем только если пустой
	}

	wp.wg.Wait()
}

func (wp *WorkerPool) pausePoolWithTimer(duration time.Duration) {
	wp.pauseMu.Lock()
	defer wp.pauseMu.Unlock()

	if wp.paused {
		return
	}

	wp.paused = true

	wp.pauseCond.Broadcast()

	go func() {
		time.Sleep(duration)
		wp.resumePool()
	}()
}

func (wp *WorkerPool) resumePool() {
	wp.pauseMu.Lock()
	defer wp.pauseMu.Unlock()

	if !wp.paused {
		return
	}

	wp.paused = false

	// разблокируем все воркеры
	wp.pauseCond.Broadcast()
}
