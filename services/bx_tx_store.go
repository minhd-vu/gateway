package services

import (
	"fmt"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/bloXroute-Labs/gateway/v2"
	log "github.com/bloXroute-Labs/gateway/v2/logger"
	pbbase "github.com/bloXroute-Labs/gateway/v2/protobuf"
	"github.com/bloXroute-Labs/gateway/v2/types"
	"github.com/bloXroute-Labs/gateway/v2/utils"
	"github.com/bloXroute-Labs/gateway/v2/utils/syncmap"
)

// BxTxStore represents the storage of transaction info for a given node
type BxTxStore struct {
	clock         utils.Clock
	hashToContent *syncmap.SyncMap[string, *types.BxTransaction]
	shortIDToHash *syncmap.SyncMap[types.ShortID, types.SHA256Hash]

	seenTxs            HashHistory
	timeToAvoidReEntry time.Duration

	cleanupFreq            time.Duration
	maxTxAge               time.Duration
	noSIDAge               time.Duration
	quit                   chan bool
	lock                   sync.Mutex
	assigner               ShortIDAssigner
	cleanedShortIDsChannel chan types.ShortIDsByNetwork
	bloom                  BloomFilter
}

// NewBxTxStore creates a new BxTxStore to store and processes all relevant transactions
func NewBxTxStore(cleanupFreq time.Duration, maxTxAge time.Duration, noSIDAge time.Duration,
	assigner ShortIDAssigner, seenTxs HashHistory, cleanedShortIDsChannel chan types.ShortIDsByNetwork,
	timeToAvoidReEntry time.Duration, bloom BloomFilter) BxTxStore {
	return newBxTxStore(utils.RealClock{}, cleanupFreq, maxTxAge, noSIDAge, assigner, seenTxs, cleanedShortIDsChannel, timeToAvoidReEntry, bloom)
}

func newBxTxStore(clock utils.Clock, cleanupFreq time.Duration, maxTxAge time.Duration,
	noSIDAge time.Duration, assigner ShortIDAssigner, seenTxs HashHistory, cleanedShortIDsChannel chan types.ShortIDsByNetwork,
	timeToAvoidReEntry time.Duration, bloom BloomFilter) BxTxStore {
	return BxTxStore{
		clock:                  clock,
		hashToContent:          syncmap.NewStringMapOf[*types.BxTransaction](),
		shortIDToHash:          syncmap.NewIntegerMapOf[types.ShortID, types.SHA256Hash](),
		seenTxs:                seenTxs,
		timeToAvoidReEntry:     timeToAvoidReEntry,
		cleanupFreq:            cleanupFreq,
		maxTxAge:               maxTxAge,
		noSIDAge:               noSIDAge,
		quit:                   make(chan bool),
		assigner:               assigner,
		cleanedShortIDsChannel: cleanedShortIDsChannel,
		bloom:                  bloom,
	}
}

// Start initializes all relevant goroutines for the BxTxStore
func (t *BxTxStore) Start() error {
	t.cleanup()
	return nil
}

// Stop closes all running go routines for BxTxStore
func (t *BxTxStore) Stop() {
	t.quit <- true
	<-t.quit
}

// Clear removes all elements from txs and shortIDToHash
func (t *BxTxStore) Clear() {
	t.hashToContent.Clear()
	t.shortIDToHash.Clear()
	log.Debugf("Cleared tx service.")
}

// Count indicates the number of stored transaction in BxTxStore
func (t *BxTxStore) Count() int {
	return t.hashToContent.Size()
}

// remove deletes a single transaction, including its shortIDs
func (t *BxTxStore) remove(hash string, reEntryProtection ReEntryProtectionFlags, reason string) {
	if tx, ok := t.hashToContent.LoadAndDelete(hash); ok {
		bxTransaction := tx
		for _, shortID := range bxTransaction.ShortIDs() {
			t.shortIDToHash.Delete(shortID)
		}
		// if asked, add the hash to the history map so we remember this transaction for some time
		// and prevent if from being added back to the TxStore
		switch reEntryProtection {
		case NoReEntryProtection:
		case ShortReEntryProtection:
			t.seenTxs.Add(hash, ShortReEntryProtectionDuration)
			t.bloom.Add([]byte(hash))
		case FullReEntryProtection:
			t.seenTxs.Add(hash, t.timeToAvoidReEntry)
			t.bloom.Add([]byte(hash))
		default:
			log.Fatalf("unknown reEntryProtection value %v for hash %v", reEntryProtection, hash)
		}
		log.Tracef("TxStore: transaction %v, network %v, shortIDs %v removed (%v). reEntryProtection %v",
			bxTransaction.Hash(), bxTransaction.NetworkNum(), bxTransaction.ShortIDs(), reason, reEntryProtection)
	}
}

