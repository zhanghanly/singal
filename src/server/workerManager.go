package singal

import (
	"sync"
	"sync/atomic"
	"time"
)

type WorkerManager struct {
	workers map[string]*Worker
	mu      sync.RWMutex
}

var gWorkerManager *WorkerManager

const (
	heartbeatTimeout      = 60 * time.Second
	workerCleanupInterval = 30 * time.Second
)

func NewWorkerManager() *WorkerManager {
	wm := &WorkerManager{
		workers: make(map[string]*Worker),
	}
	go wm.cleanupWorkers()
	return wm
}

func (wm *WorkerManager) RegisterWorker(workerId, publicIp string, publicPort uint32, useUdp bool) (*Worker, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if worker, exists := wm.workers[workerId]; exists {
		worker.mu.Lock()
		worker.lastHeartbeat = time.Now()
		worker.status = WorkerStatusOnline
		worker.mu.Unlock()
		return worker, nil
	}

	worker := &Worker{
		workerId:      workerId,
		publicIp:      publicIp,
		publicPort:    publicPort,
		useUdp:        useUdp,
		status:        WorkerStatusOnline,
		lastHeartbeat: time.Now(),
		routers:       make(map[string]*Router),
	}

	wm.workers[workerId] = worker
	logger.Infof("Worker registered: id=%s, ip=%s, port=%d", workerId, publicIp, publicPort)
	return worker, nil
}

func (wm *WorkerManager) UpdateWorkerStats(workerId string, routerCount, cpuUsage, memoryUsage uint32) bool {
	wm.mu.RLock()
	worker, exists := wm.workers[workerId]
	wm.mu.RUnlock()

	if !exists {
		return false
	}

	worker.mu.Lock()
	worker.routerCount = routerCount
	worker.cpuUsage = cpuUsage
	worker.memoryUsage = memoryUsage
	worker.lastHeartbeat = time.Now()
	worker.mu.Unlock()

	return true
}

func (wm *WorkerManager) Heartbeat(workerId string) bool {
	wm.mu.RLock()
	worker, exists := wm.workers[workerId]
	wm.mu.RUnlock()

	if !exists {
		return false
	}

	worker.mu.Lock()
	worker.lastHeartbeat = time.Now()
	worker.mu.Unlock()

	return true
}

func (wm *WorkerManager) RemoveWorker(workerId string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if worker, exists := wm.workers[workerId]; exists {
		worker.mu.Lock()
		worker.status = WorkerStatusOffline
		worker.mu.Unlock()
		delete(wm.workers, workerId)
		logger.Infof("Worker removed: id=%s", workerId)
	}
}

func (wm *WorkerManager) GetWorker(workerId string) (*Worker, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	worker, exists := wm.workers[workerId]
	return worker, exists
}

func (wm *WorkerManager) GetAllWorkers() []*Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workers := make([]*Worker, 0, len(wm.workers))
	for _, worker := range wm.workers {
		workers = append(workers, worker)
	}
	return workers
}

func (wm *WorkerManager) GetOnlineWorkers() []*Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workers := make([]*Worker, 0)
	now := time.Now()
	for _, worker := range wm.workers {
		worker.mu.RLock()
		if worker.status == WorkerStatusOnline && now.Sub(worker.lastHeartbeat) < heartbeatTimeout {
			workers = append(workers, worker)
		}
		worker.mu.RUnlock()
	}
	return workers
}

func (wm *WorkerManager) ChooseBestWorker() *Worker {
	workers := wm.GetOnlineWorkers()
	if len(workers) == 0 {
		logger.Warn("No online workers available")
		return nil
	}

	var bestWorker *Worker
	bestScore := int64(0)

	for _, worker := range workers {
		worker.mu.RLock()
		score := int64(1000) - int64(worker.routerCount*10) - int64(worker.cpuUsage) - int64(worker.memoryUsage/2)
		if bestWorker == nil || score > bestScore {
			bestWorker = worker
			bestScore = score
		}
		worker.mu.RUnlock()
	}

	return bestWorker
}

func (wm *WorkerManager) cleanupWorkers() {
	ticker := time.NewTicker(workerCleanupInterval)
	for range ticker.C {
		wm.mu.Lock()
		now := time.Now()
		for workerId, worker := range wm.workers {
			worker.mu.Lock()
			if now.Sub(worker.lastHeartbeat) > heartbeatTimeout {
				worker.status = WorkerStatusOffline
				delete(wm.workers, workerId)
				logger.Infof("Worker timed out and removed: id=%s", workerId)
			}
			worker.mu.Unlock()
		}
		wm.mu.Unlock()
	}
}

func (wm *WorkerManager) GetWorkerCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.workers)
}

func (wm *WorkerManager) AddRouter(workerId string, router *Router) bool {
	worker, exists := wm.GetWorker(workerId)
	if !exists {
		return false
	}

	worker.mu.Lock()
	defer worker.mu.Unlock()
	worker.routers[router.routerId] = router
	atomic.AddUint32(&worker.routerCount, 1)
	return true
}

func (wm *WorkerManager) RemoveRouter(workerId, routerId string) bool {
	worker, exists := wm.GetWorker(workerId)
	if !exists {
		return false
	}

	worker.mu.Lock()
	defer worker.mu.Unlock()
	if _, ok := worker.routers[routerId]; ok {
		delete(worker.routers, routerId)
		atomic.AddUint32(&worker.routerCount, ^uint32(0))
		return true
	}
	return false
}

func GetWorkerManager() *WorkerManager {
	return gWorkerManager
}

func InitWorkerManager() {
	gWorkerManager = NewWorkerManager()
}
