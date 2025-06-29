// filepath: /consensus/poi/poi_test.go
package poi

import (
    "math/big"
    "testing"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/params"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPoI_New(t *testing.T) {
    config := &params.PoIConfig{
        Period: 15,
    }
    
    poi := New(config, nil)
    
    assert.NotNil(t, poi)
    assert.Equal(t, config, poi.config)
    assert.Equal(t, 0.6, poi.alpha)
    assert.Equal(t, 0.4, poi.beta)
    assert.NotNil(t, poi.validators)
    assert.NotNil(t, poi.reputationStore)
    assert.NotNil(t, poi.performanceStore)
}

func TestPoI_InitializeValidator(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    poi.initializeValidator(validator, 100)
    
    // Check validator state
    state, exists := poi.validators[validator]
    require.True(t, exists)
    assert.Equal(t, validator, state.Address)
    assert.Equal(t, uint64(100), state.LastActiveBlock)
    assert.True(t, state.IsActive)
    
    // Check reputation
    reputation, exists := poi.reputationStore[validator]
    require.True(t, exists)
    assert.Equal(t, DefaultReputation, reputation)
    
    // Check performance metrics
    metrics, exists := poi.performanceStore[validator]
    require.True(t, exists)
    assert.NotNil(t, metrics)
}

func TestPoI_CalculatePoIScore(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    poi.initializeValidator(validator, 100)
    
    score := poi.CalculatePoIScore(validator)
    
    // Should be > 0 and <= 1
    assert.Greater(t, score, 0.0)
    assert.LessOrEqual(t, score, 1.0)
}

func TestPoI_GetReputation(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    // Test new validator
    reputation := poi.GetReputation(validator)
    assert.Equal(t, DefaultReputation, reputation)
    
    // Initialize validator with some stats
    poi.initializeValidator(validator, 100)
    state := poi.validators[validator]
    state.BlocksProduced = 100
    state.TotalTransactions = 1000
    state.SuccessfulTx = 950
    state.StartTime = time.Now().Add(-time.Hour)
    state.UpTime = 55 * time.Minute // 55 minutes out of 60
    
    reputation = poi.GetReputation(validator)
    assert.Greater(t, reputation, 0.0)
    assert.LessOrEqual(t, reputation, 1.5) // Can be > 1 due to boost
}

func TestPoI_GetPerformance(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    // Test validator without metrics
    performance := poi.GetPerformance(validator)
    assert.Equal(t, 0.5, performance)
    
    // Add performance metrics
    poi.UpdatePerformanceMetrics(validator, 
        100*time.Millisecond, // Low latency
        500,                  // Medium throughput
        0.95,                 // High availability
        50*1024*1024)         // Medium bandwidth
    
    performance = poi.GetPerformance(validator)
    assert.Greater(t, performance, 0.0)
    assert.LessOrEqual(t, performance, 1.0)
}

func TestPoI_SelectValidator(t *testing.T) {
    poi := New(nil, nil)
    
    // Initialize multiple validators
    validators := []common.Address{
        common.HexToAddress("0x1111111111111111111111111111111111111111"),
        common.HexToAddress("0x2222222222222222222222222222222222222222"),
        common.HexToAddress("0x3333333333333333333333333333333333333333"),
    }
    
    for i, validator := range validators {
        poi.initializeValidator(validator, uint64(100+i))
        
        // Give different reputation scores
        poi.reputationStore[validator] = 0.5 + float64(i)*0.2
        
        // Add performance metrics
        poi.UpdatePerformanceMetrics(validator, 
            time.Duration(100+i*50)*time.Millisecond,
            uint64(500+i*100),
            0.9+float64(i)*0.03,
            uint64(50+i*10)*1024*1024)
    }
    
    // Select validator
    selected, err := poi.SelectValidator(200)
    require.NoError(t, err)
    
    // Should be one of our validators
    found := false
    for _, validator := range validators {
        if validator == selected {
            found = true
            break
        }
    }
    assert.True(t, found)
}

func TestPoI_CooldownMechanism(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    poi.initializeValidator(validator, 100)
    
    // Simulate consecutive block production
    for i := 0; i < ConsecutiveLimit; i++ {
        poi.updateValidatorSelection(validator, uint64(100+i))
    }
    
    state := poi.validators[validator]
    assert.Equal(t, uint64(100+ConsecutiveLimit+CooldownBlocks-1), state.CooldownUntilBlock)
    
    // Should not be selectable during cooldown
    selected, err := poi.SelectValidator(uint64(100 + ConsecutiveLimit + 1))
    if err == nil {
        assert.NotEqual(t, validator, selected)
    }
}

