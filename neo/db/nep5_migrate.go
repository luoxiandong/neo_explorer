package db

import (
	"database/sql"
	"neo_explorer/core/cache"
)

// HandleNEP5Migrate handles nep5 contract migration.
func HandleNEP5Migrate(newAssetAdmin, oldAssetID, newAssetID string, txPK uint) error {
	return transact(func(tx *sql.Tx) error {
		query := "UPDATE `nep5` SET `visible` = FALSE WHERE `asset_id` = ? LIMIT 1"
		if _, err := tx.Exec(query, oldAssetID); err != nil {
			return err
		}

		query = "DELETE FROM `addr_asset` WHERE `asset_id` = ? AND `address_id` IN ("
		query += "SELECT `address_id` FROM (SELECT `address_id` FROM `addr_asset` WHERE asset_id=? AND `address_id` IN ("
		query += "SELECT `address_id` FROM `addr_asset` WHERE `asset_id` IN (?, ?) GROUP BY `address_id` HAVING COUNT(`asset_id`) = 2))a)"
		if _, err := tx.Exec(query, newAssetID, newAssetID, oldAssetID, newAssetID); err != nil {
			return err
		}

		query = "UPDATE `addr_asset` SET `asset_id` = ? WHERE `asset_id` = ?"
		if _, err := tx.Exec(query, newAssetID, oldAssetID); err != nil {
			return err
		}

		newAssetAdminId, err2 := GetVoutAddrID(newAssetAdmin)
		if err2 != nil {
			panic(err2)
		}
		addrs, holdingAddrs := cache.MigrateNEP5(newAssetAdminId, oldAssetID, newAssetID)
		query = "UPDATE `nep5` SET `addresses` = ?, `holding_addresses` = ? WHERE `asset_id` = ? LIMIT 1"
		if _, err := tx.Exec(query, addrs, holdingAddrs, newAssetID); err != nil {
			return err
		}

		query = "INSERT INTO `nep5_migrate`(`old_asset_id`, `new_asset_id`, `migrate_tx_id`) VALUES (?, ?, ?)"
		if _, err := tx.Exec(query, oldAssetID, newAssetID, txPK); err != nil {
			return err
		}

		err := updateNep5Counter(tx, txPK, -1)
		return err
	})
}
