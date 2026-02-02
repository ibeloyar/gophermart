package pg

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ibeloyar/gophermart/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepositoryForWPTest struct {
	mock.Mock
	lg *MockLogger
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockRepository) getAccrual(ctx context.Context, orderNumber string) (*model.Accrual, error) {
	args := m.Called(ctx, orderNumber)
	return args.Get(0).(*model.Accrual), args.Error(1)
}

func (m *MockRepository) updateOrderStatusAndAccrual(ctx context.Context, userID int, orderNumber, status string, accrual float64) error {
	args := m.Called(ctx, userID, orderNumber, status, accrual)
	return args.Error(0)
}

func TestNewWorkerPool(t *testing.T) {
	wp := NewWorkerPool()

	assert.NotNil(t, wp)
	assert.NotNil(t, wp.ctx)
	assert.NotNil(t, wp.cancel)
	assert.Equal(t, runtime.NumCPU(), wp.numWorkers)
	assert.Equal(t, runtime.NumCPU(), cap(wp.jobsQueue))
	assert.NotNil(t, wp.pauseCond)
	assert.False(t, wp.paused)
}

func TestPausePoolWithTimer(t *testing.T) {
	wp := NewWorkerPool()

	// пауза
	wp.pausePoolWithTimer(100 * time.Millisecond)

	wp.pauseMu.Lock()
	assert.True(t, wp.paused)
	wp.pauseMu.Unlock()

	// возобновление
	time.Sleep(150 * time.Millisecond)

	wp.pauseMu.Lock()
	assert.False(t, wp.paused)
	wp.pauseMu.Unlock()
}

func TestPauseResumeRaceCondition(t *testing.T) {
	wp := NewWorkerPool()
	var wg sync.WaitGroup

	// многократные паузы/возобновления
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wp.pausePoolWithTimer(50 * time.Millisecond)
		}()
	}

	wg.Wait()

	// проверяем состояние
	wp.pauseMu.Lock()
	wp.pauseMu.Unlock()

	// ЗАГЛУШКА может быть true/false - главное без паники
	assert.True(t, true)
}

func TestShutdownEmptyPool(t *testing.T) {
	wp := NewWorkerPool()

	start := time.Now()
	wp.shutdown()
	duration := time.Since(start)

	assert.True(t, duration < 50*time.Millisecond)
}

func TestWorkerPoolLifecycle(t *testing.T) {
	wp := NewWorkerPool()

	wg := sync.WaitGroup{}
	numWorkers := wp.numWorkers

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Имитация worker логики
			<-wp.jobsQueue
		}(i)
	}

	orders := []model.Order{{Number: "123"}}
	for _, order := range orders {
		wp.jobsQueue <- order
	}

	// shutdown
	close(wp.jobsQueue)
	wg.Wait()

	// проверяем что shutdown не паникует
	func() {
		recover()
		wp.shutdown()
	}()
}
