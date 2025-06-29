package poi

import (
    "bytes"
    "encoding/json"
    "errors"
    "sort"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethdb"
    "github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/params"
)

const (
    checkpointInterval = 1024 // Number of blocks after which to save the snapshot to the database
)

var (
    // snapshotKey is the key for storing snapshots in the database
    snapshotKey = []byte("poi-snapshot")
    
    // Define our own error constants since consensus.ErrInvalidChain is not available
    errInvalidChain = errors.New("invalid chain")
    errUnauthorized = errors.New("unauthorized validator")
)

// Snapshot is the state of the authorization voting at a given point in time
type Snapshot struct {
    config            *params.PoIConfig                   // Consensus engine parameters to fine tune behavior
    Number            uint64                             `json:"number"`     // Block number where the snapshot was created
    Hash              common.Hash                        `json:"hash"`       // Block hash where the snapshot was created
    ValidatorSet      map[common.Address]bool            `json:"validators"` // Set of authorized validators at this moment
    Recents           map[uint64]common.Address          `json:"recents"`    // Set of recent validators for spam protections
    
    // PoI specific fields
    ReputationScores  map[common.Address]float64         `json:"reputation_scores"`  // Reputation scores for each validator
    PerformanceScores map[common.Address]float64         `json:"performance_scores"` // Performance scores for each validator
    ValidatorStates   map[common.Address]*ValidatorState `json:"validator_states"`   // Detailed validator states
    Epoch             uint64                             `json:"epoch"`              // Current epoch number
    LastDecayBlock    uint64                             `json:"last_decay_block"`   // Last block where reputation decay occurred
}

// validatorsAscending implements the sort interface to allow sorting a list of addresses
type validatorsAscending []common.Address

