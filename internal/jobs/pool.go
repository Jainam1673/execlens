package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
)

// WorkerPool coordinates bounded background execution of benchmark runs.
type WorkerPool struct {
	executor    *engine.Executor
	db          *database.DB
	concurrency int
	queue       chan string
	activeMu    sync.Mutex
	activeRuns  map[string]context.CancelFunc
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewWorkerPool(db *database.DB, executor *engine.Executor, concurrency int) *WorkerPool {
	if concurrency <= 0 {
		// Default to 1 concurrent job to avoid CPU core contention and measurement distortion
		concurrency = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &WorkerPool{
		executor:    executor,
		db:          db,
		concurrency: concurrency,
		queue:       make(chan string, 100),
		activeRuns:  make(map[string]context.CancelFunc),
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < concurrency; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return p
}

func (p *WorkerPool) Submit(runID string) error {
	select {
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is stopping")
	case p.queue <- runID:
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

func (p *WorkerPool) Cancel(runID string) error {
	p.activeMu.Lock()
	cancelFn, exists := p.activeRuns[runID]
	p.activeMu.Unlock()

	if exists && cancelFn != nil {
		cancelFn()
		_ = p.db.UpdateRunStatus(context.Background(), runID, database.StatusCancelled, "User requested cancellation")
		return nil
	}

	// If not currently running (maybe queued), update state directly
	run, err := p.db.GetRun(context.Background(), runID)
	if err == nil && run != nil && (run.Status == database.StatusQueued || run.Status == database.StatusPreparing) {
		_ = p.db.UpdateRunStatus(context.Background(), runID, database.StatusCancelled, "Cancelled while queued")
		return nil
	}

	return fmt.Errorf("run %s is not active or already finished", runID)
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case runID, ok := <-p.queue:
			if !ok {
				return
			}
			p.processJob(runID)
		}
	}
}

func (p *WorkerPool) processJob(runID string) {
	// Panic safety wrapper so a panic in a worker doesn't bring down the API server
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker Panic] recovered from panic in run %s: %v", runID, r)
			_ = p.db.UpdateRunStatus(context.Background(), runID, database.StatusFailed, fmt.Sprintf("Internal worker error: %v", r))
		}
	}()

	runCtx, cancel := context.WithCancel(p.ctx)

	p.activeMu.Lock()
	p.activeRuns[runID] = cancel
	p.activeMu.Unlock()

	defer func() {
		p.activeMu.Lock()
		delete(p.activeRuns, runID)
		p.activeMu.Unlock()
		cancel()
	}()

	// Execute run through engine
	if err := p.executor.ExecuteRun(runCtx, runID); err != nil {
		log.Printf("[Worker] Run %s finished with error: %v", runID, err)
	} else {
		log.Printf("[Worker] Run %s completed successfully", runID)
	}
}

func (p *WorkerPool) Shutdown(ctx context.Context) error {
	p.cancel()
	close(p.queue)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
