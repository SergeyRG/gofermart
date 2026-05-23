package services

import (
	"sync"
	"time"
)

type FreezeManager struct {
	RWMu  sync.RWMutex
	until time.Time
}

func NewFreezeManager() *FreezeManager {
	return &FreezeManager{}
}

func (fm *FreezeManager) Freeze(sec int) {
	fm.RWMu.Lock()
	defer fm.RWMu.Unlock()

	fm.until = time.Now().Add(time.Duration(sec) * time.Second)
}

func (fm *FreezeManager) isFrozen() bool {
	fm.RWMu.RLock()
	defer fm.RWMu.RUnlock()
	return time.Now().Before(fm.until)
}

func (fm *FreezeManager) DurationRemaining() time.Duration {
	fm.RWMu.RLock()
	defer fm.RWMu.RUnlock()

	remaining := time.Until(fm.until)
	if remaining < 0 {
		return 0
	}
	return remaining
}
