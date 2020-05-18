package tasks

import (
	"fmt"
	"math/big"
	"neo_explorer/core/cache"
	"neo_explorer/core/log"
	"neo_explorer/neo/asset"
	"neo_explorer/neo/db"
	"time"
)

const gasBalanceChainSize = 5000

var (
	gasMaxPkShouldRefresh bool
	gasProgress           = Progress{}
	maxTxPkForGas         uint
)

func startGasBalanceTask() {
	gasBalanceChan := make(chan txInfo, gasBalanceChainSize)
	nextPK := db.GetLastTxPkForGasBalance() + 1

	go fetchTx(gasBalanceChan, nextPK)
	go handleTxGASBalance(gasBalanceChan)
}

func handleTxGASBalance(gasBalanceChan <-chan txInfo) {
	for info := range gasBalanceChan {
		gasChangeMap := getGASChange(info)

		if len(gasChangeMap) == 0 {
			continue
		}

		date := time.Unix(int64(info.tx.BlockTime), 0)
		err := db.ApplyGASAssetChange(info.tx, date.Format("2006-01-02"), gasChangeMap)
		if err != nil {
			panic(err)
		}

		showGasDateBalanceProgress(info.tx.ID)
	}
}

func getGASChange(info txInfo) map[uint]*big.Float {
	vins := info.vins
	vouts := info.vouts

	if len(vins) == 0 && len(vouts) == 0 {
		return nil
	}

	gasMap := make(map[uint]*big.Float)

	for _, vin := range vins {
		vinVout, err := db.GetVout(vin.TxID, vin.Vout)
		if err != nil {
			panic(err)
		}

		assetID, err := cache.GetAssetID(vinVout.AssetID)
		if err != nil {
			panic(err)
		}
		if assetID == asset.GASAssetID {
			negAmount := new(big.Float).Neg(vinVout.Value)
			updateMapValue(gasMap, vinVout.AddressId, negAmount)
		}
	}

	for _, vout := range vouts {
		assetID, err := cache.GetAssetID(vout.AssetID)
		if err != nil {
			panic(err)
		}
		if assetID == asset.GASAssetID {
			updateMapValue(gasMap, vout.AddressId, vout.Value)
		}
	}

	return gasMap
}

func updateMapValue(mp map[uint]*big.Float, key uint, offset *big.Float) {
	if mp == nil {
		mp = make(map[uint]*big.Float)
	}

	value, ok := mp[key]
	if ok {
		mp[key] = new(big.Float).Add(value, offset)
		return
	}

	mp[key] = new(big.Float).Set(offset)
}

func showGasDateBalanceProgress(currentTxPK uint) {
	if maxTxPkForGas == 0 || gasMaxPkShouldRefresh {
		gasMaxPkShouldRefresh = false
		maxTxPkForGas = db.GetHighestTxPk()
	}

	now := time.Now()
	if gasProgress.LastOutputTime == (time.Time{}) {
		gasProgress.LastOutputTime = now
	}
	if currentTxPK < maxTxPkForGas && now.Sub(gasProgress.LastOutputTime) < time.Second {
		return
	}

	GetEstimatedRemainingTime(int64(currentTxPK), int64(maxTxPkForGas), &gasProgress)
	if gasProgress.Percentage.Cmp(big.NewFloat(100)) == 0 &&
		bProgress.Finished {
		gasProgress.Finished = true
	}

	log.Printf("%sProgress of Addr-Date-Gas: %d/%d, %.4f%%\n",
		gasProgress.RemainingTimeStr,
		currentTxPK,
		maxTxPkForGas,
		gasProgress.Percentage)

	gasProgress.LastOutputTime = now

	// Send mail if fully synced.
	if gasProgress.Finished {
		// If sync lasts shortly, do not send mail.
		if time.Since(gasProgress.InitTime) < time.Minute*5 {
			return
		}

		fmt.Sprintf("Init time: %v\nEnd Time: %v\n", gasProgress.InitTime, time.Now())
	}
}
