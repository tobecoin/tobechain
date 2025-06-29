package poi

import (
    "sort"
    "sync"
    "syscall"
    "time"

    "github.com/ethereum/go-ethereum/core"
    "github.com/ethereum/go-ethereum/core/txpool"
    "github.com/ethereum/go-ethereum/core/vm"
    "github.com/ethereum/go-ethereum/core/state"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethdb"
    "github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/params"
    "github.com/ethereum/go-ethereum/rlp"
    "github.com/ethereum/go-ethereum/rpc"
    "github.com/ethereum/go-ethereum/trie"
    "github.com/hashicorp/golang-lru/v2/expirable"
    "golang.org/x/term"
)

const (
    DecayEpochSize       = 1000
    DecayFactor          = 0.7
    BoostEpoch           = 3
    BoostFactor          = 1.1
    CooldownBlocks       = 10
    ConsecutiveLimit     = 10
    SlidingWindowPercent = 0.4
    DefaultReputation    = 0.5
    MaxValidators        = 100

    LatencyWeight      = 0.25
    ThroughputWeight   = 0.25
    AvailabilityWeight = 0.25
    BandwidthWeight    = 0.25

    BlockScoreWeight = 0.4
    UptimeWeight     = 0.3
    TxSuccessWeight  = 0.3

    inmemorySnapshots  = 128
    inmemorySignatures = 4096
)

// ErrUnknownBlock is returned when the list of validators is requested for a block
// that is not part of the local blockchain.
var ErrUnknownBlock = errors.New("unknown block")

// ErrMissingSignature is returned if a block's extra-data section doesn't seem
// to contain a 65 byte secp256k1 signature.
var ErrMissingSignature = errors.New("extra-data 65 byte signature suffix missing")

// ErrInvalidSignature is returned if a block's signature doesn't match the expected one.
var ErrInvalidSignature = errors.New("invalid signature")

type MinerNotification interface {
    TriggerMining(blockNumber uint64) error
    IsActive() bool
}

type Node struct {
    ID      common.Address `json:"id"`
    Address common.Address `json:"address"`
}

type SignerFn func(signer common.Address, mimeType string, message []byte) ([]byte, error)

func New(config *params.PoIConfig, db ethdb.Database) *PoI {
    if config == nil {
        config = &params.PoIConfig{Period: 2}
    }
    mathrand.Seed(time.Now().UnixNano())

    recents := expirable.NewLRU[common.Hash, *Snapshot](inmemorySnapshots, nil, time.Hour)
    signatures := expirable.NewLRU[common.Hash, common.Address](inmemorySignatures, nil, time.Hour)

    poi := &PoI{
        config:     config,
        db:         db,
        recents:    recents,
        signatures: signatures,

        validators:       make(map[common.Address]*ValidatorState),
        reputationStore:  make(map[common.Address]float64),
        performanceStore: make(map[common.Address]*PerformanceMetrics),

        alpha: 0.6,
        beta:  0.4,
    }

    log.Info("PoI consensus engine initialized with self-mining",
        "period", config.Period,
        "alpha", poi.alpha,
        "beta", poi.beta,
        "emptyBlocks", poi.ShouldCreateEmptyBlocks(),
        "selfMining", true)

    return poi
}

func (poi *PoI) ShouldCreateEmptyBlocks() bool {
    return true
}

func (poi *PoI) GetMiningConfiguration() map[string]interface{} {
    return map[string]interface{}{
        "period":           poi.config.Period,
        "emptyBlocks":      poi.ShouldCreateEmptyBlocks(),
        "continuousMining": true,
        "validatorCount":   len(poi.GetValidators()),
        "miningReady":      poi.IsReadyToMine() == nil,
        "currentBlockInterval": poi.GetcurrentBlockInterval(),
    }
}

