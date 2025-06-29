// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
    "errors"
    "sync"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethdb"
    "github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/rlp"
)

// SecurityConfig quản lý các cấu hình bảo mật
type SecurityConfig struct {
    // Whitelist cho deploy contract  
    ContractDeployWhitelist map[common.Address]bool
    whitelistMu            sync.RWMutex
    
    // Blacklist - chặn giao dịch và tương tác
    BlacklistedAddresses   map[common.Address]bool
    blacklistMu           sync.RWMutex
    
    // Frozen accounts - đóng băng toàn bộ tài sản, contract và tương tác
    FrozenAccounts        map[common.Address]bool
    frozenMu             sync.RWMutex
    
    // DeployedContracts - lưu trữ danh sách contract đã được triển khai bởi mỗi địa chỉ
    DeployedContracts     map[common.Address][]common.Address
    contractsMu          sync.RWMutex
}

// Biến global sử dụng singleton pattern
var (
    securityConfigInstance *SecurityConfig
    once                  sync.Once
)

// GetSecurityConfig trả về instance của security config (thread-safe singleton)
func GetSecurityConfig() *SecurityConfig {
    once.Do(func() {
        if securityConfigInstance == nil {
            securityConfigInstance = &SecurityConfig{
                ContractDeployWhitelist: make(map[common.Address]bool),
                BlacklistedAddresses:    make(map[common.Address]bool),
                FrozenAccounts:          make(map[common.Address]bool),
                DeployedContracts:       make(map[common.Address][]common.Address),
            }
            log.Info("Security config initialized")
        }
    })
    return securityConfigInstance
}

// SetSecurityConfig đặt instance cấu hình bảo mật toàn cục
func SetSecurityConfig(config *SecurityConfig) {
    securityConfigInstance = config
}

// ============= CÁC HÀM WHITELIST =============
// AddToWhitelist thêm địa chỉ vào whitelist cho deploy contract
func (sc *SecurityConfig) AddToWhitelist(addr common.Address) error {
    if addr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.whitelistMu.Lock()
    defer sc.whitelistMu.Unlock()
    
    sc.ContractDeployWhitelist[addr] = true
    log.Info("Address added to contract deployment whitelist", "address", addr.Hex())
    return nil
}

// RemoveFromWhitelist xóa địa chỉ khỏi whitelist
func (sc *SecurityConfig) RemoveFromWhitelist(addr common.Address) error {
    if addr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.whitelistMu.Lock()
    defer sc.whitelistMu.Unlock()
    
    if _, exists := sc.ContractDeployWhitelist[addr]; exists {
        delete(sc.ContractDeployWhitelist, addr)
        log.Info("Address removed from contract deployment whitelist", "address", addr.Hex())
    } else {
        log.Debug("Address not in whitelist, nothing to remove", "address", addr.Hex())
    }
    return nil
}

// IsWhitelisted kiểm tra xem địa chỉ có trong whitelist hay không
func (sc *SecurityConfig) IsWhitelisted(addr common.Address) bool {
    sc.whitelistMu.RLock()
    defer sc.whitelistMu.RUnlock()
    return sc.ContractDeployWhitelist[addr]
}

// ============= CÁC HÀM BLACKLIST =============
// AddToBlacklist thêm địa chỉ vào blacklist
// Địa chỉ bị blacklist không thể giao dịch hoặc tương tác với blockchain
func (sc *SecurityConfig) AddToBlacklist(addr common.Address) error {
    if addr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.blacklistMu.Lock()
    defer sc.blacklistMu.Unlock()
    
    sc.BlacklistedAddresses[addr] = true
    log.Info("Address added to blacklist", "address", addr.Hex())
    return nil
}

// RemoveFromBlacklist xóa địa chỉ khỏi blacklist
func (sc *SecurityConfig) RemoveFromBlacklist(addr common.Address) error {
    if addr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.blacklistMu.Lock()
    defer sc.blacklistMu.Unlock()
    
    if _, exists := sc.BlacklistedAddresses[addr]; exists {
        delete(sc.BlacklistedAddresses, addr)
        log.Info("Address removed from blacklist", "address", addr.Hex())
    } else {
        log.Debug("Address not in blacklist, nothing to remove", "address", addr.Hex())
    }
    return nil
}

