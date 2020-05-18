package parse

import (
	"fmt"
	"math/big"
	"neo_explorer/core/cache"
	"neo_explorer/core/util"
	"neo_explorer/neo/asset"
	"neo_explorer/neo/db"
	"neo_explorer/neo/rpc"
	"neo_explorer/neo/smartcontract"
	"neo_explorer/neo/tx"
	"strings"
)

var txMap map[string]uint
var addrMap map[string]uint

// ParseTxs parses all raw transactions in raw blocks to Bulk.
func Txs(rawBlocks []*rpc.RawBlock, lastTxPkId *uint, LastAddrPkId *util.SafeCounter) *tx.Bulk {
	txs := tx.Bulk{}
	txMap = make(map[string]uint, 100)
	addrMap = make(map[string]uint, 100)

	for _, rawBlock := range rawBlocks {
		for _, rawTx := range rawBlock.Tx {
			*lastTxPkId++
			txMap[rawTx.TxID] = *lastTxPkId
			txs.TXs = appendTx(txs.TXs, rawBlock.Index, rawBlock.Time, &rawTx, *lastTxPkId)
			txs.TXAttrs = appendTxAttrs(txs.TXAttrs, &rawTx, *lastTxPkId)
			txs.TXVins = appendTxVin(txs.TXVins, &rawTx, *lastTxPkId)
			txs.TXVouts = appendTxVout(txs.TXVouts, &rawTx, *lastTxPkId, LastAddrPkId)
			txs.TXScripts = appendTxScripts(txs.TXScripts, &rawTx, *lastTxPkId)
			txs.Assets = appendAsset(rawBlock, txs.Assets, &rawTx)
			txs.Claims = appendClaims(txs.Claims, &rawTx, *lastTxPkId)
		}
	}

	return &txs
}

func appendTx(txs []*tx.Transaction, blockIndex uint, blockTime uint64, rawTx *rpc.RawTx, ID uint) []*tx.Transaction {
	trans := tx.Transaction{
		ID:         ID,
		BlockIndex: blockIndex,
		BlockTime:  blockTime,
		TxID:       rawTx.TxID,
		Size:       rawTx.Size,
		Type:       rawTx.Type,
		Version:    rawTx.Version,
		SysFee:     rawTx.SysFee,
		NetFee:     rawTx.NetFee,
		Nonce:      rawTx.Nonce,
		Script:     rawTx.Script,
		Gas:        rawTx.Gas,
	}
	if rawTx.Gas == nil {
		trans.Gas = big.NewFloat(0)
	}
	txs = append(txs, &trans)

	return txs
}

func appendTxAttrs(txAttrs []*tx.TransactionAttribute, rawTx *rpc.RawTx, txId uint) []*tx.TransactionAttribute {
	for _, rawAttr := range rawTx.Attributes {
		attr := tx.TransactionAttribute{
			TxId: txId,
			//TxMap:  rawTx.TxMap,
			Usage: rawAttr.Usage,
			Data:  rawAttr.Data,
		}
		txAttrs = append(txAttrs, &attr)
	}
	return txAttrs
}

func appendTxVin(txVin []*tx.TransactionVin, rawTx *rpc.RawTx, txId uint) []*tx.TransactionVin {
	for _, rawVin := range rawTx.Vin {
		txID, ok := txMap[rawVin.TxID]
		if !ok {
			txID = db.GetTx(rawVin.TxID)
			if txID < 1 {
				err, _ := fmt.Printf("appendTxVin get TxID error: %+v", rawTx)
				panic(err)
			}
		}

		vin := tx.TransactionVin{
			TxId: txId,
			//From: rawTx.TxMap,
			TxID: txID,
			Vout: rawVin.Vout,
		}
		txVin = append(txVin, &vin)
	}
	return txVin
}

func appendTxVout(txVout []*tx.TransactionVout, rawTx *rpc.RawTx, txId uint, LastAddrPkId *util.SafeCounter) []*tx.TransactionVout {
	for _, rawVout := range rawTx.Vout {
		assetId, err := cache.GetAssetId(rawVout.Asset)
		if err != nil {
			panic(err)
		}
		addrID, ok := addrMap[rawVout.Address]
		if !ok {
			addrID, err = db.GetVoutAddrID(rawVout.Address)
			if err != nil {
				LastAddrPkId.Add(1)
				addrID = uint(LastAddrPkId.Get())
			}
		}

		vout := tx.TransactionVout{
			TxId: txId,
			//TxMap:    rawTx.TxMap,
			N:         rawVout.N,
			AssetID:   assetId,
			Value:     rawVout.Value,
			Address:   rawVout.Address,
			AddressId: addrID,
		}
		addrMap[rawVout.Address] = addrID
		txVout = append(txVout, &vout)
	}
	return txVout
}

