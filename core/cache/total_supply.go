package cache

import (
	"math/big"
	"sync"
)

// AssetTotalSupplyCacheItem caches AssetMap with its total supply.
type AssetTotalSupplyCacheItem struct {
	TotalSupply *big.Float
	BlockIndex  uint
}

var (
	totalSupplyCache = make(map[uint]*AssetTotalSupplyCacheItem)
	assetCacheLock   sync.Mutex
)

// GetAssetTotalSupply returns AssetMap total supply cache record.
func GetAssetTotalSupply(assetId uint) (*big.Float, uint, bool) {
	assetCacheLock.Lock()
	defer assetCacheLock.Unlock()

	rec, ok := totalSupplyCache[assetId]
	if !ok {
		return nil, 0, false
	}

	return rec.TotalSupply, rec.BlockIndex, true
}

// UpdateAssetTotalSupply updates or sets total supply for AssetMap.
func UpdateAssetTotalSupply(assetId uint, totalSupply *big.Float, blockIndex uint) bool {
	assetCacheLock.Lock()
	defer assetCacheLock.Unlock()

	if rec, ok := totalSupplyCache[assetId]; ok {
		if rec.BlockIndex > blockIndex {
			return false
		}
	}

	totalSupplyCache[assetId] = &AssetTotalSupplyCacheItem{
		TotalSupply: totalSupply,
		BlockIndex:  blockIndex,
	}

	return true
}
