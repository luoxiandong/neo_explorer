package nep5

import (
	"encoding/hex"
	"math/big"
	"neo_explorer/core/util"
	"neo_explorer/neo/smartcontract"
)

// Nep5 db model.
type Nep5 struct {
	ID               uint
	AssetID          uint
	AdminAddress     string
	Name             string
	Symbol           string
	Decimals         uint8
	TotalSupply      *big.Float
	TxId             uint
	BlockIndex       uint
	BlockTime        uint64
	Addresses        uint64
	HoldingAddresses uint64
	Transfers        uint64
}

// RegInfo db model.
type RegInfo struct {
	TxId          uint
	ScriptHash    []byte
	Name          string
	Version       string
	Author        string
	Email         string
	Description   string
	NeedStorage   bool
	ParameterList string
	ReturnType    string
}

// Transaction db model.
type Transaction struct {
	ID         uint
	TxId       uint
	AssetID    uint
	From       string
	To         string
	Value      *big.Float
	BlockIndex uint
	BlockTime  uint64
}

// Tx represents nep5 transaction model.
type Tx struct {
	ID    uint
	TxID  string
	From  string
	To    string
	Value *big.Float
}

// GetNep5RegInfo extracts op codes from stack,
// and returns nep5 reg info if stack valid.
func GetNep5RegInfo(txId uint, opCodeDataStack *smartcontract.DataStack) (*RegInfo, bool) {
	if len(*opCodeDataStack) < 9 {
		return nil, false
	}

	for {
		if len(*opCodeDataStack) == 9 {
			break
		}

		opCodeDataStack.PopData()
	}

	scriptBytes := opCodeDataStack.PopData() // Contract Script.
	scriptHash := util.GetScriptHash(scriptBytes)
	// scriptHashHex := util.GetAssetIDFromScriptHash(scriptHash)

	regInfo := RegInfo{
		TxId:          txId,
		ScriptHash:    scriptHash,
		ParameterList: hex.EncodeToString(opCodeDataStack.PopData()),
		ReturnType:    hex.EncodeToString(opCodeDataStack.PopData()),
		NeedStorage:   opCodeDataStack.PopData()[0] == 0x01,
		Name:          string(opCodeDataStack.PopData()),
		Version:       string(opCodeDataStack.PopData()),
		Author:        string(opCodeDataStack.PopData()),
		Email:         string(opCodeDataStack.PopData()),
		Description:   string(opCodeDataStack.PopData()),
	}

	return &regInfo, true
}