func TestPoI_DecayAllReputation(t *testing.T) {
    poi := New(nil, nil)
    
    validators := []common.Address{
        common.HexToAddress("0x1111111111111111111111111111111111111111"),
        common.HexToAddress("0x2222222222222222222222222222222222222222"),
    }
    
    originalReputations := make(map[common.Address]float64)
    for _, validator := range validators {
        poi.initializeValidator(validator, 100)
        poi.reputationStore[validator] = 0.8
        originalReputations[validator] = 0.8
    }
    
    poi.DecayAllReputation()
    
    for _, validator := range validators {
        newRep := poi.reputationStore[validator]
        expectedRep := originalReputations[validator] * DecayFactor
        assert.Equal(t, expectedRep, newRep)
    }
}

func TestPoI_UpdatePerformanceMetrics(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    // Initial update
    poi.UpdatePerformanceMetrics(validator, 
        100*time.Millisecond,
        1000,
        0.99,
        100*1024*1024)
    
    metrics := poi.performanceStore[validator]
    assert.Equal(t, 100*time.Millisecond, metrics.Latency)
    assert.Equal(t, uint64(1000), metrics.Throughput)
    assert.Equal(t, 0.99, metrics.Availability)
    assert.Equal(t, uint64(100*1024*1024), metrics.Bandwidth)
    
    // Second update (should use moving average)
    poi.UpdatePerformanceMetrics(validator,
        200*time.Millisecond,
        800,
        0.95,
        80*1024*1024)
    
    metrics = poi.performanceStore[validator]
    // Should be between original and new values due to moving average
    assert.Greater(t, metrics.Latency, 100*time.Millisecond)
    assert.Less(t, metrics.Latency, 200*time.Millisecond)
}

func TestPoI_VerifyHeader(t *testing.T) {
    poi := New(nil, nil)
    
    // Create a test header
    header := &types.Header{
        Number:    big.NewInt(100),
        Time:      uint64(time.Now().Unix()),
        Coinbase:  common.HexToAddress("0x1234567890123456789012345678901234567890"),
        Extra:     make([]byte, 65), // Space for signature
    }
    
    // Initialize the validator
    poi.initializeValidator(header.Coinbase, 100)
    
    // Verify header should pass
    err := poi.verifyHeader(nil, header, nil)
    assert.NoError(t, err)
    
    // Test with invalid timestamp
    header.Time = 0
    err = poi.verifyHeader(nil, header, nil)
    assert.Error(t, err)
}

func TestPoI_BoostMechanism(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    poi.initializeValidator(validator, 100)
    
    // New validator should get boost
    reputation := poi.GetReputation(validator)
    expectedWithBoost := DefaultReputation * BoostFactor
    assert.InDelta(t, expectedWithBoost, reputation, 0.01)
    
    // Simulate many blocks to remove boost
    state := poi.validators[validator]
    state.BlocksProduced = BoostEpoch * DecayEpochSize
    
    reputation = poi.GetReputation(validator)
    // Should be less than boosted value
    assert.Less(t, reputation, expectedWithBoost)
}

func TestPoI_SlidingWindowSelection(t *testing.T) {
    poi := New(nil, nil)
    
    // Create 10 validators with different scores
    validators := make([]common.Address, 10)
    for i := 0; i < 10; i++ {
        validators[i] = common.HexToAddress(fmt.Sprintf("0x%040d", i+1))
        poi.initializeValidator(validators[i], 100)
        
        // Give increasing reputation scores
        poi.reputationStore[validators[i]] = 0.1 + float64(i)*0.1
    }
    
    // Select validator multiple times
    selectedCount := make(map[common.Address]int)
    for i := 0; i < 100; i++ {
        selected, err := poi.SelectValidator(uint64(200 + i))
        require.NoError(t, err)
        selectedCount[selected]++
    }
    
    // Top 40% should be selected more often
    topValidators := validators[6:] // Top 4 validators (40% of 10)
    topSelections := 0
    for _, validator := range topValidators {
        topSelections += selectedCount[validator]
    }
    
    // At least 50% of selections should be from top validators
    assert.Greater(t, topSelections, 50)
}

func TestPoI_PenaltySystem(t *testing.T) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    
    poi.initializeValidator(validator, 100)
    
    // Get initial reputation
    initialRep := poi.GetReputation(validator)
    
    // Add penalties
    state := poi.validators[validator]
    state.Penalties = 2
    
    // Reputation should decrease
    newRep := poi.GetReputation(validator)
    assert.Less(t, newRep, initialRep)
}

// Benchmark tests
func BenchmarkPoI_CalculatePoIScore(b *testing.B) {
    poi := New(nil, nil)
    validator := common.HexToAddress("0x1234567890123456789012345678901234567890")
    poi.initializeValidator(validator, 100)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        poi.CalculatePoIScore(validator)
    }
}

func BenchmarkPoI_SelectValidator(b *testing.B) {
    poi := New(nil, nil)
    
    // Initialize 50 validators
    for i := 0; i < 50; i++ {
        validator := common.HexToAddress(fmt.Sprintf("0x%040d", i+1))
        poi.initializeValidator(validator, 100)
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        poi.SelectValidator(uint64(200 + i))
    }
}