// RemoveShortIDs deletes a series of transactions by their short IDs. RemoveShortIDs can take a potentially large short ID array, so it should be passed by reference.
func (t *BxTxStore) RemoveShortIDs(shortIDs *types.ShortIDList, reEntryProtection ReEntryProtectionFlags, reason string) {
	// note - it is OK for hashesToRemove to hold the same hash multiple times.
	hashesToRemove := make(types.SHA256HashList, 0)
	for _, shortID := range *shortIDs {
		if hash, ok := t.shortIDToHash.Load(shortID); ok {
			hashesToRemove = append(hashesToRemove, hash)
		}
	}
	t.RemoveHashes(&hashesToRemove, reEntryProtection, reason)
}

// GetTxByShortID lookup a transaction by its shortID. return error if not found
func (t *BxTxStore) GetTxByShortID(shortID types.ShortID) (*types.BxTransaction, error) {
	if hash, ok := t.shortIDToHash.Load(shortID); ok {
		if tx, exists := t.hashToContent.Load(string(hash[:])); exists {
			return tx, nil
		}
		return nil, fmt.Errorf("transaction content for shortID %v and hash %v does not exist", shortID, hash)
	}
	return nil, fmt.Errorf("transaction with shortID %v does not exist", shortID)
}

// RemoveHashes deletes a series of transactions by their hash from BxTxStore. RemoveHashes can take a potentially large hash array, so it should be passed by reference.
func (t *BxTxStore) RemoveHashes(hashes *types.SHA256HashList, reEntryProtection ReEntryProtectionFlags, reason string) {
	for _, hash := range *hashes {
		t.remove(string(hash[:]), reEntryProtection, reason)
	}
}

// Iter returns a channel iterator for all transactions in BxTxStore
func (t *BxTxStore) Iter() (iter <-chan *types.BxTransaction) {
	newChan := make(chan *types.BxTransaction)
	go func() {

		t.hashToContent.Range(func(key string, bxTransaction *types.BxTransaction) bool {
			if t.clock.Now().Sub(bxTransaction.AddTime()) < t.maxTxAge {
				newChan <- bxTransaction
			}
			return true
		})

		close(newChan)
	}()
	return newChan
}

