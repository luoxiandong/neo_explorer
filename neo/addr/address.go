package addr

import (
	"math/big"
)

// Address db model.
type Address struct {
	ID                  uint
	Address             string
	CreatedAt           uint64
	LastTransactionTime uint64
	TransAsset          uint64
	TransNep5 			uint64
}

// Asset db model.
type Asset struct {
	ID                  uint
	Address             string
	AddressId           uint
	AssetID             uint
	Balance             *big.Float
	Transactions        uint64
	LastTransactionTime uint64
}

// AssetInfo model.
type AssetInfo struct {
	AddressId  			uint
	Address             string
	CreatedAt           uint64
	LastTransactionTime uint64
	AssetId             uint
	Balance             *big.Float
}

// Tx model.
type Tx struct {
	ID        uint
	TxID      string
	Address   string
	BlockTime uint64
	AssetType string
}
