package poi

import (
    "fmt"
    "sort"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rpc"
)

// API is a user facing RPC API to allow controlling the validator and voting
// mechanisms of the PoI scheme.
type API struct {
    chain consensus.ChainHeaderReader
    poi   *PoI
}

// GetValidatorRanking retrieves validators ranked by their PoI scores
func (api *API) GetValidatorRanking() ([]ValidatorRanking, error) {
    validators := api.poi.GetValidators()
    rankings := make([]ValidatorRanking, 0, len(validators))
    
    for _, validator := range validators {
        score := api.poi.CalculatePoIScore(validator)
        reputation := api.poi.GetReputation(validator)
        performance := api.poi.GetPerformance(validator)
        
        rankings = append(rankings, ValidatorRanking{
            Validator:   validator,
            PoIScore:    score,
            Reputation:  reputation,
            Performance: performance,
        })
    }
    
    // Sort by PoI score (descending)
    sort.Slice(rankings, func(i, j int) bool {
        return rankings[i].PoIScore > rankings[j].PoIScore
    })
    
    // Add rank information
    for i := range rankings {
        rankings[i].Rank = i + 1
    }
    
    return rankings, nil
}

// ValidatorRanking represents a validator's ranking information
type ValidatorRanking struct {
    Validator   common.Address `json:"validator"`
    Rank        int           `json:"rank"`
    PoIScore    float64       `json:"poiScore"`
    Reputation  float64       `json:"reputation"`
    Performance float64       `json:"performance"`
}

// GetNextValidator predicts the next validator to be selected
func (api *API) GetNextValidator() (common.Address, error) {
    header := api.chain.CurrentHeader()
    if header == nil {
        return common.Address{}, ErrUnknownBlock
    }
    
    return api.poi.SelectValidator(header.Number.Uint64() + 1)
}

// GetAlgorithmParams retrieves the current algorithm parameters
func (api *API) GetAlgorithmParams() (AlgorithmParams, error) {
    return AlgorithmParams{
        Alpha:                api.poi.alpha,
        Beta:                 api.poi.beta,
        DecayEpochSize:       DecayEpochSize,
        DecayFactor:          DecayFactor,
        BoostEpoch:           BoostEpoch,
        BoostFactor:          BoostFactor,
        CooldownBlocks:       CooldownBlocks,
        ConsecutiveLimit:     ConsecutiveLimit,
        SlidingWindowPercent: SlidingWindowPercent,
        DefaultReputation:    DefaultReputation,
    }, nil
}

// AlgorithmParams represents the PoI algorithm parameters
type AlgorithmParams struct {
    Alpha                float64 `json:"alpha"`
    Beta                 float64 `json:"beta"`
    DecayEpochSize       int     `json:"decayEpochSize"`
    DecayFactor          float64 `json:"decayFactor"`
    BoostEpoch           int     `json:"boostEpoch"`
    BoostFactor          float64 `json:"boostFactor"`
    CooldownBlocks       int     `json:"cooldownBlocks"`
    ConsecutiveLimit     int     `json:"consecutiveLimit"`
    SlidingWindowPercent float64 `json:"slidingWindowPercent"`
    DefaultReputation    float64 `json:"defaultReputation"`
}

// GetStats retrieves overall network statistics
func (api *API) GetStats() (NetworkStats, error) {
    validators := api.poi.GetValidators()
    
    stats := NetworkStats{
        TotalValidators: len(validators),
        ActiveValidators: 0,
        CooldownValidators: 0,
        AverageReputation: 0,
        AveragePerformance: 0,
    }
    
    header := api.chain.CurrentHeader()
    currentBlock := uint64(0)
    if header != nil {
        currentBlock = header.Number.Uint64()
    }
    
    totalReputation := 0.0
    totalPerformance := 0.0
    
    api.poi.validatorsMu.RLock()
    for _, validator := range validators {
        state, exists := api.poi.validators[validator]
        if !exists {
            continue
        }
        
        if state.IsActive {
            stats.ActiveValidators++
        }
        
        if state.CooldownUntilBlock > currentBlock {
            stats.CooldownValidators++
        }
        
        totalReputation += api.poi.GetReputation(validator)
        totalPerformance += api.poi.GetPerformance(validator)
    }
    api.poi.validatorsMu.RUnlock()
    
    if len(validators) > 0 {
        stats.AverageReputation = totalReputation / float64(len(validators))
        stats.AveragePerformance = totalPerformance / float64(len(validators))
    }
    
    return stats, nil
}

// NetworkStats represents overall network statistics
type NetworkStats struct {
    TotalValidators      int     `json:"totalValidators"`
    ActiveValidators     int     `json:"activeValidators"`
    CooldownValidators   int     `json:"cooldownValidators"`
    AverageReputation    float64 `json:"averageReputation"`
    AveragePerformance   float64 `json:"averagePerformance"`
}

// UpdatePerformanceMetrics allows manual update of performance metrics (for testing)
func (api *API) UpdatePerformanceMetrics(validator common.Address, latency int64, throughput uint64, availability float64, bandwidth uint64) error {
    api.poi.UpdatePerformanceMetrics(validator, 
        time.Duration(latency),
        throughput,
        availability,
        bandwidth)
    return nil
}

// TriggerDecay manually triggers reputation decay (for testing)
func (api *API) TriggerDecay() error {
    api.poi.DecayAllReputation()
    return nil
}

