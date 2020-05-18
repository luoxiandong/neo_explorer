package db

import (
	"database/sql"
	"fmt"
	"neo_explorer/core/util"
	"neo_explorer/neo/nep5"
)

// InsertSCInfos persists new smart contracts info into db.
func InsertSCInfos(scRegInfos []*nep5.RegInfo, txPK uint) error {
	if len(scRegInfos) == 0 {
		return UpdateLastTxPkForSC(txPK)
	}

	return transact(func(trans *sql.Tx) error {
		query := "INSERT INTO `smartcontract_info`(`tx_id`, `script_hash`, `name`, `version`, `author`, `email`, `description`, `need_storage`, `parameter_list`, `return_type`) VALUES "
		args := []interface{}{}

		for _, regInfo := range scRegInfos {
			scriptHashHex := util.GetAssetIDFromScriptHash(regInfo.ScriptHash)
			query += fmt.Sprintf("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?), ")
			args = append(args, regInfo.TxId, scriptHashHex, regInfo.Name, regInfo.Version, regInfo.Author, regInfo.Email, regInfo.Description, regInfo.NeedStorage, regInfo.ParameterList, regInfo.ReturnType)
		}

		_, err := trans.Exec(query[:len(query)-2], args...)
		if err != nil {
			panic(err)
		}

		return updateCounter(trans, "last_tx_pk_for_sc", int64(txPK))
	})
}