func (poi *PoI) InitializeFromGenesis(genesisValidator common.Address) {
    if genesisValidator == (common.Address{}) {
        // log.Warn("Genesis validator address is empty")
        return
    }
    poi.validatorsMu.Lock()
    poi.validators[genesisValidator] = &ValidatorState{
        Address:           genesisValidator,
        BlocksProduced:    0,
        LastActiveBlock:   0,
        ConsecutiveBlocks: 0,
        CooldownUntilBlock: 0,
        UpTime:            0,
        StartTime:         time.Now(),
        TotalTransactions: 0,
        SuccessfulTx:      0,
        Penalties:         0,
        IsActive:          true,
    }
    poi.validatorsMu.Unlock()

    poi.scoreMu.Lock()
    poi.reputationStore[genesisValidator] = DefaultReputation
    poi.performanceStore[genesisValidator] = &PerformanceMetrics{
        Latency:      time.Second,
        Throughput:   100,
        Availability: 1.0,
        Bandwidth:    10 * 1024 * 1024,
        LastUpdated:  time.Now(),
    }
    poi.scoreMu.Unlock()

    poi.lock.Lock()
    if poi.signer == (common.Address{}) {
        poi.signer = genesisValidator
        log.Info("Set genesis validator as default signer", "signer", genesisValidator.Hex())
    }
    if poi.signFn == nil {
        log.Info("Creating signing function for genesis validator...")
        privateKey, err := crypto.GenerateKey()
        if err != nil {
            log.Error("Failed to generate private key for genesis", "error", err)
        } else {
            poi.signFn = poi.AutoGenerateSignFn(privateKey)
            log.Info("Auto-generated signing function for genesis validator",
                "validator", genesisValidator.Hex())
        }
    }
    poi.lock.Unlock()

    signer, hasSignFn, ready := poi.GetAuthorizationStatus()
    log.Info("Genesis validator initialized and set as signer",
        "validator", genesisValidator.Hex(),
        "signer", signer.Hex(),
        "hasSignFn", hasSignFn,
        "ready", ready)
}

func (poi *PoI) ForcecurrentBlockPreparation(chain consensus.ChainHeaderReader, blockchain *core.BlockChain) {
    if chain == nil || blockchain == nil {
        log.Error("ForcecurrentBlockPreparation: chain or blockchain is nil")
        return
    }
    parentHeader := chain.CurrentHeader()
    if parentHeader == nil {
        log.Error("Current header not found, cannot continue mining")
        return
    }
    currentBlock := parentHeader.Number.Uint64()
    currentBlock := currentBlock + 1

    // Ngăn tạo block nếu đã tồn tại
    if blockchain.GetBlockByNumber(currentBlock) != nil {
        return
    }

    if err := poi.IsReadyToMine(); err != nil {
        log.Error("Not ready for next block", "error", err)
        return
    }
    validators := poi.GetValidators()
    if len(validators) == 0 {
        log.Error("No validators for next block")
        return
    }

    parentHeader = chain.GetHeaderByNumber(currentBlock)
    if parentHeader == nil {
        log.Error("Parent block not found, cannot continue mining", "blockNumber", currentBlock)
        return
    }

    header := &types.Header{
        Number:     big.NewInt(int64(currentBlock)),
        ParentHash: parentHeader.Hash(),
        Extra:      make([]byte, 65),
        Time:       uint64(time.Now().Unix()),
        Difficulty: big.NewInt(1),
        GasLimit:   uint64(10000000),
    }
    validator, err := poi.SelectValidator(currentBlock)
    if err != nil {
        log.Error("Failed to select validator", "error", err)
        return
    }
    header.Coinbase = validator
    if header.BaseFee == nil {
        header.BaseFee = big.NewInt(1000000000)
    }
    if header.Difficulty == nil {
        header.Difficulty = big.NewInt(1)
    }
    if header.Number == nil {
        header.Number = big.NewInt(int64(currentBlock))
    }

    var txs []*types.Transaction
    if poi.txpool != nil {
        pendingTxs := poi.txpool.Pending(txpool.PendingFilter{})
        for _, addrTxs := range pendingTxs {
            for _, lazyTx := range addrTxs {
                txs = append(txs, lazyTx.Tx)
            }
        }
    }

    parentState, err := blockchain.StateAt(parentHeader.Root)
    if err != nil {
        log.Error("Failed to create state from parent", "error", err)
        return
    }
    state := parentState.Copy()

    receipts := make([]*types.Receipt, 0, len(txs))
    gasPool := core.GasPool(header.GasLimit)
    gasUsed := uint64(0)

    for _, tx := range txs {
        if header.BaseFee == nil {
            header.BaseFee = big.NewInt(1000000000)
        }
        if header.Difficulty == nil {
            header.Difficulty = big.NewInt(1)
        }
        if header.Number == nil {
            header.Number = big.NewInt(int64(currentBlock))
        }

        receipt, err := core.ApplyTransaction(
            blockchain.Config(),
            blockchain,
            &header.Coinbase,
            &gasPool,
            state,
            header,
            tx,
            &gasUsed,
            vm.Config{},
        )
        if err != nil {
            log.Error("Failed to apply transaction", "error", err)
            continue
        }
        receipts = append(receipts, receipt)
    }
    header.GasUsed = gasUsed

    block, err := poi.FinalizeAndAssemble(chain, header, state, txs, nil, receipts, nil)
    if err != nil {
        log.Error("Failed to assemble block", "error", err)
        return
    }

    resultCh := make(chan *types.Block, 1)
    stopCh := make(chan struct{})

    if err := poi.Seal(chain, block, resultCh, stopCh); err != nil {
        log.Error("Failed to seal block", "error", err)
        return
    }
    sealedBlock := <-resultCh

    if _, err := blockchain.InsertChain([]*types.Block{sealedBlock}); err != nil {
        log.Error("Failed to insert sealed block into chain", "error", err)
        return
    }
    log.Info("Block sealed and inserted", "number", sealedBlock.NumberU64())
}

