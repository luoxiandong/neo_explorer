package db

import (
	"database/sql"
	"fmt"
	"math/big"
	"neo_explorer/core/cache"
	"neo_explorer/core/log"
	"neo_explorer/core/util"
	"neo_explorer/neo/addr"
	"neo_explorer/neo/asset"
	"neo_explorer/neo/nep5"
	"neo_explorer/neo/tx"
	"sort"
	"strings"
)

type addrInfo struct {
	addr    string
	balance *big.Float
}

// GetInvocationTxs returns invocation transactions.
func GetInvocationTxs(startPk uint, limit uint) []*tx.Transaction {
	const query = "SELECT `id`, `block_index`, `block_time`, `txid`, `size`, `type`, `version`, `sys_fee`, `net_fee`, `nonce`, `script`, `gas` FROM `tx` WHERE `id` >= ? AND `type` = ? ORDER BY ID ASC LIMIT ?"
	rows, err := wrappedQuery(query, startPk, "InvocationTransaction", limit)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	result := []*tx.Transaction{}

	for rows.Next() {
		var t tx.Transaction
		sysFeeStr := ""
		netFeeStr := ""
		gasStr := ""

		err := rows.Scan(
			&t.ID,
			&t.BlockIndex,
			&t.BlockTime,
			&t.TxID,
			&t.Size,
			&t.Type,
			&t.Version,
			&sysFeeStr,
			&netFeeStr,
			&t.Nonce,
			&t.Script,
			&gasStr,
		)

		if err != nil {
			panic(err)
		}

		t.SysFee = util.StrToBigFloat(sysFeeStr)
		t.NetFee = util.StrToBigFloat(netFeeStr)
		t.Gas = util.StrToBigFloat(gasStr)

		result = append(result, &t)
	}

	return result
}

// GetNep5AssetDecimals returns all nep5 asset_id with decimal.
func GetNep5AssetDecimals() map[uint]uint8 {
	nep5Decimals := make(map[uint]uint8)
	const query = "SELECT `asset_id`, `decimals` FROM `nep5`"
	rows, err := wrappedQuery(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var assetId uint
		var decimal uint8
		if err := rows.Scan(&assetId, &decimal); err != nil {
			panic(err)
		}
		nep5Decimals[assetId] = decimal
	}

	return nep5Decimals
}

// GetTxScripts returns script string of transaction.
func GetTxScripts(txId uint) ([]*tx.TransactionScripts, error) {
	var txScripts []*tx.TransactionScripts
	const query = "SELECT `id`, `tx_id`, `invocation`, `verification` FROM `tx_scripts` WHERE `tx_id` = ?"
	rows, err := wrappedQuery(query, txId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		txScript := tx.TransactionScripts{}
		rows.Scan(
			&txScript.ID,
			&txScript.TxId,
			&txScript.Invocation,
			&txScript.Verification,
		)
		txScripts = append(txScripts, &txScript)
	}

	return txScripts, nil
}

