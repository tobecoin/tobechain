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

package ethapi

import (
    "context"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core"
    "github.com/ethereum/go-ethereum/log"
    
)

// SecurityAPI cung cấp các API liên quan đến bảo mật
type SecurityAPI struct {
    b Backend
}

// NewSecurityAPI tạo instance mới của SecurityAPI
func NewSecurityAPI(b Backend) *SecurityAPI {
    return &SecurityAPI{b}
}


// ============= WHITELIST APIS =============
// AddToWhitelist thêm địa chỉ vào whitelist deploy contract
func (api *SecurityAPI) AddToWhitelist(ctx context.Context, address common.Address) (bool, error) {

    core.GetSecurityConfig().AddToWhitelist(address)
    return true, nil
}


// RemoveFromWhitelist xóa địa chỉ khỏi whitelist
func (api *SecurityAPI) RemoveFromWhitelist(ctx context.Context, address common.Address) (bool, error) {
    core.GetSecurityConfig().RemoveFromWhitelist(address)
    return true, nil
}

// GetWhitelist trả về danh sách whitelist hiện tại
func (api *SecurityAPI) GetWhitelist(ctx context.Context) ([]common.Address, error) {
    addresses := []common.Address{}
    
    // Sử dụng helper function thay vì truy cập trực tiếp mutex
    whitelistMap := core.GetSecurityConfig().GetWhitelistAddresses()
    
    for addr := range whitelistMap {
        addresses = append(addresses, addr)
    }
    return addresses, nil
}

// ============= BLACKLIST APIS =============
// AddToBlacklist thêm địa chỉ vào blacklist
// Địa chỉ bị blacklist không thể giao dịch hoặc tương tác với blockchain
func (api *SecurityAPI) AddToBlacklist(ctx context.Context, address common.Address) (bool, error) {
    core.GetSecurityConfig().AddToBlacklist(address)
    return true, nil
}

// RemoveFromBlacklist xóa địa chỉ khỏi blacklist
func (api *SecurityAPI) RemoveFromBlacklist(ctx context.Context, address common.Address) (bool, error) {
    core.GetSecurityConfig().RemoveFromBlacklist(address)
    return true, nil
}

// GetBlacklist trả về danh sách blacklist hiện tại
func (api *SecurityAPI) GetBlacklist(ctx context.Context) ([]common.Address, error) {
    addresses := []common.Address{}
    
    // Sử dụng helper function thay vì truy cập trực tiếp mutex
    blacklistMap := core.GetSecurityConfig().GetBlacklistAddresses()
    
    for addr := range blacklistMap {
        addresses = append(addresses, addr)
    }
    return addresses, nil
}

// ============= FREEZE APIS =============
// FreezeAccount đóng băng toàn bộ tài sản, contract và tương tác của địa chỉ
func (api *SecurityAPI) FreezeAccount(ctx context.Context, address common.Address) (bool, error) {
    core.GetSecurityConfig().FreezeAccount(address)
    return true, nil
}

// UnfreezeAccount bỏ đóng băng tài khoản
func (api *SecurityAPI) UnfreezeAccount(ctx context.Context, address common.Address) (bool, error) {
    core.GetSecurityConfig().UnfreezeAccount(address)
    return true, nil
}

// GetFrozenAccounts trả về danh sách accounts bị đóng băng
func (api *SecurityAPI) GetFrozenAccounts(ctx context.Context) ([]common.Address, error) {
    addresses := []common.Address{}
    
    // Sử dụng helper function thay vì truy cập trực tiếp mutex
    frozenMap := core.GetSecurityConfig().GetFrozenAddresses()
    
    for addr := range frozenMap {
        addresses = append(addresses, addr)
    }
    return addresses, nil
}

// ============= CONTRACT MANAGEMENT APIS =============
// GetDeployedContracts trả về danh sách contract đã được triển khai bởi một địa chỉ
func (api *SecurityAPI) GetDeployedContracts(ctx context.Context, address common.Address) ([]common.Address, error) {
    return core.GetSecurityConfig().GetDeployedContracts(address), nil
}

// GetAllDeployedContracts trả về map của tất cả các contracts đã được triển khai
func (api *SecurityAPI) GetAllDeployedContracts(ctx context.Context) (map[string][]string, error) {
    contractsMap := core.GetSecurityConfig().GetAllDeployedContracts()
    
    // Thêm log để debug
    log.Debug("Getting all deployed contracts", "count", len(contractsMap))
    
    // Chuyển đổi sang map[string][]string để tiện sử dụng trong JSON-RPC
    result := make(map[string][]string)
    for owner, contracts := range contractsMap {
        ownerStr := owner.Hex()
        contractsStr := make([]string, len(contracts))
        for i, contract := range contracts {
            contractsStr[i] = contract.Hex()
        }
        result[ownerStr] = contractsStr
        
        // Thêm log chi tiết
        log.Debug("Owner contracts", "owner", ownerStr, "contracts", len(contractsStr))
    }
    
    return result, nil
}

// IsContractFrozen kiểm tra xem một contract có bị đóng băng hay không
func (api *SecurityAPI) IsContractFrozen(ctx context.Context, contractAddr common.Address) (bool, error) {
    return core.GetSecurityConfig().IsContractFrozen(contractAddr), nil
}