func (poi *PoI) GetcurrentBlockInterval() time.Duration {
    return time.Duration(poi.config.Period) * time.Second
}

func PoIRLP(header *types.Header) []byte {
    b := make([]byte, len(header.Extra)-65)
    copy(b, header.Extra[:len(header.Extra)-65])
    h := &types.Header{
        ParentHash:  header.ParentHash,
        UncleHash:   header.UncleHash,
        Coinbase:    header.Coinbase,
        Root:        header.Root,
        TxHash:      header.TxHash,
        ReceiptHash: header.ReceiptHash,
        Bloom:       header.Bloom,
        Difficulty:  header.Difficulty,
        Number:      header.Number,
        GasLimit:    header.GasLimit,
        GasUsed:     header.GasUsed,
        Time:        header.Time,
        Extra:       b,
        MixDigest:   header.MixDigest,
        Nonce:       header.Nonce,
    }
    return rlpHash(h).Bytes()
}

func SealHash(header *types.Header) common.Hash {
    return rlpHash([]interface{}{
        header.ParentHash,
        header.UncleHash,
        header.Coinbase,
        header.Root,
        header.TxHash,
        header.ReceiptHash,
        header.Bloom,
        header.Difficulty,
        header.Number,
        header.GasLimit,
        header.GasUsed,
        header.Time,
        header.Extra[:len(header.Extra)-65],
        header.MixDigest,
        header.Nonce,
    })
}

func rlpHash(x interface{}) (h common.Hash) {
    hash := crypto.Keccak256Hash(mustRlpEncode(x))
    return hash
}

func mustRlpEncode(x interface{}) []byte {
    b, err := rlp.EncodeToBytes(x)
    if err != nil {
        panic(fmt.Sprintf("RLP encoding failed: %v", err))
    }
    return b
}

func (poi *PoI) Author(header *types.Header) (common.Address, error) {
    return header.Coinbase, nil
}

