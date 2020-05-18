package db

import (
	"database/sql"
	"fmt"
	"neo_explorer/neo/asset"
	"neo_explorer/neo/block"
	"neo_explorer/neo/tx"
	"strings"
)

// InsertBlock inserts raw block data into database.
func InsertBlock(maxIndex int, blocks []*block.Block, txBulk *tx.Bulk) error {
	insertBlocksCmd := generateInsertCmdForBlock(blocks)
	insertTxsCmd := generateInsertCmdForTxs(txBulk.TXs)
	insertTxAttrsCmd := generateInsertCmdForTxAttrs(txBulk.TXAttrs)
	insertTxVinsCmd := generateInsertCmdForTxVins(txBulk.TXVins)
	insertTxVoutsCmd := generateInsertCmdForTxVouts(txBulk.TXVouts)
	insertTxScriptsCmd := generateInsertCmdForTxScripts(txBulk.TXScripts)
	insertAssetsCmd := generateInsertCmdForAssets(txBulk.Assets)
	insertClaims := generateInsertCmdForClaims(txBulk.Claims)

	cmdList := []string{
		insertBlocksCmd,
		insertTxsCmd,
		insertTxAttrsCmd,
		insertTxVinsCmd,
		insertTxVoutsCmd,
		insertTxScriptsCmd,
		insertAssetsCmd,
		insertClaims,
	}

	return transact(func(tx *sql.Tx) error {
		for _, cmd := range cmdList {
			if cmd == "" {
				continue
			}
			if _, err := tx.Exec(cmd); err != nil {
				return err
			}
		}

		// Update tx type counter.
		txTypeCounter := countTxTypes(txBulk.TXs)
		for txType, cnt := range txTypeCounter {
			err := updateTxCounter(tx, txType, cnt)
			if err != nil {
				return err
			}
		}

		err := updateCounter(tx, "last_block_index", int64(maxIndex))
		return err

	})
}

func generateInsertCmdForBlock(blocks []*block.Block) string {
	if len(blocks) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `block` (`hash`, `size`, `version`, `previousblockhash`, `merkleroot`, `time`, `index`, `nonce`, `nextconsensus`, `script_invocation`, `script_verification`, `nextblockhash`) VALUES ")

	for _, b := range blocks {
		strBuilder.WriteString(fmt.Sprintf("('%s', %d, %d, '%s', '%s', %d, %d, '%s', '%s', '%s', '%s', '%s'),", b.Hash, b.Size, b.Version, b.PreviousBlockHash, b.MerkleRoot, b.Time, b.Index, b.Nonce, b.NextConsensus, b.ScriptInvocation, b.ScriptVerification, b.NextBlockhash))
	}

	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForTxs(txs []*tx.Transaction) string {
	if len(txs) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `tx` (`block_index`, `block_time`, `txid`, `size`, `type`, `version`, `sys_fee`, `net_fee`, `nonce`, `script`, `gas`) VALUES ")

	for _, tx := range txs {
		strBuilder.WriteString(fmt.Sprintf("(%d, %d, '%s', %d, '%s', %d, %.8f, %.8f, %d, '%s', %.8f),", tx.BlockIndex, tx.BlockTime, tx.TxID, tx.Size, tx.Type, tx.Version, tx.SysFee, tx.NetFee, tx.Nonce, tx.Script, tx.Gas))
	}
	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForTxAttrs(txAttrs []*tx.TransactionAttribute) string {
	if len(txAttrs) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `tx_attr` (`tx_id`, `usage`, `data`) VALUES ")

	for _, attr := range txAttrs {
		strBuilder.WriteString(fmt.Sprintf("('%d','%s', '%s'),", attr.TxId, attr.Usage, attr.Data))
	}

	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForTxVins(txVins []*tx.TransactionVin) string {
	if len(txVins) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `tx_vin` (`tx_id`,`txid`, `vout`) VALUES ")

	for _, vin := range txVins {
		strBuilder.WriteString(fmt.Sprintf("('%d','%d', %d),", vin.TxId, vin.TxID, vin.Vout))
	}

	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForTxVouts(txVouts []*tx.TransactionVout) string {
	if len(txVouts) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `tx_vout` (`tx_id`, `n`, `asset_id`, `value`, `address`, `address_id`) VALUES ")

	for _, vout := range txVouts {
		strBuilder.WriteString(fmt.Sprintf("('%d', %d, '%d', %.8f, '%s', '%d'),", vout.TxId, vout.N, vout.AssetID, vout.Value, vout.Address, vout.AddressId))
	}

	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForTxScripts(txScripts []*tx.TransactionScripts) string {
	if len(txScripts) == 0 {
		return ""
	}
	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `tx_scripts` (`tx_id`, `invocation`, `verification`) VALUES ")

	for _, script := range txScripts {
		strBuilder.WriteString(fmt.Sprintf("('%d', '%s', '%s'),", script.TxId, script.Invocation, script.Verification))
	}
	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForAssets(assets []*asset.Asset) string {
	if len(assets) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `asset` (`block_index`, `block_time`, `version`, `asset_id`, `type`, `name`, `amount`, `available`, `precision`, `owner`, `admin`, `issuer`, `expiration`, `frozen`, `addresses`, `transactions`) VALUES ")

	for _, asset := range assets {
		strBuilder.WriteString(fmt.Sprintf("(%d, %d, %d, '%s', '%s', '%s', %.8f, %.8f, %d, '%s', '%s', '%s', %d, %t, %d, %d),", asset.BlockIndex, asset.BlockTime, asset.Version, asset.AssetID, asset.Type, asset.Name, asset.Amount, asset.Available, asset.Precision, asset.Owner, asset.Admin, asset.Issuer, asset.Expiration, asset.Frozen, asset.Addresses, asset.Transactions))
	}
	return strings.TrimSuffix(strBuilder.String(), ",")
}

func generateInsertCmdForClaims(claims []*tx.TransactionClaims) string {
	if len(claims) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	strBuilder.WriteString("INSERT INTO `tx_claims` (`tx_id`, `vout`) VALUES ")

	for _, claim := range claims {
		strBuilder.WriteString(fmt.Sprintf("('%d', %d),", claim.TxId, claim.Vout))
	}
	return strings.TrimSuffix(strBuilder.String(), ",")
}

func countTxTypes(txs []*tx.Transaction) map[int]int {
	txTypeCounter := make(map[int]int)

	for _, t := range txs {
		switch t.Type {
		case "RegisterTransaction":
			txTypeCounter[tx.RegisterTransaction]++
		case "MinerTransaction":
			txTypeCounter[tx.MinerTransaction]++
		case "IssueTransaction":
			txTypeCounter[tx.IssueTransaction]++
		case "InvocationTransaction":
			txTypeCounter[tx.InvocationTransaction]++
		case "ContractTransaction":
			txTypeCounter[tx.ContractTransaction]++
		case "ClaimTransaction":
			txTypeCounter[tx.ClaimTransaction]++
		case "PublishTransaction":
			txTypeCounter[tx.PublishTransaction]++
		case "EnrollmentTransaction":
			txTypeCounter[tx.EnrollmentTransaction]++
		}
	}

	return txTypeCounter
}