// IsBlacklisted kiểm tra xem địa chỉ có nằm trong blacklist hay không
func (sc *SecurityConfig) IsBlacklisted(addr common.Address) bool {
    sc.blacklistMu.RLock()
    defer sc.blacklistMu.RUnlock()
    return sc.BlacklistedAddresses[addr]
}

// ============= CÁC HÀM FREEZE ACCOUNT =============
// FreezeAccount đóng băng toàn bộ tài sản, contract và tương tác của địa chỉ
func (sc *SecurityConfig) FreezeAccount(addr common.Address) error {
    if addr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.frozenMu.Lock()
    defer sc.frozenMu.Unlock()
    
    sc.FrozenAccounts[addr] = true
    log.Info("Account frozen", "address", addr.Hex())
    return nil
}

// UnfreezeAccount bỏ đóng băng tài khoản
func (sc *SecurityConfig) UnfreezeAccount(addr common.Address) error {
    if addr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.frozenMu.Lock()
    defer sc.frozenMu.Unlock()
    
    if _, exists := sc.FrozenAccounts[addr]; exists {
        delete(sc.FrozenAccounts, addr)
        log.Info("Account unfrozen", "address", addr.Hex())
    } else {
        log.Debug("Account not frozen, nothing to unfreeze", "address", addr.Hex())
    }
    return nil
}

// IsFrozen kiểm tra xem tài khoản có bị đóng băng hay không
func (sc *SecurityConfig) IsFrozen(addr common.Address) bool {
    sc.frozenMu.RLock()
    defer sc.frozenMu.RUnlock()
    return sc.FrozenAccounts[addr]
}

// ============= CÁC HÀM QUẢN LÝ CONTRACT =============
// RegisterDeployedContract lưu trữ thông tin contract đã được triển khai bởi một địa chỉ
func (sc *SecurityConfig) RegisterDeployedContract(owner common.Address, contractAddr common.Address) error {
    if owner == (common.Address{}) || contractAddr == (common.Address{}) {
        return errors.New("invalid address: zero address")
    }
    
    sc.contractsMu.Lock()
    defer sc.contractsMu.Unlock()
    
    if _, exists := sc.DeployedContracts[owner]; !exists {
        sc.DeployedContracts[owner] = make([]common.Address, 0)
    }
    
    // Kiểm tra xem contract đã được đăng ký chưa để tránh trùng lặp
    for _, addr := range sc.DeployedContracts[owner] {
        if addr == contractAddr {
            log.Debug("Contract already registered", "owner", owner.Hex(), "contract", contractAddr.Hex())
            return nil
        }
    }
    
    sc.DeployedContracts[owner] = append(sc.DeployedContracts[owner], contractAddr)
    log.Info("Contract registered successfully", "owner", owner.Hex(), "contract", contractAddr.Hex(), "total", len(sc.DeployedContracts[owner]))
    return nil
}

// GetDeployedContracts trả về danh sách contract đã được triển khai bởi một địa chỉ
func (sc *SecurityConfig) GetDeployedContracts(owner common.Address) []common.Address {
    sc.contractsMu.RLock()
    defer sc.contractsMu.RUnlock()
    
    if contracts, exists := sc.DeployedContracts[owner]; exists {
        // Tạo một bản sao để tránh thay đổi trực tiếp
        result := make([]common.Address, len(contracts))
        copy(result, contracts)
        return result
    }
    
    return make([]common.Address, 0)
}

// IsContractFrozen kiểm tra xem một contract có bị đóng băng hay không
// Contract bị đóng băng nếu chủ sở hữu của nó bị đóng băng
func (sc *SecurityConfig) IsContractFrozen(contractAddr common.Address) bool {
    if contractAddr == (common.Address{}) {
        return false
    }
    
    // Tìm chủ sở hữu của contract
    var owner common.Address
    var found bool
    
    sc.contractsMu.RLock()
    for ownerAddr, contracts := range sc.DeployedContracts {
        for _, addr := range contracts {
            if addr == contractAddr {
                owner = ownerAddr
                found = true
                break
            }
        }
        if found {
            break
        }
    }
    sc.contractsMu.RUnlock()
    
    if !found {
        return false
    }
    
    // Kiểm tra nếu chủ sở hữu bị đóng băng
    return sc.IsFrozen(owner)
}

