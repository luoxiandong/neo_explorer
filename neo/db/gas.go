package db

import (
	"database/sql"
	"fmt"
	"math/big"
	"neo_explorer/core/util"
	"neo_explorer/neo/tx"
)

// GasDateBalance is the struct to store GAS balance-date values.
type GasDateBalance struct {
	Date    string
	Balance *big.Float
}

var gasDateCache = make(map[uint]*GasDateBalance)

// ApplyGASAssetChange persists daily gas balance changes into DB.
func ApplyGASAssetChange(tx *tx.Transaction, date string, gasChangeMap map[uint]*big.Float) error {
	for addr, gasChange := range gasChangeMap {
		err := transact(func(trans *sql.Tx) error {
			gasDateBalanceCache, ok := gasDateCache[addr]
			if !ok {
				dataCache := GasDateBalance{
					Date:    date,
					Balance: gasChange,
				}

				gasDateCache[addr] = &dataCache

				lastDate, balance := queryAddrGasDateRecord(addr)
				if balance == nil || lastDate != date {
					if balance != nil {
						dataCache.Balance = new(big.Float).Add(balance, gasChange)
					}

					err := insertGasDateBalanceRecord(trans, addr, date, dataCache.Balance)
					if err != nil {
						return err
					}
				} else {
					dataCache.Balance = new(big.Float).Add(balance, gasChange)
					err := updateGasDateBalanceRecord(trans, addr, date, dataCache.Balance)
					if err != nil {
						return err
					}
				}

				err := updateCounter(trans, "last_tx_pk_gas_balance", int64(tx.ID))
				return err
			}

			newBalance := new(big.Float).Add(gasDateBalanceCache.Balance, gasChange)
			gasDateBalanceCache.Balance = newBalance

			if gasDateBalanceCache.Date == date {
				updateGasDateBalanceRecord(trans, addr, date, newBalance)
			} else {
				gasDateBalanceCache.Date = date
				insertGasDateBalanceRecord(trans, addr, date, newBalance)
			}

			err := updateCounter(trans, "last_tx_pk_gas_balance", int64(tx.ID))
			return err
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func queryAddrGasDateRecord(addressId uint) (string, *big.Float) {
	tableName := getAddrDateGasTableName(addressId)
	query := fmt.Sprintf("SELECT `date`, `balance` FROM `%s` ", tableName)
	query += fmt.Sprintf("WHERE `address_id` = '%d' ", addressId)
	query += fmt.Sprintf("ORDER BY `id` DESC LIMIT 1")

	var date string
	var balanceStr string
	err := db.QueryRow(query).Scan(&date, &balanceStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}

		if !connErr(err) {
			panic(err)
		}

		reconnect()
		return queryAddrGasDateRecord(addressId)
	}

	return date, util.StrToBigFloat(balanceStr)
}

func insertGasDateBalanceRecord(trans *sql.Tx, addressId uint, date string, balance *big.Float) error {
	tableName := getAddrDateGasTableName(addressId)
	query := fmt.Sprintf("INSERT INTO `%s`(`address_id`, `date`, `balance`) ", tableName)
	query += fmt.Sprintf("VALUES ('%d', '%s', %.8f)", addressId, date, balance)

	_, err := trans.Exec(query)
	return err
}

func updateGasDateBalanceRecord(trans *sql.Tx, addressId uint, date string, gasChange *big.Float) error {
	tableName := getAddrDateGasTableName(addressId)
	query := fmt.Sprintf("UPDATE `%s` ", tableName)
	query += fmt.Sprintf("SET `balance` = %.8f ", gasChange)
	query += fmt.Sprintf("WHERE `address_id` = '%d' and `date` = '%s' ", addressId, date)
	query += "LIMIT 1"

	_, err := trans.Exec(query)
	if err != nil {
		if !connErr(err) {
			panic(err)
		}

		reconnect()
		return updateGasDateBalanceRecord(trans, addressId, date, gasChange)
	}

	return nil
}

func getAddrDateGasTableName(addressId uint) string {
	//addr := GetAddrID(addressId)
	//suffix := strings.ToLower(string(addr[len(addr)-1]))
	//return "addr_gas_balance_" + suffix
	return "addr_gas_balance"
}
