package gop

import "sync/atomic"

func needAdditionalWorker(currentPoolSize int32, maxPoolSize int, percent int) bool {
	if maxPoolSize == 0 || percent == 0 {
		return false
	}
	if float64(currentPoolSize)/float64(maxPoolSize)*100 > float64(percent) {
		return true
	}
	return false
}

func onExtraWorkerSpawnedWrapper(onExtraWorkerSpawned func(), counter *int32) func() {
	return func() {
		onExtraWorkerSpawned()
		atomic.AddInt32(counter, 1)
	}
}

func onExtraWorkerFinishedWrapper(onExtraWorkerFinished func(), counter *int32) func() {
	return func() {
		onExtraWorkerFinished()
		atomic.AddInt32(counter, -1)
	}
}

func onTaskTakenWrapper(onTaskTaken func(), counter *int32) func() {
	return func() {
		onTaskTaken()
		atomic.AddInt32(counter, 1)
	}
}

func onTaskFinishedWrapper(onTaskFinished func(), counter *int32) func() {
	return func() {
		onTaskFinished()
		atomic.AddInt32(counter, -1)
	}
}
