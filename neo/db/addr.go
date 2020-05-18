package db

import (
	"database/sql"
	"fmt"
	"neo_explorer/core/cache"
	"neo_explorer/core/log"
	"neo_explorer/core/util"
	"neo_explorer/neo/addr"
	"neo_explorer/neo/asset"
)

// GetAddrAssetInfo returns all addresses with it's assets.
func GetAddrAssetInfo() []*addr.AssetInfo {
	const query = "SELECT `address`.`id`, `address`.`address`, `address`.`created_at`, `address`.`last_transaction_time`, `addr_asset`.`asset_id`, `addr_asset`.`balance` FROM `addr_asset` LEFT JOIN `address` ON `address`.`id`=`addr_asset`.`address_id`"

	result := []*addr.AssetInfo{}

	rows, err := wrappedQuery(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		m := &addr.AssetInfo{}
		var balanceStr string

		err := rows.Scan(
			&m.AddressId,
			&m.Address,
			&m.CreatedAt,
			&m.LastTransactionTime,
			&m.AssetId,
			&balanceStr,
		)

		if err != nil {
			panic(err)
		}

		m.Balance = util.StrToBigFloat(balanceStr)

		result = append(result, m)
	}

	return result
}

// returns true if new address created.
func updateAddrInfo(tx *sql.Tx, blockTime uint64, txID string, addr string, assetType string) (bool, error) {
	var incrAsset, incrNep5 = 0, 0
	switch assetType {
	case asset.ASSET:
		incrAsset = 1
	case asset.NEP5:
		incrNep5 = 1
	default:
		panic("Unsupported asset Type: " + assetType)
	}

	addressId, err := GetVoutAddrID(addr)
	if err != nil {
		panic(err)
	}
	addrCache, created := cache.GetAddrOrCreate(addr, blockTime, addressId)

	if created {
		const createAddrQuery = "INSERT INTO `address` (`id`, `address`, `created_at`, `last_transaction_time`, `trans_asset`, `trans_nep5`) VALUES (?, ?, ?, ?, ?, ?)"
		_, err = tx.Exec(createAddrQuery, addressId, addr, blockTime, blockTime, incrAsset, incrNep5)
		if err != nil {
			log.Error.Printf("TxMap: %s, addr=%s, assetType=%s\n", txID, addr, assetType)
			return true, err
		}

		return true, nil
	}

	query := fmt.Sprintf("UPDATE `address` SET `trans_asset` = `trans_asset` + %d, `trans_nep5` = `trans_nep5` + %d", incrAsset, incrNep5)
	// Because task tx and task nep5 runs in parallel,
	// maybe one task executes before the other one with a bigger blockTime.
	if addrCache.UpdateCreatedTime(blockTime) {
		query += fmt.Sprintf(", `created_at` = %d", blockTime)
	}
	if addrCache.UpdateLastTxTime(blockTime) {
		query += fmt.Sprintf(", `last_transaction_time` = %d", blockTime)
	}
	query += fmt.Sprintf(" WHERE `address` = '%s' LIMIT 1", addr)

	_, err = tx.Exec(query)
	return false, err
}

// returns true if new address created.
func createAddrInfoIfNotExist(tx *sql.Tx, blockTime uint64, addr string) (bool, error) {
	addressId, err := GetVoutAddrID(addr)
	if err != nil {
		panic(err)
	}
	_, created := cache.GetAddrOrCreate(addr, blockTime, addressId)
	if created {
		const createAddrQuery = "INSERT INTO `address` (`id`, `address`, `created_at`, `last_transaction_time`, `trans_asset`, `trans_nep5`) VALUES (?, ?, ?, ?, ?, ?)"
		_, err := tx.Exec(createAddrQuery, addressId, addr, blockTime, blockTime, 0, 0)
		return true, err
	}

	return false, nil
}

func GetVoutAddrID(addr string) (uint, error) {
	var id uint
	query := "SELECT `address_id` FROM `tx_vout` WHERE `address` = ? LIMIT 1"
	err := db.QueryRow(query, addr).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		if !connErr(err) {
			panic(err)
		}
		reconnect()
		return GetVoutAddrID(addr)
	}
	if id < 1 {
		return 0, fmt.Errorf("GetVoutAddrID Get Error : %s", addr)
	}

	return id, nil
}

func GetAddrID(addr string) (uint, error) {
	var id uint
	query := "SELECT `id` FROM `address` WHERE `address` = ? LIMIT 1"
	err := db.QueryRow(query, addr).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		if !connErr(err) {
			panic(err)
		}
		reconnect()
		return GetAddrID(addr)
	}
	if id < 1 {
		return 0, fmt.Errorf("GetVoutAddrID Get Error : %s", addr)
	}

	return id, nil
}

func GetVoutAddrCount() uint {
	var count uint
	query := "SELECT COUNT(DISTINCT `address`) FROM `tx_vout` ORDER BY `id` ASC"
	err := db.QueryRow(query).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		if !connErr(err) {
			panic(err)
		}
		reconnect()
		return GetVoutAddrCount()
	}

	return count
}
