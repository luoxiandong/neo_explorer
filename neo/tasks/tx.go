/*
To restart this task from beginning, execute the following sqls:

TRUNCATE TABLE `utxo`;
DELETE FROM `addr_asset` WHERE LENGTH(`asset_id`) = 66;
DELETE FROM `addr_tx` WHERE `asset_type` = 'asset';
UPDATE `counter` SET
    `last_tx_pk` = 0,
    `cnt_tx_reg` = 0,
    `cnt_tx_miner` = 0,
    `cnt_tx_issue` = 0,
    `cnt_tx_invocation` = 0,
    `cnt_tx_contract` = 0,
    `cnt_tx_claim` = 0,
    `cnt_tx_publish` = 0,
	`cnt_tx_enrollment` = 0
WHERE `id` = 1;
UPDATE `asset` SET `addresses` = 0, `available` = 0, `transactions` = 0;
UPDATE `address` SET `trans_asset` = 0;
TRUNCATE TABLE `asset_tx`;
UPDATE `counter` SET `last_asset_tx_pk` = 0 WHERE `id` = 1;

*/

package tasks

import (
	"fmt"
	"math/big"
	"neo_explorer/core/log"
	"neo_explorer/neo/db"
	"neo_explorer/neo/tx"
	"strconv"
	"time"
)

const txChanSize = 5000

var (
	// TxMaxPkShouldRefresh indicates if highest tx pk should be refreshed.
	TxMaxPkShouldRefresh bool
	tProgress            = Progress{}
	maxTxPK              uint
	addressMap           map[string]uint
)

type txInfo struct {
	tx    *tx.Transaction
	vins  []*tx.TransactionVin
	vouts []*tx.TransactionVout
}

func startTxTask() {
	txChan := make(chan txInfo, txChanSize)
	nextPK := db.GetLastTxPkCounter() + 1

	go fetchTx(txChan, nextPK)
	go handleTx(txChan)
}

func fetchTx(txChan chan<- txInfo, nextPK uint) {

	for {
		txs := db.GetTxs(nextPK, 1000, "")
		if len(txs) == 0 {
			//log.Printf("Waiting for new transactions...[fetchTx]\n")
			time.Sleep(2 * time.Second)
			continue
		}

		nextPK = txs[len(txs)-1].ID + 1
		txIDs := []string{}

		for _, tx := range txs {
			txIDs = append(txIDs, strconv.Itoa(int(tx.ID)))
		}

		vinMap, voutMap, err := db.GetVinVout(txIDs)
		if err != nil {
			panic(err)
		}

		for _, tx := range txs {
			txChan <- txInfo{
				tx:    tx,
				vins:  vinMap[tx.ID],
				vouts: voutMap[tx.ID],
			}
		}
	}
}

func handleTx(txChan <-chan txInfo) {
	for txInfo := range txChan {
		tx := txInfo.tx
		vins := txInfo.vins
		vouts := txInfo.vouts

		err := db.ApplyVinsVouts(tx, vins, vouts)
		if err != nil {
			panic(err)
		}

		showTxProgress(tx.ID)
	}
}

func showTxProgress(currentTxPk uint) {
	if maxTxPK == 0 || TxMaxPkShouldRefresh {
		TxMaxPkShouldRefresh = false
		maxTxPK = db.GetHighestTxPk()
	}

	now := time.Now()
	if tProgress.LastOutputTime == (time.Time{}) {
		tProgress.LastOutputTime = now
	}
	if currentTxPk < maxTxPK && now.Sub(tProgress.LastOutputTime) < time.Second {
		return
	}

	GetEstimatedRemainingTime(int64(currentTxPk), int64(maxTxPK), &tProgress)
	if tProgress.Percentage.Cmp(big.NewFloat(100)) == 0 &&
		bProgress.Finished {
		tProgress.Finished = true
	}

	log.Printf("%s Progress of transactions: %d/%d, %.4f%%\n",
		tProgress.RemainingTimeStr,
		currentTxPk,
		maxTxPK,
		tProgress.Percentage)

	tProgress.LastOutputTime = now

	// Send mail if fully synced.
	if tProgress.Finished {
		// If sync lasts shortly, do not send mail.
		if time.Since(tProgress.InitTime) < time.Minute*5 {
			return
		}

		fmt.Sprintf("Init time: %v\nEnd Time: %v\n", tProgress.InitTime, time.Now())
	}
}
