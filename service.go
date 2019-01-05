package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/valyala/gorpc"
)

const (
	polling = time.Millisecond * 100
)

type Item struct {
	Key   string
	Value string
	PID   int
}

type Service struct {
	storage []*Item
	mutex   *sync.RWMutex
}

func (service *Service) Get(key string) string {
	service.mutex.RLock()
	item := service.find(key)
	service.mutex.RUnlock()
	if item != nil {
		return item.Value
	}

	return ""
}

func (service *Service) find(key string) *Item {
	for _, item := range service.storage {
		if item.Key == key {
			return item
		}
	}

	return nil
}

func (service *Service) Set(set *Item) {
	service.mutex.Lock()
	item := service.find(set.Key)
	if item == nil {
		service.storage = append(service.storage, set)
	} else {
		item.Value = set.Value
		item.PID = set.PID
	}
	service.mutex.Unlock()

	if set.PID != 0 {
		go service.wait(set)
	}
}

func (service *Service) List(reserved bool) []*Item {
	items := []*Item{}
	service.mutex.RLock()
	for _, item := range service.storage {
		items = append(items, item)
	}
	service.mutex.RUnlock()

	return items
}

func (service *Service) wait(target *Item) {
	procPath := filepath.Join("/proc", fmt.Sprint(target.PID))

	for {
		file, err := os.Open(procPath)
		if file != nil {
			file.Close()
		}
		if os.IsNotExist(err) {
			break
		}
		time.Sleep(polling)
	}

	service.mutex.Lock()
	for i, item := range service.storage {
		if item.Key == target.Key && item.PID == target.PID {
			service.storage = append(service.storage[:i], service.storage[i+1:]...)
			break
		}
	}

	service.mutex.Unlock()
}

func NewService() *Service {
	return &Service{
		storage: []*Item{},
		mutex:   &sync.RWMutex{},
	}
}

func NewServiceDispatcher(service *Service) *gorpc.Dispatcher {
	dispatcher := gorpc.NewDispatcher()
	dispatcher.AddFunc("get", service.Get)
	dispatcher.AddFunc("set", service.Set)
	dispatcher.AddFunc("list", service.List)

	return dispatcher
}
