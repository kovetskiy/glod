package main

import (
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/valyala/gorpc"
)

const (
	polling = time.Millisecond * 100
)

type Item struct {
	Value string
	PID   int
}

type RequestSet struct {
	Key   string
	Value string
	PID   int
}

type Service struct {
	storage map[string]Item
	mutex   *sync.RWMutex
}

func (service *Service) Get(key string) string {
	service.mutex.RLock()
	item, ok := service.storage[key]
	service.mutex.RUnlock()
	if ok {
		return item.Value
	}

	return ""
}

func (service *Service) Set(set *RequestSet) {
	service.mutex.Lock()
	service.storage[set.Key] = Item{set.Value, set.PID}
	service.mutex.Unlock()

	if set.PID != 0 {
		go service.wait(set.PID, set.Key)
	}
}

func (service *Service) wait(pid int, key string) {
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Warningf(err, "unable to find process %d for key %q", pid, key)
		return
	}

	for process.Signal(syscall.Signal(0)) == nil {
		time.Sleep(polling)
	}

	service.mutex.Lock()
	delete(service.storage, key)
	service.mutex.Unlock()
}

func NewService() *Service {
	return &Service{
		storage: map[string]Item{},
		mutex:   &sync.RWMutex{},
	}
}

func NewServiceDispatcher(service *Service) *gorpc.Dispatcher {
	dispatcher := gorpc.NewDispatcher()
	dispatcher.AddFunc("get", service.Get)
	dispatcher.AddFunc("set", service.Set)

	return dispatcher
}