func (poi *PoI) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
    return poi.verifyHeader(chain, header, nil)
}

func (poi *PoI) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
    abort := make(chan struct{})
    results := make(chan error, len(headers))
    go func() {
        for i, header := range headers {
            err := poi.verifyHeader(chain, header, headers[:i])
            select {
            case <-abort:
                return
            case results <- err:
            }
        }
    }()
    return abort, results
}

func (poi *PoI) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
    if header.Number == nil {
        return errors.New("header number is nil")
    }
    number := header.Number.Uint64()
    if number == 0 {
        return nil
    }
    if header.Time <= 0 {
        return errors.New("invalid timestamp")
    }
    if err := poi.verifyValidator(chain, header); err != nil {
        return err
    }
    return nil
}

func (poi *PoI) verifyValidator(chain consensus.ChainHeaderReader, header *types.Header) error {
    validator := header.Coinbase
    poi.validatorsMu.RLock()
    state, exists := poi.validators[validator]
    poi.validatorsMu.RUnlock()
    if !exists {
        poi.initializeValidator(validator, header.Number.Uint64())
        return nil
    }
    if state.CooldownUntilBlock > header.Number.Uint64() {
        return fmt.Errorf("validator %s is in cooldown until block %d",
            validator.Hex(), state.CooldownUntilBlock)
    }
    return nil
}

func (poi *PoI) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
    if chain == nil {
        log.Error("Prepare: chain header reader is nil")
        return errors.New("chain header reader is nil")
    }
    blockNumber := header.Number.Uint64()
    log.Debug("=== Prepare Called ===",
        "number", blockNumber,
        "timestamp", time.Now().Unix(),
        "parentHash", header.ParentHash.Hex()[:10]+"...")

    if blockNumber == 1 {
        poi.validatorsMu.RLock()
        hasValidators := len(poi.validators) > 0
        poi.validatorsMu.RUnlock()
        if !hasValidators {
            genesisValidator := header.Coinbase
            if genesisValidator != (common.Address{}) {
                log.Info("Setting up genesis validator from coinbase", "validator", genesisValidator.Hex())
                poi.InitializeFromGenesis(genesisValidator)
                poi.lock.RLock()
                currentSigner := poi.signer
                poi.lock.RUnlock()
                if currentSigner == (common.Address{}) {
                    poi.lock.Lock()
                    poi.signer = genesisValidator
                    poi.lock.Unlock()
                    log.Info("Set genesis validator as signer", "signer", genesisValidator.Hex())
                }
            }
        }
    }
    log.Debug("Preparing continuous mining",
        "blockNumber", blockNumber,
        "hasValidators", len(poi.GetValidators()) > 0)

    validator, err := poi.SelectValidator(blockNumber)
    if err != nil {
        log.Error("Failed to select validator", "error", err)
        return err
    }
    header.Coinbase = validator
    header.Nonce = types.BlockNonce{}
    header.MixDigest = common.Hash{}
    if len(header.Extra) < 65 {
        newExtra := make([]byte, 65)
        copy(newExtra, header.Extra)
        header.Extra = newExtra
        log.Debug("Extended header.Extra for signature", "blockNumber", blockNumber, "newLen", 65)
    }
    // Track parent's time in a variable to avoid using 'parent' directly
    var parentTime uint64
    if chain == nil {
        log.Warn("No chain provided; skipping parent header lookup")
    } else {
        parent := chain.GetHeader(header.ParentHash, blockNumber-1)
        if parent == nil {
            return fmt.Errorf("parent header not found for block %d", blockNumber)
        }
        parentTime = parent.Time
        minTime := parentTime + poi.config.Period
        currentTime := uint64(time.Now().Unix())
        if minTime > currentTime {
            header.Time = minTime
            log.Debug("Set future timestamp for period compliance",
                "blockNumber", blockNumber,
                "time", minTime,
                "delay", minTime-currentTime,
                "period", poi.config.Period)
        } else {
            header.Time = currentTime
            log.Debug("Set current timestamp", "blockNumber", blockNumber, "time", currentTime)
        }
    }
    // Use parentTime instead of parent.Time for logs
    log.Debug("=== Block Prepared Successfully ===",
        "validator", header.Coinbase.Hex(),
        "number", blockNumber,
        "parentTime", parentTime,
        "blockTime", header.Time,
        "timeDiff", header.Time - parentTime,
        "readyToSeal", poi.IsReadyToMine() == nil)

    return nil
}