// Add adds a new transaction to BxTxStore
func (t *BxTxStore) Add(hash types.SHA256Hash, content types.TxContent, shortID types.ShortID, networkNum types.NetworkNum,
	_ bool, flags types.TxFlags, timestamp time.Time, _ int64, sender types.Sender) TransactionResult {
	if shortID == types.ShortIDEmpty && len(content) == 0 {
		debug.PrintStack()
		panic("Bad usage of Add function - content and shortID can't be both missing")
	}
	result := TransactionResult{}
	if t.clock.Now().Sub(timestamp) > t.maxTxAge {
		result.Transaction = types.NewBxTransaction(hash, networkNum, flags, timestamp)
		result.DebugData = fmt.Sprintf("Transaction is too old - %v", timestamp)
		return result
	}

	hashStr := string(hash[:])
	// if the hash is in history we treat it as IgnoreSeen
	if t.refreshSeenTx(hash) {
		// if the hash is in history, but we get a shortID for it, it means that the hash was not in the ATR history
		// and some GWs may get and use this shortID. In such a case we should remove the hash from history and allow
		// it to be added to the TxStore
		if shortID != types.ShortIDEmpty {
			t.seenTxs.Remove(hashStr)
		} else {
			result.Transaction = types.NewBxTransaction(hash, networkNum, flags, timestamp)
			result.DebugData = "Transaction already seen and deleted from store"
			result.AlreadySeen = true
			return result
		}
	}

	bxTransaction := types.NewBxTransaction(hash, networkNum, flags, timestamp)

	if tx, exists := t.hashToContent.LoadOrStore(hashStr, bxTransaction); exists {
		bxTransaction = tx
	} else {
		result.NewTx = true
	}

	// make sure we are the only process that makes changes to the transaction
	bxTransaction.Lock()

	if !bxTransaction.Flags().IsPaid() && flags.IsPaid() {
		result.Reprocess = true
		bxTransaction.AddFlags(types.TFPaidTx)
	}
	if !bxTransaction.Flags().ShouldDeliverToNode() && flags.ShouldDeliverToNode() {
		result.Reprocess = true
		bxTransaction.AddFlags(types.TFDeliverToNode)
	}

	// check hash in the bloom filter only after it is not seen in txStore
	if t.bloom != nil && shortID == types.ShortIDEmpty && !result.Reprocess && result.NewTx && t.bloom.Check(hash.Bytes()) {
		result.DebugData = "Transaction ignored due to already seen in bloom filter"
		result.AlreadySeen = true
	}

	// if shortID was not provided, assign shortID (if we are running as assigner)
	// note that assigner.Next() provides ShortIDEmpty if we are not assigning
	// also, shortID is not assigned if transaction is validators_only
	// if we assigned shortID, result.AssignedShortID hold non ShortIDEmpty value
	if result.NewTx && shortID == types.ShortIDEmpty && !bxTransaction.Flags().IsValidatorsOnly() && !bxTransaction.Flags().IsNextValidator() && !bxTransaction.Flags().IsValidatorsOnly() {
		shortID = t.assigner.Next()
		result.AssignedShortID = shortID
	}

	result.NewSID = bxTransaction.AddShortID(shortID)
	result.NewContent = bxTransaction.SetContent(content)
	// set sender only if it has new content in order to avoid false sender when the shortID is not new
	if result.NewContent {
		bxTransaction.SetSender(sender)
	}
	result.Transaction = bxTransaction
	bxTransaction.Unlock()

	if result.NewSID {
		t.shortIDToHash.Store(shortID, bxTransaction.Hash())
	}

	return result
}

type networkData struct {
	maxAge     time.Duration
	ages       []int
	cleanAge   int
	cleanNoSID int
}

func (t *BxTxStore) clean() (cleaned int, cleanedShortIDs types.ShortIDsByNetwork) {
	currTime := t.clock.Now()

	var networks = make(map[types.NetworkNum]*networkData)
	cleanedShortIDs = make(types.ShortIDsByNetwork)

	t.hashToContent.Range(func(key string, bxTransaction *types.BxTransaction) bool {
		netData, netDataExists := networks[bxTransaction.NetworkNum()]
		if !netDataExists {
			netData = &networkData{}
			networks[bxTransaction.NetworkNum()] = netData
		}
		txAge := int(currTime.Sub(bxTransaction.AddTime()) / time.Second)
		networks[bxTransaction.NetworkNum()].ages = append(networks[bxTransaction.NetworkNum()].ages, txAge)

		return true
	})

	for net, netData := range networks {
		// if we are below the number of allowed Txs, no need to do anything
		if len(netData.ages) <= bxgateway.TxStoreMaxSize {
			networks[net].maxAge = t.maxTxAge
			continue
		}
		// per network, sort ages in ascending order
		sort.Ints(netData.ages)
		// in order to avoid many cleanup msgs, cleanup only 90% of the TxStoreMaxSize
		networks[net].maxAge = time.Duration(netData.ages[int(bxgateway.TxStoreMaxSize*0.9)-1]) * time.Second
		if networks[net].maxAge > t.maxTxAge {
			networks[net].maxAge = t.maxTxAge
		}
		log.Debugf("TxStore size for network %v is %v. Cleaning %v transactions older than %v",
			net, len(netData.ages), len(netData.ages)-bxgateway.TxStoreMaxSize, networks[net].maxAge)
	}

	t.hashToContent.Range(func(key string, bxTransaction *types.BxTransaction) bool {

		networkNum := bxTransaction.NetworkNum()
		netData, netDataExists := networks[networkNum]
		removeReason := ""
		txAge := currTime.Sub(bxTransaction.AddTime())

		if netDataExists && txAge > netData.maxAge {
			removeReason = fmt.Sprintf("transation age %v is greater than  %v", txAge, netData.maxAge)
			netData.cleanAge++
		} else {
			if txAge > t.noSIDAge && len(bxTransaction.ShortIDs()) == 0 {
				removeReason = fmt.Sprintf("transation age %v but no short ID", txAge)
				netData.cleanNoSID++
			}
		}

		if removeReason != "" {
			// remove the transaction by hash from both maps
			// no need to add the hash to the history as it is deleted after long time
			// dec-5-2021: add to hash history to prevent a lot of reentry (BSC, Polygon)
			t.remove(key, FullReEntryProtection, removeReason)
			cleanedShortIDs[networkNum] = append(cleanedShortIDs[networkNum], bxTransaction.ShortIDs()...)
		}

		return true
	})

	for net, netData := range networks {
		log.Debugf("TxStore network %v #txs before cleanup %v cleaned %v missing SID entries and %v aged entries",
			net, len(netData.ages), netData.cleanNoSID, netData.cleanAge)
		cleaned += netData.cleanNoSID + netData.cleanAge
	}

	return cleaned, cleanedShortIDs
}

