package utils

import (
	"sync"
)

// WorkerPool 并发任务处理池
type WorkerPool struct {
	maxWorkers int
	taskQueue  chan func()
	wg         sync.WaitGroup
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 10 // 默认值
	}

	pool := &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan func(), maxWorkers*2),
	}

	// 启动工作协程
	for i := 0; i < maxWorkers; i++ {
		go pool.worker()
	}

	return pool
}

// worker 工作协程
func (p *WorkerPool) worker() {
	for task := range p.taskQueue {
		task()
		p.wg.Done()
	}
}

// Submit 提交任务到池
func (p *WorkerPool) Submit(task func()) {
	p.wg.Add(1)
	p.taskQueue <- task
}

// Wait 等待所有任务完成
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Close 关闭工作池
func (p *WorkerPool) Close() {
	close(p.taskQueue)
}

// ParallelExecute 并行执行多个任务并等待完成
func ParallelExecute(tasks []func()) {
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(t func()) {
			defer wg.Done()
			t()
		}(task)
	}

	wg.Wait()
}

// ParallelExecuteWithLimit 限制并发数的并行执行
func ParallelExecuteWithLimit(tasks []func(), maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	pool := NewWorkerPool(maxConcurrent)
	defer pool.Close()

	for _, task := range tasks {
		pool.Submit(task)
	}

	pool.Wait()
}

// AsyncTask 异步任务结构
type AsyncTask struct {
	Err    error
	Result interface{}
}

// ExecuteWithResults 并行执行任务并收集结果
func ExecuteWithResults(tasks []func() (interface{}, error)) []AsyncTask {
	results := make([]AsyncTask, len(tasks))
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for i, task := range tasks {
		go func(index int, t func() (interface{}, error)) {
			defer wg.Done()
			result, err := t()
			results[index] = AsyncTask{
				Err:    err,
				Result: result,
			}
		}(i, task)
	}

	wg.Wait()
	return results
}
