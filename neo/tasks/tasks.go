package tasks

import (
	"fmt"
	"neo_explorer/core/buffer"
	"neo_explorer/core/cache"
	"neo_explorer/core/config"
	"neo_explorer/core/log"
	"neo_explorer/neo/db"
	"neo_explorer/neo/rpc"
	"time"
)

func Run() {
	log.Printf("Init cache.")
	// Init cache to speed up db queries
	assetInfo := db.GetAssetInfo()
	cache.LoadAssetsInfo(assetInfo)

	addrAssetInfo := db.GetAddrAssetInfo()
	cache.LoadAddrAssetInfo(addrAssetInfo)

	lastTxPkId = db.GetTxCount()
	LastAddrPkId.Set(int(db.GetVoutAddrCount()))

	dbHeight := db.GetLastHeight()
	initTask(dbHeight)

	// download blocks from network , put in blockBuffer
	for i := 0; i < config.GetGoroutines(); i++ {
		go fetchBlock()
	}

	blockChannel = make(chan *rpc.RawBlock, bufferSize)
	// get from blockBuffer and send blockChannel queue
	go arrangeBlock(dbHeight, blockChannel)
	// save to database from blockChannel queue
	go storeBlock(blockChannel)

	go startNep5Task()

	// get utxo/addr_asset/asset from tx
	go startTxTask()

	go startUpdateCounterTask()

	// get asset_tx from tx
	go startAssetTxTask()

	go startGasBalanceTask()

	go startSCTask()

	go tick()
}

func initTask(dbHeight int) {
	blockBuffer = buffer.NewBuffer(dbHeight)
	bestHeight := rpc.RefreshServers()

	log.Printf("Current params for block persistance:\n")
	log.Printf("\tdb block height = %d\n", dbHeight)
	log.Printf("\trpc best height = %d\n", bestHeight)
}

func tick() {
	t := time.NewTicker(time.Second * 5)
	for {
		<-t.C
		fmt.Printf(" ticker blockBuffer Size : %d \n", blockBuffer.Size())
		fmt.Printf(" ticker assetMap Size : %v,%v \n", cache.AssetMap, cache.AssetPairMap)
	}
}
