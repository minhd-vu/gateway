package types

import (
	"encoding/hex"
	"errors"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// NewBeaconBlockNotification creates beacon block notification
func NewBeaconBlockNotification(block interfaces.ReadOnlySignedBeaconBlock) (BlockNotification, error) {
	switch block.Version() {
	case version.Bellatrix:
		blk, err := block.PbBellatrixBlock()
		if err != nil {
			return nil, err
		}

		hash, err := blk.GetBlock().HashTreeRoot()
		if err != nil {
			return nil, err
		}

		return &BellatrixBlockNotification{
			Hash:                       hex.EncodeToString(hash[:]),
			SignedBeaconBlockBellatrix: blk,
		}, nil
	case version.Capella:
		blk, err := block.PbCapellaBlock()
		if err != nil {
			return nil, err
		}

		hash, err := blk.GetBlock().HashTreeRoot()
		if err != nil {
			return nil, err
		}

		return &CapellaBlockNotification{
			Hash:                     hex.EncodeToString(hash[:]),
			SignedBeaconBlockCapella: blk,
		}, nil
	default:
		return nil, fmt.Errorf("not supported %s", version.String(block.Version()))
	}
}

// CapellaBlockNotification represents capella beacon block notification
type CapellaBlockNotification struct {
	*ethpb.SignedBeaconBlockCapella

	Hash string `json:"hash"`

	notificationType FeedType      `json:"-"`
	source           *NodeEndpoint `json:"-"`
}

// WithFields returns notification with specified fields
func (beaconBlockNotification *CapellaBlockNotification) WithFields(fields []string) Notification {
	block := CapellaBlockNotification{}
	for _, param := range fields {
		switch param {
		case "hash":
			block.Hash = beaconBlockNotification.Hash
		case "header":
			if block.SignedBeaconBlockCapella == nil {
				block.SignedBeaconBlockCapella = &ethpb.SignedBeaconBlockCapella{}
			}

			if block.Block == nil {
				block.Block = &ethpb.BeaconBlockCapella{}
			}

			block.Block.Slot = beaconBlockNotification.GetBlock().GetSlot()
			block.Block.ProposerIndex = beaconBlockNotification.GetBlock().GetProposerIndex()
			block.Block.ParentRoot = beaconBlockNotification.GetBlock().GetParentRoot()
			block.Block.StateRoot = beaconBlockNotification.GetBlock().GetStateRoot()
		case "slot":
			if block.SignedBeaconBlockCapella == nil {
				block.SignedBeaconBlockCapella = &ethpb.SignedBeaconBlockCapella{}
			}

			if block.SignedBeaconBlockCapella.Block == nil {
				block.Block = &ethpb.BeaconBlockCapella{}
			}

			block.Block.Slot = beaconBlockNotification.GetBlock().GetSlot()
		case "body":
			if block.SignedBeaconBlockCapella == nil {
				block.SignedBeaconBlockCapella = &ethpb.SignedBeaconBlockCapella{}
			}

			if block.SignedBeaconBlockCapella.Block == nil {
				block.Block = &ethpb.BeaconBlockCapella{}
			}

			block.Block.Body = beaconBlockNotification.GetBlock().GetBody()
		}
	}

	return &block
}

// Filters converts filters as field value map
func (beaconBlockNotification *CapellaBlockNotification) Filters(filters []string) map[string]interface{} {
	return nil
}

// LocalRegion -
func (beaconBlockNotification *CapellaBlockNotification) LocalRegion() bool {
	return false
}

// GetHash returns block hash
func (beaconBlockNotification *CapellaBlockNotification) GetHash() string {
	return beaconBlockNotification.Hash
}

// SetNotificationType - set feed name
func (beaconBlockNotification *CapellaBlockNotification) SetNotificationType(feedName FeedType) {
	beaconBlockNotification.notificationType = feedName
}

// NotificationType - feed name
func (beaconBlockNotification *CapellaBlockNotification) NotificationType() FeedType {
	return beaconBlockNotification.notificationType
}

// SetSource - source blockchain node endpoint
func (beaconBlockNotification *CapellaBlockNotification) SetSource(source *NodeEndpoint) {
	beaconBlockNotification.source = source
}

// Source - source blockchain node endpoint
func (beaconBlockNotification *CapellaBlockNotification) Source() *NodeEndpoint {
	return beaconBlockNotification.source
}

// IsNil returns true if nil
func (beaconBlockNotification *CapellaBlockNotification) IsNil() bool {
	return beaconBlockNotification == nil
}

// Clone clones notification
func (beaconBlockNotification *CapellaBlockNotification) Clone() BlockNotification {
	n := *beaconBlockNotification
	return &n
}

// BellatrixBlockNotification represents bellatrix beacon block notification
type BellatrixBlockNotification struct {
	*ethpb.SignedBeaconBlockBellatrix

	Hash string `json:"hash"`

	notificationType FeedType      `json:"-"`
	source           *NodeEndpoint `json:"-"`
}

// WithFields returns notification with specified fields
func (beaconBlockNotification *BellatrixBlockNotification) WithFields(fields []string) Notification {
	block := BellatrixBlockNotification{}
	for _, param := range fields {
		switch param {
		case "hash":
			block.Hash = beaconBlockNotification.Hash
		case "header":
			if block.SignedBeaconBlockBellatrix == nil {
				block.SignedBeaconBlockBellatrix = &ethpb.SignedBeaconBlockBellatrix{}
			}

			if block.Block == nil {
				block.Block = &ethpb.BeaconBlockBellatrix{}
			}

			block.Block.Slot = beaconBlockNotification.GetBlock().GetSlot()
			block.Block.ProposerIndex = beaconBlockNotification.GetBlock().GetProposerIndex()
			block.Block.ParentRoot = beaconBlockNotification.GetBlock().GetParentRoot()
			block.Block.StateRoot = beaconBlockNotification.GetBlock().GetStateRoot()
		case "slot":
			if block.SignedBeaconBlockBellatrix == nil {
				block.SignedBeaconBlockBellatrix = &ethpb.SignedBeaconBlockBellatrix{}
			}

			if block.SignedBeaconBlockBellatrix.Block == nil {
				block.Block = &ethpb.BeaconBlockBellatrix{}
			}

			block.Block.Slot = beaconBlockNotification.GetBlock().GetSlot()
		case "body":
			if block.SignedBeaconBlockBellatrix == nil {
				block.SignedBeaconBlockBellatrix = &ethpb.SignedBeaconBlockBellatrix{}
			}

			if block.SignedBeaconBlockBellatrix.Block == nil {
				block.Block = &ethpb.BeaconBlockBellatrix{}
			}

			block.Block.Body = beaconBlockNotification.GetBlock().GetBody()
		}
	}

	return &block
}

// Filters converts filters as field value map
func (beaconBlockNotification *BellatrixBlockNotification) Filters(filters []string) map[string]interface{} {
	return nil
}

// LocalRegion -
func (beaconBlockNotification *BellatrixBlockNotification) LocalRegion() bool {
	return false
}

// GetHash returns block hash
func (beaconBlockNotification *BellatrixBlockNotification) GetHash() string {
	return beaconBlockNotification.Hash
}

// SetNotificationType - set feed name
func (beaconBlockNotification *BellatrixBlockNotification) SetNotificationType(feedName FeedType) {
	beaconBlockNotification.notificationType = feedName
}

// NotificationType - feed name
func (beaconBlockNotification *BellatrixBlockNotification) NotificationType() FeedType {
	return beaconBlockNotification.notificationType
}

// SetSource - source blockchain node endpoint
func (beaconBlockNotification *BellatrixBlockNotification) SetSource(source *NodeEndpoint) {
	beaconBlockNotification.source = source
}

// Source - source blockchain node endpoint
func (beaconBlockNotification *BellatrixBlockNotification) Source() *NodeEndpoint {
	return beaconBlockNotification.source
}

// IsNil returns true if nil
func (beaconBlockNotification *BellatrixBlockNotification) IsNil() bool {
	return beaconBlockNotification == nil
}

// Clone clones notification
func (beaconBlockNotification *BellatrixBlockNotification) Clone() BlockNotification {
	n := *beaconBlockNotification
	return &n
}

// EthBlockNotification - represents a single block
type EthBlockNotification struct {
	BlockHash        *ethcommon.Hash          `json:"hash,omitempty"`
	Header           *Header                  `json:"header,omitempty"`
	Transactions     []map[string]interface{} `json:"transactions,omitempty"`
	Uncles           []Header                 `json:"uncles,omitempty"`
	ValidatorInfo    []*FutureValidatorInfo   `json:"future_validator_info,omitempty"`
	notificationType FeedType
	source           *NodeEndpoint
}

// NewEthBlockNotification creates ETH block notification
func NewEthBlockNotification(hash ethcommon.Hash, block *ethtypes.Block, info []*FutureValidatorInfo) (*EthBlockNotification, error) {
	if hash == (ethcommon.Hash{}) {
		return nil, errors.New("empty block hash")
	}

	ethTxs := make([]map[string]interface{}, 0)
	for _, tx := range block.Transactions() {
		var ethTx *EthTransaction
		txHash, err := NewSHA256Hash(tx.Hash().Bytes())
		if err != nil {
			return nil, err
		}

		// send EmptySender to cause extraction of real sender
		ethTx, err = NewEthTransaction(txHash, tx, EmptySender)
		if err != nil {
			return nil, err
		}
		fields := ethTx.Fields(AllFields)
		// todo: calculate gasPrice for DynamicFeeTxType properly
		if ethTx.Type() == ethtypes.DynamicFeeTxType {
			fields["gasPrice"] = fields["maxFeePerGas"]
		}
		ethTxs = append(ethTxs, ethTx.Fields(AllFields))
	}
	ethUncles := make([]Header, 0, len(block.Uncles()))
	for _, uncle := range block.Uncles() {
		ethUncle := ConvertEthHeaderToBlockNotificationHeader(uncle)
		ethUncles = append(ethUncles, *ethUncle)
	}
	return &EthBlockNotification{
		BlockHash:     &hash,
		Header:        ConvertEthHeaderToBlockNotificationHeader(block.Header()),
		Transactions:  ethTxs,
		Uncles:        ethUncles,
		ValidatorInfo: info,
	}, nil
}

// FutureValidatorInfo - represents information about the validator information of the second block after the current block
type FutureValidatorInfo struct {
	BlockHeight uint64 `json:"block_height"`
	WalletID    string `json:"wallet_id"`
	Accessible  bool   `json:"accessible"`
}

// Header - represents Ethereum block header
type Header struct {
	ParentHash       ethcommon.Hash     `json:"parentHash"`
	Sha3Uncles       ethcommon.Hash     `json:"sha3Uncles"`
	Miner            *ethcommon.Address `json:"miner"`
	StateRoot        ethcommon.Hash     `json:"stateRoot"`
	TransactionsRoot ethcommon.Hash     `json:"transactionsRoot"`
	ReceiptsRoot     ethcommon.Hash     `json:"receiptsRoot"`
	LogsBloom        string             `json:"logsBloom"`
	Difficulty       string             `json:"difficulty"`
	Number           string             `json:"number"`
	GasLimit         string             `json:"gasLimit"`
	GasUsed          string             `json:"gasUsed"`
	Timestamp        string             `json:"timestamp"`
	ExtraData        string             `json:"extraData"`
	MixHash          ethcommon.Hash     `json:"mixHash"`
	Nonce            string             `json:"nonce"`
	BaseFee          *int               `json:"baseFeePerGas,omitempty"`
	WithdrawalsHash  *ethcommon.Hash    `json:"withdrawalsRoot,omitempty"`
	hexNumber        uint64
}

// GetNumber returns the block number from the header in uint64
func (h Header) GetNumber() uint64 {
	return h.hexNumber
}

// ConvertEthHeaderToBlockNotificationHeader converts Ethereum header to bloxroute Ethereum Header
func ConvertEthHeaderToBlockNotificationHeader(ethHeader *ethtypes.Header) *Header {
	newHeader := Header{
		ParentHash:       ethHeader.ParentHash,
		Sha3Uncles:       ethHeader.UncleHash,
		Miner:            &ethHeader.Coinbase,
		StateRoot:        ethHeader.Root,
		TransactionsRoot: ethHeader.TxHash,
		ReceiptsRoot:     ethHeader.ReceiptHash,
		LogsBloom:        fmt.Sprintf("0x%x", hex.EncodeToString(ethHeader.Bloom.Bytes())),
		Difficulty:       hexutil.EncodeBig(ethHeader.Difficulty),
		hexNumber:        ethHeader.Number.Uint64(),
		Number:           hexutil.EncodeBig(ethHeader.Number),
		GasLimit:         hexutil.EncodeUint64(ethHeader.GasLimit),
		GasUsed:          hexutil.EncodeUint64(ethHeader.GasUsed),
		Timestamp:        hexutil.EncodeUint64(ethHeader.Time),
		ExtraData:        hexutil.Encode(ethHeader.Extra),
		MixHash:          ethHeader.MixDigest,
		Nonce:            hexutil.EncodeUint64(ethHeader.Nonce.Uint64()),
		WithdrawalsHash:  ethHeader.WithdrawalsHash,
	}
	if ethHeader.BaseFee != nil {
		baseFee := int(ethHeader.BaseFee.Int64())
		newHeader.BaseFee = &baseFee
	}
	return &newHeader
}

// WithFields returns notification with specified fields
func (ethBlockNotification *EthBlockNotification) WithFields(fields []string) Notification {
	block := EthBlockNotification{}
	for _, param := range fields {
		switch param {
		case "hash":
			block.BlockHash = ethBlockNotification.BlockHash
		case "header":
			block.Header = ethBlockNotification.Header
		case "transactions":
			block.Transactions = ethBlockNotification.Transactions
		case "uncles":
			block.Uncles = ethBlockNotification.Uncles
		case "future_validator_info":
			block.ValidatorInfo = ethBlockNotification.ValidatorInfo
		}
	}
	return &block
}

// Filters converts filters as field value map
func (ethBlockNotification *EthBlockNotification) Filters(filters []string) map[string]interface{} {
	return nil
}

// LocalRegion -
func (ethBlockNotification *EthBlockNotification) LocalRegion() bool {
	return false
}

// GetHash returns block hash
func (ethBlockNotification *EthBlockNotification) GetHash() string {
	return ethBlockNotification.BlockHash.Hex()
}

// SetNotificationType - set feed name
func (ethBlockNotification *EthBlockNotification) SetNotificationType(feedName FeedType) {
	ethBlockNotification.notificationType = feedName
}

// NotificationType - feed name
func (ethBlockNotification *EthBlockNotification) NotificationType() FeedType {
	return ethBlockNotification.notificationType
}

// SetSource - source blockchain node endpoint
func (ethBlockNotification *EthBlockNotification) SetSource(source *NodeEndpoint) {
	ethBlockNotification.source = source
}

// Source - source blockchain node endpoint
func (ethBlockNotification *EthBlockNotification) Source() *NodeEndpoint {
	return ethBlockNotification.source
}

// IsNil return true if nil
func (ethBlockNotification *EthBlockNotification) IsNil() bool {
	return ethBlockNotification == nil
}

// Clone clones notification
func (ethBlockNotification *EthBlockNotification) Clone() BlockNotification {
	n := *ethBlockNotification
	return &n
}
