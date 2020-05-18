package tx

import (
	"math/big"
	"neo_explorer/neo/asset"
)

const (
	RegisterTransaction   = iota //用于资产登记的特殊交易
	MinerTransaction             //共识交易，用于分配字节费的特殊交易
	IssueTransaction             //用于分发资产的特殊交易
	InvocationTransaction        //调用智能合约的交易
	ContractTransaction          //合约交易，这是最常用的一种交易
	ClaimTransaction             //用于分配NeoGas的特殊交易
	PublishTransaction           //发布智能合约的交易
	EnrollmentTransaction        //用于报名成为记账候选人的特殊交易
)

// Bulk stores innner content of parsed raw block data.
type Bulk struct {
	TXs       []*Transaction
	TXAttrs   []*TransactionAttribute
	TXVins    []*TransactionVin
	TXVouts   []*TransactionVout
	TXScripts []*TransactionScripts
	Assets    []*asset.Asset
	Claims    []*TransactionClaims
}

// Transaction db model.
type Transaction struct {
	ID         uint
	BlockIndex uint
	BlockTime  uint64
	TxID       string
	Size       uint
	Type       string
	Version    uint
	// Attribute List
	// Vin List
	// Vout List
	SysFee *big.Float
	NetFee *big.Float
	// Scripts
	Nonce  int64
	Script string
	Gas    *big.Float
}

// TransactionAttribute of transactions.
type TransactionAttribute struct {
	ID   uint
	TxId uint
	//TxMap  string
	Usage string
	Data  string
}

// TransactionVin of transacitons.
type TransactionVin struct {
	// ID   uint
	TxId uint
	//From string
	TxID uint
	Vout uint16
}

// TransactionVout of transaction.
type TransactionVout struct {
	// ID      uint
	TxId uint
	//TxMap    string
	N         uint16
	AssetID   uint
	Value     *big.Float
	Address   string
	AddressId uint
}

// TransactionScripts of transaction.
type TransactionScripts struct {
	ID   uint
	TxId uint
	//TxMap         string
	Invocation   string
	Verification string
}

// TransactionClaims of transaction.
type TransactionClaims struct {
	ID   uint
	TxId uint
	//TxMap string
	Vout uint16
}

// AddrAssetIDTx is the bundle of address, asset_id and txid.
type AddrAssetIDTx struct {
	//Address string
	AddressId uint
	AssetID uint
	TxId    uint
	//TxMap    string
}