// ============= HELPER FUNCTIONS =============
// Helper functions để tránh expose mutex ra ngoài
// =============================================

// GetWhitelistAddresses trả về copy của whitelist map
func (sc *SecurityConfig) GetWhitelistAddresses() map[common.Address]bool {
    sc.whitelistMu.RLock()
    defer sc.whitelistMu.RUnlock()
    
    result := make(map[common.Address]bool)
    for addr, val := range sc.ContractDeployWhitelist {
        result[addr] = val
    }
    return result
}

// GetBlacklistAddresses trả về copy của blacklist map
func (sc *SecurityConfig) GetBlacklistAddresses() map[common.Address]bool {
    sc.blacklistMu.RLock()
    defer sc.blacklistMu.RUnlock()
    
    result := make(map[common.Address]bool)
    for addr, val := range sc.BlacklistedAddresses {
        result[addr] = val
    }
    return result
}

// GetFrozenAddresses trả về copy của frozen accounts map
func (sc *SecurityConfig) GetFrozenAddresses() map[common.Address]bool {
    sc.frozenMu.RLock()
    defer sc.frozenMu.RUnlock()
    
    result := make(map[common.Address]bool)
    for addr, val := range sc.FrozenAccounts {
        result[addr] = val
    }
    return result
}

// GetAllDeployedContracts trả về map của tất cả các contracts đã được triển khai
func (sc *SecurityConfig) GetAllDeployedContracts() map[common.Address][]common.Address {
    sc.contractsMu.RLock()
    defer sc.contractsMu.RUnlock()
    
    log.Debug("Getting all deployed contracts", "count", len(sc.DeployedContracts))
    
    result := make(map[common.Address][]common.Address)
    for owner, contracts := range sc.DeployedContracts {
        contractsCopy := make([]common.Address, len(contracts))
        copy(contractsCopy, contracts)
        result[owner] = contractsCopy
        
        log.Debug("Owner contracts", "owner", owner.Hex(), "count", len(contracts))
        for i, contract := range contracts {
            log.Debug("Contract details", "index", i, "address", contract.Hex())
        }
    }
    
    return result
}

// ============= PERSISTENCE FUNCTIONS =============
// Các hàm lưu trữ và khôi phục cấu hình bảo mật

// Cấu trúc dữ liệu để lưu trữ
type securityConfigData struct {
    Whitelist  []common.Address
    Blacklist  []common.Address
    Frozen     []common.Address
    Contracts []contractOwnerMapping
}
// Thêm kiểu mới để lưu trữ mapping giữa owner và contracts
type contractOwnerMapping struct {
    Owner     common.Address
    Contracts []common.Address
}

