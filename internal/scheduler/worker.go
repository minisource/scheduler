package scheduler

import (
	"context"
	"sync"

	"github.com/minisource/scheduler/internal/models"
)

// JobTask represents a job to be processed by a worker
type JobTask struct {
	Job       models.Job
	Execution models.JobExecution
}

// WorkerFunc is the function type for processing jobs
type WorkerFunc func(task JobTask)

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers    int
	workerFunc WorkerFunc
	taskQueue  chan JobTask
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	mu         sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, fn WorkerFunc) *WorkerPool {
	if workers < 1 {
		workers = 1
	}

	return &WorkerPool{
		workers:    workers,
		workerFunc: fn,
		taskQueue:  make(chan JobTask, workers*10), // Buffer 10x workers
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start(ctx context.Context) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true
	p.mu.Unlock()

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	close(p.taskQueue)
	p.wg.Wait()
}

// Submit submits a job task to the pool
func (p *WorkerPool) Submit(task JobTask) bool {
	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return false
	}
	p.mu.RUnlock()

	select {
	case p.taskQueue <- task:
		return true
	default:
		// Queue is full
		return false
	}
}

// worker is the main worker loop
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			p.workerFunc(task)
		}
	}
}

// QueueSize returns the current queue size
func (p *WorkerPool) QueueSize() int {
	return len(p.taskQueue)
}

// WorkerCount returns the number of workers
func (p *WorkerPool) WorkerCount() int {
	return p.workers
}

// IsRunning returns whether the pool is running
func (p *WorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
