package cache_test

import (
	"neo_explorer/core/cache"

	"neo_explorer/core/config"
	"neo_explorer/neo/db"
	"testing"
)

func TestLoadAddrAssetInfo(t *testing.T) {
	config.Load()
	db.Init()
	addrAssetInfo := db.GetAddrAssetInfo()
	cache.LoadAddrAssetInfo(addrAssetInfo)
}