func appendTxScripts(txScripts []*tx.TransactionScripts, rawTx *rpc.RawTx, txId uint) []*tx.TransactionScripts {
	for _, rawScript := range rawTx.Scripts {
		script := tx.TransactionScripts{
			TxId: txId,
			//TxMap:         rawTx.TxMap,
			Invocation:   rawScript.Invocation,
			Verification: rawScript.Verification,
		}
		txScripts = append(txScripts, &script)
	}
	return txScripts
}

func appendAsset(rawBlock *rpc.RawBlock, assets []*asset.Asset, rawTx *rpc.RawTx) []*asset.Asset {
	var asset *asset.Asset
	if rawTx.Type == typeName(tx.RegisterTransaction) {
		asset = parseAssetFromRegisterTransaction(rawBlock.Index, rawTx)
	} else if rawTx.Type == typeName(tx.InvocationTransaction) {
		// Example: 0x4a629db0af0d9c7ee0e11f4f4894765f5ab2579bcc8b4a203e4c6814a9784f00(testnet).
		if strings.HasSuffix(rawTx.Script, smartcontract.AssetFingerPrint) {
			asset = parseAssetFromInvocationTransaction(rawTx.Script)
			if asset == nil {
				return assets
			}

			// Supplement the rest fields.
			asset.Version = 0
			asset.AssetID = rawTx.TxID
			asset.Expiration = uint64(rawBlock.Index + 2000000)
		}
	}

	if asset == nil {
		return assets
	}

	id, err := cache.GetAssetId(rawTx.TxID)
	if err != nil {
		panic(err)
	}
	asset.ID = id

	asset.BlockIndex = rawBlock.Index
	asset.BlockTime = rawBlock.Time
	asset.Addresses = 0
	asset.Transactions = 0

	assets = append(assets, asset)
	return assets
}

func appendClaims(claims []*tx.TransactionClaims, rawTx *rpc.RawTx, txId uint) []*tx.TransactionClaims {
	for _, rawClaim := range rawTx.Claims {
		claim := tx.TransactionClaims{
			TxId: txId,
			//TxMap: rawClaim.TxMap,
			Vout: rawClaim.Vout,
		}
		claims = append(claims, &claim)
	}
	return claims
}

func parseAssetFromRegisterTransaction(blockIndex uint, rawTx *rpc.RawTx) *asset.Asset {
	newAsset := asset.Asset{
		Version:    rawTx.Version,
		AssetID:    rawTx.TxID,
		Type:       rawTx.Asset.Type,
		Name:       rawTx.Asset.Name[0].Name,
		Amount:     rawTx.Asset.Amount,
		Available:  big.NewFloat(0),
		Precision:  rawTx.Asset.Precision,
		Owner:      rawTx.Asset.Owner,
		Admin:      rawTx.Asset.Admin,
		Issuer:     rawTx.Asset.Owner,
		Expiration: uint64(blockIndex + 2*2000000),
		Frozen:     false,
	}

	if newAsset.AssetID == asset.NEOAssetID {
		newAsset.Name = asset.NEO
	} else if newAsset.AssetID == asset.GASAssetID {
		newAsset.Name = asset.GAS
	}

	return &newAsset
}

func parseAssetFromInvocationTransaction(sc string) *asset.Asset {
	asset := smartcontract.GetAssetInfo(sc)
	return asset
}

func typeName(txType int) string {
	switch txType {
	case tx.RegisterTransaction:
		return "RegisterTransaction"
	case tx.MinerTransaction:
		return "MinerTransaction"
	case tx.IssueTransaction:
		return "IssueTransaction"
	case tx.InvocationTransaction:
		return "InvocationTransaction"
	case tx.ContractTransaction:
		return "ContractTransaction"
	case tx.ClaimTransaction:
		return "ClaimTransaction"
	case tx.PublishTransaction:
		return "PublishTransaction"
	case tx.EnrollmentTransaction:
		return "EnrollmentTransaction"
	default:
		err := fmt.Errorf("unknown transaction type: %d", txType)
		panic(err)
	}
}