func (poi *PoI) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB,
    txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
    poi.updateValidatorStateSimple(header.Coinbase, header.Number.Uint64(), len(txs))
}

func (poi *PoI) FinalizeAndAssemble(
    chain consensus.ChainHeaderReader,
    header *types.Header,
    state *state.StateDB,
    txs []*types.Transaction,
    uncles []*types.Header,
    receipts []*types.Receipt,
    withdrawals []*types.Withdrawal) (*types.Block, error) {

    // Đảm bảo các trường không nil trước khi tạo block
    if header.Difficulty == nil {
        header.Difficulty = big.NewInt(1)
    }
    if header.BaseFee == nil {
        header.BaseFee = big.NewInt(1000000000)
    }

    header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
    return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil)), nil
}

func (poi *PoI) Seal(
    chain consensus.ChainHeaderReader,
    block *types.Block,
    results chan<- *types.Block,
    stop <-chan struct{}) error {

    header := block.Header()
    if header.Difficulty == nil {
        header.Difficulty = big.NewInt(1)
    }
    if header.BaseFee == nil {
        header.BaseFee = big.NewInt(1000000000)
    }

    poi.lock.RLock()
    signer, signFn := poi.signer, poi.signFn
    poi.lock.RUnlock()

    validators := poi.GetValidators()
    if signer == (common.Address{}) {
        return errors.New("no signer configured")
    }
    if signFn == nil {
        return errors.New("signing function not set")
    }
    if signer != header.Coinbase {
        return fmt.Errorf("validator not allowed to seal block - signer=%s, coinbase=%s",
            signer.Hex(), header.Coinbase.Hex())
    }
    if len(validators) > 1 {
        if err := poi.checkRecentSignerConstraints(signer, header.Number.Uint64()); err != nil {
            return err
        }
    }
    delay := poi.calculateSealingDelay(header, signer, len(validators))

    go func() {
        if delay > 0 {
            select {
            case <-stop:
                return
            case <-time.After(delay):
            }
        }
        sighash, err := signFn(signer, "application/x-ethereum-block", PoIRLP(header))
        if err != nil {
            log.Error("Failed to sign block", "number", header.Number.Uint64(), "error", err)
            return
        }
        if len(sighash) != 65 {
            log.Error("Invalid signature length", "number", header.Number.Uint64(), "length", len(sighash))
            return
        }
        copy(header.Extra[len(header.Extra)-65:], sighash)

        select {
        case results <- block.WithSeal(header):
            log.Info("Block sealed successfully", "number", header.Number.Uint64(), "validator", signer.Hex())
            poi.handlePostSealSelfMining(header.Number.Uint64(), header)
        case <-stop:
            return
        default:
        }
    }()
    return nil
}

func (poi *PoI) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
    return big.NewInt(1)
}

func (poi *PoI) SealHash(header *types.Header) common.Hash {
    return SealHash(header)
}

func (poi *PoI) Close() error {
    return nil
}

func (poi *PoI) APIs(chain consensus.ChainHeaderReader) []rpc.API {
    poi.SetChain(chain)
    return []rpc.API{{
        Namespace: "poi",
        Version:   "1.0",
        Service:   &API{chain: chain, poi: poi},
        Public:    true,
    }}
}