// InsertNep5Asset inserts new nep5 asset into db.
func InsertNep5Asset(trans *tx.Transaction, nep5 *nep5.Nep5, regInfo *nep5.RegInfo, addrAsset *addr.Asset, atHeight uint) error {
	return transact(func(tx *sql.Tx) error {
		insertNep5Sql := fmt.Sprintf("INSERT INTO `nep5` (`asset_id`, `admin_address`, `name`, `symbol`, `decimals`, `total_supply`, `tx_id`, `block_index`, `block_time`, `addresses`, `holding_addresses`, `transfers`) VALUES('%d', '%s', '%s', '%s', %d, %.8f, '%d', %d, %d, %d, %d, %d)", nep5.AssetID, nep5.AdminAddress, nep5.Name, nep5.Symbol, nep5.Decimals, nep5.TotalSupply, nep5.TxId, nep5.BlockIndex, nep5.BlockTime, nep5.Addresses, nep5.HoldingAddresses, nep5.Transfers)
		res, err := tx.Exec(insertNep5Sql)
		if err != nil {
			return err
		}

		newPK, err := res.LastInsertId()
		if err != nil {
			return err
		}
		const insertNep5RegInfo = "INSERT INTO `nep5_reg_info` (`nep5_id`, `name`, `version`, `author`, `email`, `description`, `need_storage`, `parameter_list`, `return_type`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
		if _, err := tx.Exec(insertNep5RegInfo, newPK, regInfo.Name, regInfo.Version, regInfo.Author, regInfo.Email, regInfo.Description, regInfo.NeedStorage, regInfo.ParameterList, regInfo.ReturnType); err != nil {
			return err
		}

		addrCreated := false
		if addrAsset != nil {
			addrCreated, err = createAddrInfoIfNotExist(tx, trans.BlockTime, addrAsset.Address)
			if err != nil {
				log.Error.Printf("TxMap: %s, nep5Info: %+v, regInfo=%+v, addrAsset=%+v, atHeight=%d\n", trans.TxID, nep5, regInfo, addrAsset, atHeight)
				return err
			}

			if _, ok := cache.GetAddrAsset(addrAsset.AddressId, addrAsset.AssetID); !ok {
				cache.CreateAddrAsset(addrAsset.AddressId, addrAsset.AssetID, addrAsset.Balance, atHeight)
				insertAddrAssetQuery := fmt.Sprintf("INSERT INTO `addr_asset` (`address_id`, `asset_id`, `balance`, `transactions`, `last_transaction_time`) VALUES ('%d', '%d', %.8f, %d, %d)", addrAsset.AddressId, addrAsset.AssetID, addrAsset.Balance, addrAsset.Transactions, addrAsset.LastTransactionTime)
				if _, err := tx.Exec(insertAddrAssetQuery); err != nil {
					return err
				}
			}
		}

		if addrCreated {
			if err := incrAddrCounter(tx, 1); err != nil {
				return err
			}
		}
		return updateNep5Counter(tx, trans.ID, -1)
	})
}

// UpdateNep5TotalSupplyAndAddrAsset updates nep5 total supply and admin balance.
func UpdateNep5TotalSupplyAndAddrAsset(blockTime uint64, blockIndex uint, addr string, balance *big.Float, assetId uint, totalSupply *big.Float) error {
	return transact(func(tx *sql.Tx) error {
		addrCreated := false
		var err error

		addressId, err := GetVoutAddrID(addr)
		if err != nil {
			panic(err)
		}
		if balance.Cmp(big.NewFloat(0)) == 1 {
			if addrCreated, err = createAddrInfoIfNotExist(tx, blockTime, addr); err != nil {
				log.Error.Printf("blockTime=%d, blockIndex=%d, addr=%s, balance=%v, assetId=%d, totalSupply=%v\n",
					blockTime, blockIndex, addr, balance, assetId, totalSupply)
				return err
			}
			cachedAddr, _ := cache.GetAddrOrCreate(addr, blockTime, addressId)
			addrAssetCache, created := cachedAddr.GetAddrAssetOrCreate(assetId, balance)

			if created {
				insertAddrAssetQuery := fmt.Sprintf("INSERT INTO `addr_asset` (`address_id`, `asset_id`, `balance`, `transactions`, `last_transaction_time`) VALUES ('%d', '%d', %.8f, %d, %d)", addressId, assetId, balance, 0, blockTime)
				if _, err := tx.Exec(insertAddrAssetQuery); err != nil {
					return err
				}
				const incrNep5AddrQuery = "UPDATE `nep5` SET `addresses` = `addresses` + 1, `holding_addresses` = `holding_addresses` + 1 WHERE `asset_id` = ? LIMIT 1"
				if _, err := tx.Exec(incrNep5AddrQuery, assetId); err != nil {
					return err
				}
			} else {
				if addrAssetCache.UpdateBalance(balance, blockIndex) {
					query := fmt.Sprintf("UPDATE `addr_asset` SET `balance` = %.8f WHERE `address` = '%s' AND `asset_id` = '%d' LIMIT 1", balance, addr, assetId)
					if _, err := tx.Exec(query); err != nil {
						return err
					}
				}
			}
		} else {
			// balance is zero.
			if addrAssetCache, ok := cache.GetAddrAsset(addressId, assetId); ok {
				if addrAssetCache.UpdateBalance(balance, blockIndex) {
					const updateBalanceQuery = "UPDATE `nep5` SET `holding_addresses` = `holding_addresses` - 1 WHERE `asset_id` = ? LIMIT 1"
					if _, err := tx.Exec(updateBalanceQuery, assetId); err != nil {
						return err
					}
				}
			}
		}

		// Update nep5 total supply.
		if err := UpdateNep5TotalSupply(tx, assetId, totalSupply); err != nil {
			return err
		}

		if addrCreated {
			return incrAddrCounter(tx, 1)
		}

		return nil
	})
}

