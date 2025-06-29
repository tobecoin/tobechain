package core

import (
    "errors"
    "sync"
    "github.com/ethereum/go-ethereum/common"
)

var (
    ErrBlacklisted = errors.New("address is blacklisted")
    ErrFrozen      = errors.New("address is frozen")
)

type SecurityManager struct {
    blacklist map[common.Address]bool
    frozen    map[common.Address]bool
    mu        sync.RWMutex
    enabled   bool
}

// Global security manager instance
var Security = &SecurityManager{
    blacklist: make(map[common.Address]bool),
    frozen:    make(map[common.Address]bool),
    enabled:   false,
}

func (s *SecurityManager) Enable() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.enabled = true
}

func (s *SecurityManager) IsEnabled() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.enabled
}

func (s *SecurityManager) BlacklistAddress(addr common.Address) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.blacklist[addr] = true
    s.frozen[addr] = true
}

func (s *SecurityManager) IsBlacklisted(addr common.Address) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if !s.enabled {
        return false
    }
    return s.blacklist[addr]
}

func (s *SecurityManager) IsFrozen(addr common.Address) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if !s.enabled {
        return false
    }
    return s.frozen[addr]
}

func (s *SecurityManager) RemoveFromBlacklist(addr common.Address) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.blacklist, addr)
    delete(s.frozen, addr)
}