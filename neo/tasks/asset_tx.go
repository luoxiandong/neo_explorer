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

const assetTxChanSize = 5000

var (
	// AssetTxMaxPkShouldRefresh indicates if highest tx pk should be refreshed.
	AssetTxMaxPkShouldRefresh bool
	assetProgress             = Progress{}
	maxTxPKforAssetTx         uint
)

func startAssetTxTask() {
	assetTxChan := make(chan *txInfo, assetTxChanSize)

	go fetchAssetTx(assetTxChan)
	go handleAssetTx(assetTxChan)
}

func fetchAssetTx(assetTxChan chan<- *txInfo) {
	nextPK := db.GetLastAssetTxPkCounter() + 1

	for {
		txs := db.GetTxs(nextPK, 50, "")
		if len(txs) == 0 {
			//log.Printf("Waiting for new transactions...[fetchAssetTx]\n")
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
			assetTxChan <- &txInfo{
				tx:    tx,
				vins:  vinMap[tx.ID],
				vouts: voutMap[tx.ID],
			}
		}
	}
}

func handleAssetTx(assetTxChan <-chan *txInfo) {
	records := []tx.AddrAssetIDTx{}
	maxPK := uint64(0)

	for {
		select {
		case t := <-assetTxChan:
			maxPK = uint64(t.tx.ID)
			records = processAssetTx(records, t)
		case <-time.After(2 * time.Second):
			recordAddrAssetIDTx(records, int64(maxPK))
			records = records[:0]
		}
	}
}

func processAssetTx(records []tx.AddrAssetIDTx, t *txInfo) []tx.AddrAssetIDTx {
	if t != nil {
		vins := t.vins
		vouts := t.vouts
		uniqueKey := make(map[string]bool)

		for _, vin := range vins {
			vinVout, err := db.GetVout(vin.TxID, vin.Vout)
			if err != nil {
				panic(err)
			}

			key := fmt.Sprintf("%s%d%d", vinVout.Address, vinVout.AssetID, t.tx.ID)
			if _, ok := uniqueKey[key]; ok {
				continue
			}

			records = append(records, tx.AddrAssetIDTx{
				AddressId: vinVout.AddressId,
				AssetID: vinVout.AssetID,
				//TxID:    t.tx.TxID,
				TxId:    t.tx.ID,
			})

			uniqueKey[key] = true
		}

		for _, vout := range vouts {
			key := fmt.Sprintf("%s%d%d", vout.Address, vout.AssetID, t.tx.ID)
			if _, ok := uniqueKey[key]; ok {
				continue
			}

			records = append(records, tx.AddrAssetIDTx{
				AddressId: vout.AddressId,
				AssetID: vout.AssetID,
				//TxID:    t.tx.TxID,
				TxId:    t.tx.ID,
			})

			uniqueKey[key] = true
		}
	}

	if len(records) == 0 {
		return nil
	}

	if len(records) >= 100 {
		recordAddrAssetIDTx(records, int64(t.tx.ID))
		return nil
	}

	return records
}

func recordAddrAssetIDTx(records []tx.AddrAssetIDTx, maxPK int64) {
	if len(records) == 0 {
		return
	}

	err := db.RecordAddrAssetIDTx(records, maxPK)
	if err != nil {
		panic(err)
	}

	showAssetTxProgress(uint(maxPK))
}

func showAssetTxProgress(currentTxPk uint) {
	if maxTxPKforAssetTx == 0 || AssetTxMaxPkShouldRefresh {
		AssetTxMaxPkShouldRefresh = false
		maxTxPKforAssetTx = db.GetHighestTxPk()
	}

	now := time.Now()
	if assetProgress.LastOutputTime == (time.Time{}) {
		assetProgress.LastOutputTime = now
	}
	if currentTxPk < maxTxPKforAssetTx && now.Sub(assetProgress.LastOutputTime) < time.Second {
		return
	}

	GetEstimatedRemainingTime(int64(currentTxPk), int64(maxTxPKforAssetTx), &assetProgress)
	if assetProgress.Percentage.Cmp(big.NewFloat(100)) == 0 &&
		bProgress.Finished {
		assetProgress.Finished = true
	}

	log.Printf("%sProgress of asset tx: %d/%d, %.4f%%\n",
		assetProgress.RemainingTimeStr,
		currentTxPk,
		maxTxPKforAssetTx,
		assetProgress.Percentage)
	assetProgress.LastOutputTime = now

	// Send mail if fully synced.
	if assetProgress.Finished {
		// If sync lasts shortly, do not send mail.
		if time.Since(assetProgress.InitTime) < time.Minute*5 {
			return
		}
		fmt.Sprintf("Init time: %v\nEnd Time: %v\n", assetProgress.InitTime, time.Now())
	}
}