// UpdateNep5TotalSupply updates total supply of nep5 asset.
func UpdateNep5TotalSupply(tx *sql.Tx, assetId uint, totalSupply *big.Float) error {
	query := fmt.Sprintf("UPDATE `nep5` SET `total_supply` = %.8f WHERE `asset_id` = '%d' LIMIT 1", totalSupply, assetId)

	_, err := tx.Exec(query)

	return err
}

// InsertNep5transaction inserts new nep5 transaction into db.
func InsertNep5transaction(trans *tx.Transaction, appLogIdx int, assetId uint, fromAddr string, fromBalance *big.Float, toAddr string, toBalance *big.Float, transferValue *big.Float, totalSupply *big.Float) error {
	return transact(func(tx *sql.Tx) error {
		addrsOffset := 0
		holdingAddrsOffset := 0

		addrInfoPair := []addrInfo{
			{addr: fromAddr, balance: fromBalance},
			{addr: toAddr, balance: toBalance},
		}

		// Handle special case.
		if fromAddr == toAddr {
			addrInfoPair = addrInfoPair[:1]
		} else {
			// Sort address to avoid potential deadlock.
			sort.SliceStable(addrInfoPair, func(i, j int) bool {
				return addrInfoPair[i].addr < addrInfoPair[j].addr
			})
		}

		addrCreatedCnt := 0

		for _, info := range addrInfoPair {
			addr := info.addr
			balance := info.balance

			if len(addr) == 0 {
				continue
			}

			addrCreated, err := updateAddrInfo(tx, trans.BlockTime, trans.TxID, addr, asset.NEP5)
			if err != nil {
				return err
			}
			if addrCreated {
				addrCreatedCnt++
			}

			addressId, err := GetVoutAddrID(addr)
			if err != nil {
				panic(err)
			}
			cachedAddr, _ := cache.GetAddrOrCreate(addr, trans.BlockTime, addressId)
			addrAssetCache, created := cachedAddr.GetAddrAssetOrCreate(assetId, balance)

			if balance.Cmp(big.NewFloat(0)) == 1 {
				if created || addrAssetCache.Balance.Cmp(big.NewFloat(0)) == 0 {
					holdingAddrsOffset++
				}
			} else { // have no balance currently.
				if !created && addrAssetCache.Balance.Cmp(big.NewFloat(0)) == 1 {
					holdingAddrsOffset--
				}
			}

			if created {
				addrsOffset++
			}

			// Insert addr_asset record if not exist or update record.
			if created {
				insertAddrAssetQuery := fmt.Sprintf("INSERT INTO `addr_asset` (`address_id`, `asset_id`, `balance`, `transactions`, `last_transaction_time`) VALUES ('%d', '%d', %.8f, %d, %d)", addressId, assetId, balance, 1, trans.BlockTime)
				if _, err := tx.Exec(insertAddrAssetQuery); err != nil {
					return err
				}
			} else {
				addrAssetCache.UpdateBalance(balance, trans.BlockIndex)
				updateAddrAssetQuery := fmt.Sprintf("UPDATE `addr_asset` SET `balance` = %.8f, `transactions` = `transactions` + 1, `last_transaction_time` = %d WHERE `address_id` = '%d' AND `asset_id` = '%d' LIMIT 1", balance, trans.BlockTime, addressId, assetId)
				if _, err := tx.Exec(updateAddrAssetQuery); err != nil {
					return err
				}
			}
		}

		// Update nep5 transactions and addresses counter.
		txSQL := fmt.Sprintf("UPDATE `nep5` SET `addresses` = `addresses` + %d, `holding_addresses` = `holding_addresses` + %d, `transfers` = `transfers` + 1 WHERE `asset_id` = '%d' LIMIT 1;", addrsOffset, holdingAddrsOffset, assetId)

		// Insert nep5 transaction record.
		txSQL += fmt.Sprintf("INSERT INTO `nep5_tx` (`tx_id`, `asset_id`, `from`, `to`, `value`, `block_index`, `block_time`) VALUES ('%d', '%d', '%s', '%s', %.8f, %d, %d);", trans.ID, assetId, fromAddr, toAddr, transferValue, trans.BlockIndex, trans.BlockTime)

		// Handle resultant of storage injection attach.
		if totalSupply != nil {
			txSQL += fmt.Sprintf("UPDATE `nep5` SET `total_supply` = %.8f WHERE `asset_id` = '%d' LIMIT 1;", totalSupply, assetId)
		}

		if _, err := tx.Exec(txSQL); err != nil {
			return err
		}

		if addrCreatedCnt > 0 {
			if err := incrAddrCounter(tx, addrCreatedCnt); err != nil {
				return err
			}
		}
		err := updateNep5Counter(tx, trans.ID, appLogIdx)
		return err
	})
}