func (s validatorsAscending) Len() int           { return len(s) }
func (s validatorsAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s validatorsAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// newSnapshot creates a new snapshot with the specified startup parameters
func newSnapshot(config *params.PoIConfig, number uint64, hash common.Hash, validators []common.Address) *Snapshot {
    snap := &Snapshot{
        config:            config,
        Number:            number,
        Hash:              hash,
        ValidatorSet:      make(map[common.Address]bool),
        Recents:           make(map[uint64]common.Address),
        ReputationScores:  make(map[common.Address]float64),
        PerformanceScores: make(map[common.Address]float64),
        ValidatorStates:   make(map[common.Address]*ValidatorState),
        Epoch:             number / config.Epoch,
        LastDecayBlock:    0,
    }
    
    for _, validator := range validators {
        snap.ValidatorSet[validator] = true
        snap.ReputationScores[validator] = 0.5 // Default reputation for new validators
        snap.PerformanceScores[validator] = 0.5 // Default performance for new validators
        snap.ValidatorStates[validator] = &ValidatorState{
            Address:       validator,
            JoinedAtBlock: number,
            Latency:       100.0, // Default 100ms latency
            Throughput:    10.0,  // Default 10 TPS
            Bandwidth:     1.0,   // Default bandwidth score
            IsActive:      true,
            StartTime:     time.Now(),
        }
    }
    
    return snap
}

// loadSnapshot loads an existing snapshot from the database
func loadSnapshot(config *params.PoIConfig, db ethdb.Database, hash common.Hash) (*Snapshot, error) {
    blob, err := db.Get(append(snapshotKey, hash[:]...))
    if err != nil {
        return nil, err
    }
    
    snap := new(Snapshot)
    if err := json.Unmarshal(blob, snap); err != nil {
        return nil, err
    }
    
    snap.config = config
    
    // Initialize maps if they're nil (for backward compatibility)
    if snap.ReputationScores == nil {
        snap.ReputationScores = make(map[common.Address]float64)
    }
    if snap.PerformanceScores == nil {
        snap.PerformanceScores = make(map[common.Address]float64)
    }
    if snap.ValidatorStates == nil {
        snap.ValidatorStates = make(map[common.Address]*ValidatorState)
    }
    
    return snap, nil
}

// store inserts the snapshot into the database
func (s *Snapshot) store(db ethdb.Database) error {
    blob, err := json.Marshal(s)
    if err != nil {
        return err
    }
    return db.Put(append(snapshotKey, s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot
func (s *Snapshot) copy() *Snapshot {
    cpy := &Snapshot{
        config:            s.config,
        Number:            s.Number,
        Hash:              s.Hash,
        ValidatorSet:      make(map[common.Address]bool),
        Recents:           make(map[uint64]common.Address),
        ReputationScores:  make(map[common.Address]float64),
        PerformanceScores: make(map[common.Address]float64),
        ValidatorStates:   make(map[common.Address]*ValidatorState),
        Epoch:             s.Epoch,
        LastDecayBlock:    s.LastDecayBlock,
    }
    
    for validator, authorized := range s.ValidatorSet {
        cpy.ValidatorSet[validator] = authorized
    }
    
    for block, validator := range s.Recents {
        cpy.Recents[block] = validator
    }
    
    for validator, score := range s.ReputationScores {
        cpy.ReputationScores[validator] = score
    }
    
    for validator, score := range s.PerformanceScores {
        cpy.PerformanceScores[validator] = score
    }
    
    for validator, state := range s.ValidatorStates {
        cpy.ValidatorStates[validator] = &ValidatorState{
            Address:            state.Address,
            BlocksProduced:     state.BlocksProduced,
            ConsecutiveBlocks:  state.ConsecutiveBlocks,
            CooldownUntilBlock: state.CooldownUntilBlock,
            LastActiveBlock:    state.LastActiveBlock,
            TotalUptime:        state.TotalUptime,
            SuccessfulTxs:      state.SuccessfulTxs,
            TotalTxs:           state.TotalTxs,
            Latency:            state.Latency,
            Throughput:         state.Throughput,
            Bandwidth:          state.Bandwidth,
            JoinedAtBlock:      state.JoinedAtBlock,
            UpTime:             state.UpTime,
            StartTime:          state.StartTime,
            TotalTransactions:  state.TotalTransactions,
            SuccessfulTx:       state.SuccessfulTx,
            Penalties:          state.Penalties,
            IsActive:           state.IsActive,
        }
    }
    
    return cpy
}

// validators retrieves the list of authorized validators in ascending order
func (s *Snapshot) validators() []common.Address {
    validators := make([]common.Address, 0, len(s.ValidatorSet))
    for validator, authorized := range s.ValidatorSet {
        if authorized {
            validators = append(validators, validator)
        }
    }
    sort.Sort(validatorsAscending(validators))
    return validators
}

// inturn returns if a validator at a given block height is in-turn or not
func (s *Snapshot) inturn(number uint64, validator common.Address) bool {
    validators := s.validators()
    if len(validators) == 0 {
        return false
    }
    offset := (number + 1) % uint64(len(validators))
    return validators[offset] == validator
}

// apply creates a new authorization snapshot by applying the given headers to the original one
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
    // Allow passing in no headers for cleaner code
    if len(headers) == 0 {
        return s, nil
    }
    
    // Sanity check that the headers can be applied
    for i := 0; i < len(headers)-1; i++ {
        if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
            return nil, errInvalidChain
        }
    }
    if headers[0].Number.Uint64() != s.Number+1 {
        return nil, errInvalidChain
    }
    
    // Iterate through the headers and create a new snapshot
    snap := s.copy()
    
    for _, header := range headers {
        number := header.Number.Uint64()
        validator := header.Coinbase
        
        // Update epoch if necessary
        if s.config != nil && s.config.Epoch > 0 {
            snap.Epoch = number / s.config.Epoch
        }
        
        // Remove any votes on checkpoint blocks
        if s.config != nil && s.config.Epoch > 0 && number%s.config.Epoch == 0 {
            snap.Recents = make(map[uint64]common.Address)
        }
        
        // Resolve the authorization key and check against validators
        if !snap.ValidatorSet[validator] {
            return nil, errUnauthorized
        }
        
        // Update validator state
        if state, exists := snap.ValidatorStates[validator]; exists {
            state.BlocksProduced++
            state.LastActiveBlock = number
            state.TotalUptime++
            
            // Check for consecutive blocks
            if len(snap.Recents) > 0 {
                lastValidator := snap.Recents[number-1]
                if lastValidator == validator {
                    state.ConsecutiveBlocks++
                } else {
                    state.ConsecutiveBlocks = 1
                }
            } else {
                state.ConsecutiveBlocks = 1
            }
            
            // Apply cooldown if too many consecutive blocks
            if state.ConsecutiveBlocks >= 10 {
                state.CooldownUntilBlock = number + 10
                state.ConsecutiveBlocks = 0
            }
        } else {
            // Initialize new validator state
            snap.ValidatorStates[validator] = &ValidatorState{
                Address:           validator,
                BlocksProduced:    1,
                ConsecutiveBlocks: 1,
                LastActiveBlock:   number,
                TotalUptime:       1,
                JoinedAtBlock:     number,
                Latency:           100.0,
                Throughput:        10.0,
                Bandwidth:         1.0,
                IsActive:          true,
                StartTime:         time.Now(),
            }
        }
        
        // Update recent validators for spam protection
        snap.Recents[number] = validator
        
        // Reputation decay mechanism
        if s.config != nil && s.config.Epoch > 0 && number%s.config.Epoch == 0 && number > snap.LastDecayBlock {
            snap.applyReputationDecay()
            snap.LastDecayBlock = number
        }
        
        // Update reputation and performance scores
        snap.updateScores(validator, header)
        
        // Delete too old recents
        if limit := uint64(len(snap.ValidatorSet)/2 + 1); number >= limit {
            delete(snap.Recents, number-limit)
        }
    }
    
    snap.Number += uint64(len(headers))
    snap.Hash = headers[len(headers)-1].Hash()
    
    return snap, nil
}

// applyReputationDecay applies the reputation decay mechanism
func (s *Snapshot) applyReputationDecay() {
    decayFactor := 0.7 // Keep 70% of reputation
    
    for validator := range s.ReputationScores {
        s.ReputationScores[validator] *= decayFactor
        // Ensure minimum reputation
        if s.ReputationScores[validator] < 0.1 {
            s.ReputationScores[validator] = 0.1
        }
    }
    
    log.Info("Applied reputation decay", "factor", decayFactor, "validators", len(s.ReputationScores))
}

// updateScores updates reputation and performance scores for a validator
func (s *Snapshot) updateScores(validator common.Address, header *types.Header) {
    state, exists := s.ValidatorStates[validator]
    if !exists {
        return
    }
    
    // Update reputation score based on block production and uptime
    blockScore := float64(state.BlocksProduced) / float64(s.Number+1)
    if blockScore > 1.0 {
        blockScore = 1.0
    }
    
    uptimeScore := float64(state.TotalUptime) / float64(s.Number-state.JoinedAtBlock+1)
    if uptimeScore > 1.0 {
        uptimeScore = 1.0
    }
    
    txSuccessRate := 1.0
    if state.TotalTxs > 0 {
        txSuccessRate = float64(state.SuccessfulTxs) / float64(state.TotalTxs)
    }
    
    // Calculate reputation score
    reputation := 0.4*blockScore + 0.3*uptimeScore + 0.3*txSuccessRate
    
    // Apply boost for new validators
    if s.config != nil && state.BlocksProduced < 3000 { // 3 epochs of 1000 blocks each
        boostFactor := 1.1 // 10% boost
        reputation *= boostFactor
    }
    
    if reputation > 1.0 {
        reputation = 1.0
    }
    
    s.ReputationScores[validator] = reputation
    
    // Update performance score based on latency, throughput, availability, bandwidth
    latencyScore := 1.0 - (state.Latency / 1000.0) // Normalize latency (lower is better)
    if latencyScore < 0 {
        latencyScore = 0
    }
    
    throughputScore := state.Throughput / 100.0 // Normalize throughput
    if throughputScore > 1.0 {
        throughputScore = 1.0
    }
    
    availabilityScore := uptimeScore // Same as uptime score
    
    bandwidthScore := state.Bandwidth
    if bandwidthScore > 1.0 {
        bandwidthScore = 1.0
    }
    
    performance := 0.25*latencyScore + 0.25*throughputScore + 0.25*availabilityScore + 0.25*bandwidthScore
    
    s.PerformanceScores[validator] = performance
}

// calculatePoIScore calculates the combined PoI score for a validator
func (s *Snapshot) calculatePoIScore(validator common.Address, alpha, beta float64) float64 {
    reputation := s.ReputationScores[validator]
    performance := s.PerformanceScores[validator]
    
    return alpha*reputation + beta*performance
}

// getTopValidators returns the top validators based on PoI scores
func (s *Snapshot) getTopValidators(alpha, beta float64, percentage float64) []common.Address {
    type validatorScore struct {
        address common.Address
        score   float64
    }
    
    var scores []validatorScore
    for validator := range s.ValidatorSet {
        if !s.ValidatorSet[validator] {
            continue
        }
        
        // Skip validators in cooldown
        if state, exists := s.ValidatorStates[validator]; exists {
            if s.Number < state.CooldownUntilBlock {
                continue
            }
        }
        
        score := s.calculatePoIScore(validator, alpha, beta)
        scores = append(scores, validatorScore{validator, score})
    }
    
    // Sort by score descending
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].score > scores[j].score
    })
    
    // Return top percentage
    count := int(float64(len(scores)) * percentage)
    if count == 0 && len(scores) > 0 {
        count = 1
    }
    
    result := make([]common.Address, count)
    for i := 0; i < count && i < len(scores); i++ {
        result[i] = scores[i].address
    }
    
    return result
}

// isValidValidator checks if a validator is authorized and not in cooldown
func (s *Snapshot) isValidValidator(validator common.Address, number uint64) bool {
    if !s.ValidatorSet[validator] {
        return false
    }
    
    if state, exists := s.ValidatorStates[validator]; exists {
        return number >= state.CooldownUntilBlock
    }
    
    return true
}

// updateValidatorPerformance updates performance metrics for a validator
func (s *Snapshot) updateValidatorPerformance(validator common.Address, latency, throughput, bandwidth float64) {
    if state, exists := s.ValidatorStates[validator]; exists {
        state.Latency = latency
        state.Throughput = throughput
        state.Bandwidth = bandwidth
    }
}

// updateValidatorTransactions updates transaction statistics for a validator
func (s *Snapshot) updateValidatorTransactions(validator common.Address, successful, total uint64) {
    if state, exists := s.ValidatorStates[validator]; exists {
        state.SuccessfulTxs = successful
        state.TotalTxs = total
    }
}