// AddPenalty adds a penalty to a validator (for testing)
func (api *API) AddPenalty(validator common.Address) error {
    api.poi.validatorsMu.Lock()
    defer api.poi.validatorsMu.Unlock()
    
    state, exists := api.poi.validators[validator]
    if !exists {
        return fmt.Errorf("validator %s not found", validator.Hex())
    }
    
    state.Penalties++
    return nil
}

// SetValidatorActive sets a validator's active status
func (api *API) SetValidatorActive(validator common.Address, active bool) error {
    api.poi.validatorsMu.Lock()
    defer api.poi.validatorsMu.Unlock()
    
    state, exists := api.poi.validators[validator]
    if !exists {
        return fmt.Errorf("validator %s not found", validator.Hex())
    }
    
    state.IsActive = active
    return nil
}

// GetValidatorHistory retrieves historical data for a validator
func (api *API) GetValidatorHistory(validator common.Address, fromBlock, toBlock uint64) (ValidatorHistory, error) {
    api.poi.validatorsMu.RLock()
    state, exists := api.poi.validators[validator]
    api.poi.validatorsMu.RUnlock()
    
    if !exists {
        return ValidatorHistory{}, fmt.Errorf("validator %s not found", validator.Hex())
    }
    
    // This is a simplified version - in practice, you'd need to store historical data
    history := ValidatorHistory{
        Validator:     validator,
        FromBlock:     fromBlock,
        ToBlock:       toBlock,
        BlocksProduced: state.BlocksProduced,
        TotalTransactions: state.TotalTransactions,
        SuccessfulTransactions: state.SuccessfulTx,
        Penalties:     state.Penalties,
        UpTime:        state.UpTime,
    }
    
    return history, nil
}

// ValidatorHistory represents historical data for a validator
type ValidatorHistory struct {
    Validator              common.Address `json:"validator"`
    FromBlock              uint64         `json:"fromBlock"`
    ToBlock                uint64         `json:"toBlock"`
    BlocksProduced         uint64         `json:"blocksProduced"`
    TotalTransactions      uint64         `json:"totalTransactions"`
    SuccessfulTransactions uint64         `json:"successfulTransactions"`
    Penalties              uint64         `json:"penalties"`
    UpTime                 time.Duration  `json:"upTime"`
}

// GetTopValidators retrieves the top N validators by PoI score
func (api *API) GetTopValidators(n int) ([]ValidatorRanking, error) {
    rankings, err := api.GetValidatorRanking()
    if err != nil {
        return nil, err
    }
    
    if n > len(rankings) {
        n = len(rankings)
    }
    
    return rankings[:n], nil
}

// GetEligibleValidators retrieves validators eligible for selection (not in cooldown)
func (api *API) GetEligibleValidators() ([]common.Address, error) {
    header := api.chain.CurrentHeader()
    if header == nil {
        return nil, ErrUnknownBlock
    }
    
    currentBlock := header.Number.Uint64()
    validators := api.poi.GetValidators()
    eligible := make([]common.Address, 0)
    
    api.poi.validatorsMu.RLock()
    for _, validator := range validators {
        state, exists := api.poi.validators[validator]
        if !exists {
            continue
        }
        
        if state.IsActive && state.CooldownUntilBlock <= currentBlock {
            eligible = append(eligible, validator)
        }
    }
    api.poi.validatorsMu.RUnlock()
    
    return eligible, nil
}

type ValidatorFullInfo struct {
    Address           common.Address `json:"address"`
    PoIScore          float64        `json:"poiScore"`
    Reputation        float64        `json:"reputation"`
    Performance       float64        `json:"performance"`
    BlocksProduced    uint64         `json:"blocksProduced"`
    LastActiveBlock   uint64         `json:"lastActiveBlock"`
    ConsecutiveBlocks uint64         `json:"consecutiveBlocks"`
    CooldownUntil     uint64         `json:"cooldownUntilBlock"`
    UpTime            string         `json:"upTime"`
    TotalTx           uint64         `json:"totalTransactions"`
    SuccessfulTx      uint64         `json:"successfulTx"`
    Penalties         uint64         `json:"penalties"`
    IsActive          bool           `json:"isActive"`
}

// RPC: poi_getValidatorFullInfo(address)
func (api *API) GetValidatorFullInfo(addr common.Address) (*ValidatorFullInfo, error) {
    api.poi.validatorsMu.RLock()
    state, exists := api.poi.validators[addr]
    api.poi.validatorsMu.RUnlock()
    if !exists {
        return nil, fmt.Errorf("validator %s not found", addr.Hex())
    }

    score := api.poi.CalculatePoIScore(addr)
    reputation := api.poi.GetReputation(addr)
    performance := api.poi.GetPerformance(addr)

    info := &ValidatorFullInfo{
        Address:           addr,
        PoIScore:          score,
        Reputation:        reputation,
        Performance:       performance,
        BlocksProduced:    state.BlocksProduced,
        LastActiveBlock:   state.LastActiveBlock,
        ConsecutiveBlocks: state.ConsecutiveBlocks,
        CooldownUntil:     state.CooldownUntilBlock,
        UpTime:            state.UpTime.String(),
        TotalTx:           state.TotalTransactions,
        SuccessfulTx:      state.SuccessfulTx,
        Penalties:         state.Penalties,
        IsActive:          state.IsActive,
    }
    return info, nil
}