// GetMaxNonEmptyScriptTxPk returns largest pk of invocation transaction.
func GetMaxNonEmptyScriptTxPk() uint {
	const query = "SELECT `id` from `tx` WHERE `type` = ? ORDER BY `id` DESC LIMIT 1"

	var pk uint
	err := db.QueryRow(query, "InvocationTransaction").Scan(&pk)
	if err != nil && err != sql.ErrNoRows {
		if !connErr(err) {
			panic(err)
		}
		reconnect()
		return GetMaxNonEmptyScriptTxPk()
	}

	return pk
}

// GetNep5TxRecords returns paged nep5 transactions from db.
func GetNep5TxRecords(pk uint, limit int) ([]*nep5.Transaction, error) {
	const query = "SELECT `id`, `tx_id`, `asset_id`, `from`, `to`, `value`, `block_index`, `block_time` FROM `nep5_tx` WHERE `id` > ? ORDER BY `id` ASC LIMIT ?"
	rows, err := wrappedQuery(query, pk, limit)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	records := []*nep5.Transaction{}

	for rows.Next() {
		var id uint
		var txId uint
		var assetID uint
		var from string
		var to string
		var valueStr string
		var blockIndex uint
		var blockTime uint64

		err := rows.Scan(&id, &txId, &assetID, &from, &to, &valueStr, &blockIndex, &blockTime)
		if err != nil {
			return nil, err
		}

		record := &nep5.Transaction{
			ID:         id,
			TxId:       txId,
			AssetID:    assetID,
			From:       from,
			To:         to,
			Value:      util.StrToBigFloat(valueStr),
			BlockIndex: blockIndex,
			BlockTime:  blockTime,
		}
		records = append(records, record)
	}

	return records, nil
}

// InsertNep5AddrTxRec inserts addr_tx record of nep5 transactions.
func InsertNep5AddrTxRec(nep5TxRecs []*nep5.Transaction, lastPk uint) error {
	if len(nep5TxRecs) == 0 {
		return nil
	}

	return transact(func(tx *sql.Tx) error {
		var strBuilder strings.Builder

		strBuilder.WriteString("INSERT INTO `addr_tx` (`tx_id`, `address_id`, `block_time`, `asset_type`) VALUES ")

		for _, rec := range nep5TxRecs {
			if len(rec.From) > 0 {
				fromId, err := GetVoutAddrID(rec.From)
				if err != nil {
					panic(err)
				}
				strBuilder.WriteString(fmt.Sprintf("('%d', '%d', %d, '%s'),", rec.TxId, fromId, rec.BlockTime, asset.NEP5))
			}
			if len(rec.To) > 0 {
				toId, err := GetVoutAddrID(rec.To)
				if err != nil {
					panic(err)
				}
				strBuilder.WriteString(fmt.Sprintf("('%d', '%d', %d, '%s'),", rec.TxId, toId, rec.BlockTime, asset.NEP5))
			}
		}
		var query = strBuilder.String()
		if query[len(query)-1] != ',' {
			return nil
		}

		query = strings.TrimSuffix(query, ",")
		query += "ON DUPLICATE KEY UPDATE `address_id`=`address_id`"

		if _, err := tx.Exec(query); err != nil {
			return err
		}

		return UpdateNep5TxPkForAddrTx(tx, lastPk)
	})
}