func (poi *PoI) updateValidatorStateSimple(validator common.Address, blockNumber uint64, txCount int) {
    poi.validatorsMu.Lock()
    defer poi.validatorsMu.Unlock()
    state, exists := poi.validators[validator]
    if !exists {
        state = poi.initializeValidatorState(validator, blockNumber)
    }
    state.BlocksProduced++
    state.LastActiveBlock = blockNumber
    state.TotalTransactions += uint64(txCount)
    state.SuccessfulTx += uint64(txCount)
    now := time.Now()
    if state.StartTime.IsZero() {
        state.StartTime = now
    }
    state.UpTime = now.Sub(state.StartTime)
    poi.validators[validator] = state
}

func (poi *PoI) IsReadyToMine() error {
    poi.lock.RLock()
    defer poi.lock.RUnlock()

    if poi.signer == (common.Address{}) {
        return errors.New("no signer configured - call Authorize() first")
    }

    if poi.signFn == nil {
        return errors.New("no signing function configured - call Authorize() with signFn")
    }

    return nil
}

func (poi *PoI) GetAuthorizationStatus() (signer common.Address, hasSignFn bool, ready bool) {
    poi.lock.RLock()
    defer poi.lock.RUnlock()

    signer = poi.signer
    hasSignFn = poi.signFn != nil
    ready = signer != (common.Address{}) && hasSignFn

    return
}

func (poi *PoI) DebugAuthorizationStatus() {
    signer, hasSignFn, ready := poi.GetAuthorizationStatus()

    log.Info("Authorization Status Debug",
        "signer", signer.Hex(),
        "hasSignFn", hasSignFn,
        "ready", ready,
        "error", poi.IsReadyToMine())
}

func (poi *PoI) DebugMiningStatus(blockNumber uint64) {
    signer, hasSignFn, ready := poi.GetAuthorizationStatus()
    validators := poi.GetValidators()

    log.Info("=== Mining Status Debug ===",
        "blockNumber", blockNumber,
        "signer", signer.Hex(),
        "hasSignFn", hasSignFn,
        "ready", ready,
        "validatorCount", len(validators),
        "canContinue", ready && len(validators) > 0)

    if len(validators) > 0 {
        for i, v := range validators {
            poi.validatorsMu.RLock()
            state := poi.validators[v]
            poi.validatorsMu.RUnlock()

            log.Debug("Validator status",
                "index", i,
                "validator", v.Hex(),
                "active", state.IsActive,
                "cooldownUntil", state.CooldownUntilBlock,
                "consecutive", state.ConsecutiveBlocks,
                "lastActive", state.LastActiveBlock)
        }
    }
}

func (poi *PoI) CheckMiningContinuity(currentBlock uint64) {
    log.Info("=== Mining Continuity Check ===",
        "currentBlock", currentBlock,
        "expectedNext", currentBlock+1)

    validators := poi.GetValidators()
    log.Info("Active validators check", "count", len(validators))

    signer, hasSignFn, ready := poi.GetAuthorizationStatus()
    log.Info("Authorization status check",
        "signer", signer.Hex(),
        "hasSignFn", hasSignFn,
        "ready", ready)

    poi.validatorsMu.RLock()
    cooldownCount := 0
    activeCount := 0
    for addr, state := range poi.validators {
        if state.IsActive {
            activeCount++
            if state.CooldownUntilBlock > currentBlock {
                cooldownCount++
                log.Debug("Validator in cooldown",
                    "validator", addr.Hex(),
                    "cooldownUntil", state.CooldownUntilBlock,
                    "currentBlock", currentBlock)
            }
        }
    }
    poi.validatorsMu.RUnlock()

    log.Info("Validator status summary",
        "active", activeCount,
        "inCooldown", cooldownCount,
        "available", activeCount-cooldownCount)

    if ready && len(validators) > 0 && (activeCount-cooldownCount) > 0 {
        log.Info("✅ Mining should continue normally")
    } else {
        log.Warn("⚠️ Mining may be blocked",
            "ready", ready,
            "validators", len(validators),
            "available", activeCount-cooldownCount)
    }
}