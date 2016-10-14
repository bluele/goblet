package goblet

import (
	"reflect"
	"sync"
)

type cache struct {
	mu   sync.RWMutex
	data map[string]*cacheRecord
}

func newCache() *cache {
	return &cache{
		data: make(map[string]*cacheRecord),
	}
}

type cacheRecord struct {
	value reflect.Value
	err   error
}

func (cc *cache) set(name string, record *cacheRecord) {
	cc.mu.Lock()
	cc.data[name] = record
	cc.mu.Unlock()
}

func (cc *cache) get(name string) (*cacheRecord, bool) {
	cc.mu.RLock()
	record, ok := cc.data[name]
	cc.mu.RUnlock()
	return record, ok
}
