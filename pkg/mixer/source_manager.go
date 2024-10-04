package mixer

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/wayjam/tvbox-mixproxy/config"
)

type Sourcer interface {
	GetSource(name string) ([]byte, error)
}

var (
	_ Sourcer = &SourceManager{}
	_ Sourcer = &Source{}
)

type SourceManager struct {
	sources map[string]*Source
	mu      sync.RWMutex
	ticker  *time.Ticker
	done    chan bool
}

type Source struct {
	config     config.Source
	lastUpdate time.Time
	data       []byte // Change this to []byte
	lastError  time.Time
	errorCount int
}

func (s *Source) Type() config.SourceType {
	return s.config.Type
}

func (s *Source) Name() string {
	return s.config.Name
}

func (s *Source) GetSource(name string) ([]byte, error) {
	return s.data, nil
}

func NewSourceManager(sources []config.Source) *SourceManager {
	sm := &SourceManager{
		sources: make(map[string]*Source),
		ticker:  time.NewTicker(1 * time.Minute), // 每分钟检查一次
		done:    make(chan bool),
	}

	for _, s := range sources {
		sm.sources[s.Name] = &Source{
			config: s,
		}
	}

	go sm.refreshLoop()

	return sm
}

func (sm *SourceManager) refreshLoop() {
	for {
		select {
		case <-sm.ticker.C:
			sm.refreshExpiredSources()
		case <-sm.done:
			sm.ticker.Stop()
			return
		}
	}
}

func (sm *SourceManager) refreshExpiredSources() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for name, source := range sm.sources {
		if time.Since(source.lastUpdate) > time.Duration(source.config.Interval)*time.Second {
			go sm.refreshSource(name) // 异步刷新，避免阻塞
		}
	}
}

func (sm *SourceManager) GetSource(name string) ([]byte, error) {
	sm.mu.RLock()
	source, ok := sm.sources[name]
	sm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("source not found: %s", name)
	}

	if time.Since(source.lastUpdate) > time.Duration(source.config.Interval)*time.Second || source.data == nil {
		if err := sm.refreshSource(name); err != nil {
			return nil, err
		}
	}

	return source.data, nil
}

func (sm *SourceManager) refreshSource(name string) error {
	sm.mu.Lock()
	source, ok := sm.sources[name]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("source not found: %s", name)
	}

	// 指数退避
	if !source.lastError.IsZero() {
		backoff := time.Duration(math.Pow(2, float64(source.errorCount))) * time.Second
		if time.Since(source.lastError) < backoff {
			sm.mu.Unlock()
			return fmt.Errorf("too many errors, try again later")
		}
	}

	sm.mu.Unlock()

	data, err := config.LoadData(source.config.URL)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err != nil {
		source.lastError = time.Now()
		source.errorCount++
		return err
	}

	source.data = data
	source.lastUpdate = time.Now()
	source.lastError = time.Time{}
	source.errorCount = 0
	return nil
}

func (sm *SourceManager) Close() {
	sm.done <- true
}