// SaveConfig lưu cấu hình bảo mật vào cơ sở dữ liệu
func (sc *SecurityConfig) SaveConfig(db ethdb.Database) error {
    if db == nil {
        return errors.New("database connection is nil")
    }
    
    // Tạo log để chỉ ra rằng chúng ta đang lưu
    log.Info("Đang lưu cấu hình bảo mật vào cơ sở dữ liệu")
    
    // Thu thập dữ liệu
    data := securityConfigData{
        Whitelist: make([]common.Address, 0),
        Blacklist: make([]common.Address, 0),
        Frozen:    make([]common.Address, 0),
        Contracts: make([]contractOwnerMapping, 0),
    }
    
    // Lấy whitelist
    sc.whitelistMu.RLock()
    for addr := range sc.ContractDeployWhitelist {
        data.Whitelist = append(data.Whitelist, addr)
    }
    sc.whitelistMu.RUnlock()
    
    // Lấy blacklist
    sc.blacklistMu.RLock()
    for addr := range sc.BlacklistedAddresses {
        data.Blacklist = append(data.Blacklist, addr)
    }
    sc.blacklistMu.RUnlock()
    
    // Lấy frozen accounts
    sc.frozenMu.RLock()
    for addr := range sc.FrozenAccounts {
        data.Frozen = append(data.Frozen, addr)
    }
    sc.frozenMu.RUnlock()
    
    // Lấy deployed contracts và chuyển đổi map thành slice
    sc.contractsMu.RLock()
    for owner, contracts := range sc.DeployedContracts {
        contractsCopy := make([]common.Address, len(contracts))
        copy(contractsCopy, contracts)
        data.Contracts = append(data.Contracts, contractOwnerMapping{
            Owner:     owner,
            Contracts: contractsCopy,
        })
    }
    sc.contractsMu.RUnlock()
    
    // Encode và lưu
    encoded, err := rlp.EncodeToBytes(data)
    if err != nil {
        log.Error("Không thể mã hóa cấu hình bảo mật", "lỗi", err)
        return err
    }
    
    err = db.Put([]byte("security-config"), encoded)
    if err != nil {
        log.Error("Không thể lưu cấu hình bảo mật vào cơ sở dữ liệu", "lỗi", err)
        return err
    }
    
    log.Info("Đã lưu cấu hình bảo mật vào cơ sở dữ liệu thành công", 
             "whitelist", len(data.Whitelist), 
             "blacklist", len(data.Blacklist), 
             "frozen", len(data.Frozen), 
             "contracts", len(data.Contracts))
    return nil
}

// LoadSecurityConfig tải cấu hình bảo mật từ cơ sở dữ liệu
func LoadSecurityConfig(db ethdb.Database) (*SecurityConfig, error) {
    if db == nil {
        return nil, errors.New("database connection is nil")
    }
    
    encoded, err := db.Get([]byte("security-config"))
    if err != nil {
        // Kiểm tra lỗi "not found"
        if err.Error() == "not found" || err.Error() == "leveldb: not found" {
            log.Info("Không tìm thấy cấu hình bảo mật trong cơ sở dữ liệu, khởi tạo cấu hình trống")
            return &SecurityConfig{
                ContractDeployWhitelist: make(map[common.Address]bool),
                BlacklistedAddresses:    make(map[common.Address]bool),
                FrozenAccounts:          make(map[common.Address]bool),
                DeployedContracts:       make(map[common.Address][]common.Address),
            }, nil
        }
        log.Error("Không thể tải cấu hình bảo mật từ cơ sở dữ liệu", "lỗi", err)
        return nil, err
    }
    
    var data securityConfigData
    if err := rlp.DecodeBytes(encoded, &data); err != nil {
        log.Error("Không thể giải mã cấu hình bảo mật", "lỗi", err)
        return nil, err
    }
    
    // Tạo config mới
    sc := &SecurityConfig{
        ContractDeployWhitelist: make(map[common.Address]bool),
        BlacklistedAddresses:    make(map[common.Address]bool),
        FrozenAccounts:          make(map[common.Address]bool),
        DeployedContracts:       make(map[common.Address][]common.Address),
    }
    
    // Khôi phục whitelist
    for _, addr := range data.Whitelist {
        sc.ContractDeployWhitelist[addr] = true
    }
    
    // Khôi phục blacklist
    for _, addr := range data.Blacklist {
        sc.BlacklistedAddresses[addr] = true
    }
    
    // Khôi phục frozen accounts
    for _, addr := range data.Frozen {
        sc.FrozenAccounts[addr] = true
    }
    
    // Khôi phục deployed contracts (chuyển đổi slice thành map)
    for _, ownerMapping := range data.Contracts {
        owner := ownerMapping.Owner
        contracts := ownerMapping.Contracts
        
        contractsCopy := make([]common.Address, len(contracts))
        copy(contractsCopy, contracts)
        sc.DeployedContracts[owner] = contractsCopy
    }
    
    log.Info("Đã tải cấu hình bảo mật từ cơ sở dữ liệu", 
             "whitelist", len(data.Whitelist), 
             "blacklist", len(data.Blacklist), 
             "frozen", len(data.Frozen), 
             "contracts", len(data.Contracts))
        
    return sc, nil
}