// CleanNow performs an immediate cleanup of the TxStore
func (t *BxTxStore) CleanNow() {
	mapSizeBeforeClean := t.Count()
	timeStart := t.clock.Now()
	cleaned, cleanedShortIDs := t.clean()
	log.Debugf("TxStore cleaned %v entries in %v. size before clean: %v size after clean: %v",
		cleaned, t.clock.Now().Sub(timeStart), mapSizeBeforeClean, t.Count())
	if t.cleanedShortIDsChannel != nil && len(cleanedShortIDs) > 0 {
		t.cleanedShortIDsChannel <- cleanedShortIDs
	}
}

func (t *BxTxStore) cleanup() {
	ticker := t.clock.Ticker(t.cleanupFreq)
	for {
		select {
		case <-ticker.Alert():
			t.CleanNow()
		case <-t.quit:
			t.quit <- true
			ticker.Stop()
			return
		}
	}
}

// Get returns a single transaction from the transaction service
func (t *BxTxStore) Get(hash types.SHA256Hash) (*types.BxTransaction, bool) {
	// reset the timestamp of this hash in the seenTx hashHistory, if it exists in the hashHistory
	if t.refreshSeenTx(hash) {
		return nil, false
	}
	tx, ok := t.hashToContent.Load(string(hash[:]))
	if !ok {
		return nil, ok
	}
	return tx, ok
}

// Known returns whether if a tx hash is in seenTx
func (t *BxTxStore) Known(hash types.SHA256Hash) bool {
	return t.refreshSeenTx(hash)
}

// HasContent returns if a given transaction is in the transaction service
func (t *BxTxStore) HasContent(hash types.SHA256Hash) bool {
	tx, ok := t.Get(hash)
	if !ok {
		return false
	}
	return tx.Content() != nil
}

// Summarize returns some info about the tx service
func (t *BxTxStore) Summarize() *pbbase.TxStoreReply {
	networks := make(map[types.NetworkNum]*pbbase.TxStoreNetworkData)
	res := pbbase.TxStoreReply{
		TxCount:      uint64(t.hashToContent.Size()),
		ShortIdCount: uint64(t.shortIDToHash.Size()),
	}

	t.hashToContent.Range(func(key string, bxTransaction *types.BxTransaction) bool {

		networkData, exists := networks[bxTransaction.NetworkNum()]
		if !exists {
			networkData = &pbbase.TxStoreNetworkData{}
			networkData.OldestTx = bxTransaction.Protobuf()
			networkData.TxCount++
			networkData.Network = uint64(bxTransaction.NetworkNum())
			networkData.ShortIdCount += uint64(len(bxTransaction.ShortIDs()))
			networks[bxTransaction.NetworkNum()] = networkData

			// continue iteration
			return true
		}
		oldestTx := networkData.OldestTx
		oldestTxTS := oldestTx.AddTime
		if bxTransaction.AddTime().Before(oldestTxTS.AsTime()) {
			networkData.OldestTx = bxTransaction.Protobuf()
		}
		networkData.TxCount++
		networkData.ShortIdCount += uint64(len(bxTransaction.ShortIDs()))

		return true
	})

	for _, netData := range networks {
		res.NetworkData = append(res.NetworkData, netData)
	}

	return &res
}

func (t *BxTxStore) refreshSeenTx(hash types.SHA256Hash) bool {
	if t.seenTxs.Exists(string(hash[:])) {
		t.seenTxs.Add(string(hash[:]), t.timeToAvoidReEntry)
		return true
	}
	return false
}
