package cache

import (
	"math/big"
	"neo_explorer/neo/addr"
	"sync"
)

// AddrCacheItem caches address related data.
type AddrCacheItem struct {
	CreatedAt           uint64
	LastTransactionTime uint64
	AddrAssetCache      map[uint]*AddrAssetCacheItem
}

// AddrAssetCacheItem records balance of address assets.
type AddrAssetCacheItem struct {
	Balance *big.Float
	// This balance is 'up to date' till 'BlockIndex'.
	BlockIndex uint
}

var (
	addrCache     map[uint]*AddrCacheItem
	addrCacheLock sync.RWMutex

	// assetAlias maps all AssetMap with an integer number,
	// so we can reduce memory usage of cache.
	assetLock sync.RWMutex
)

// LoadAddrAssetInfo caches all addr asset info.
func LoadAddrAssetInfo(addrAssetInfo []*addr.AssetInfo) {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	addrCache = make(map[uint]*AddrCacheItem)

	for _, info := range addrAssetInfo {
		_, ok := addrCache[info.AddressId]
		if !ok {
			addrCache[info.AddressId] = &AddrCacheItem{
				CreatedAt:           info.CreatedAt,
				LastTransactionTime: info.LastTransactionTime,
				AddrAssetCache:      make(map[uint]*AddrAssetCacheItem),
			}
		}

		addrCache[info.AddressId].AddrAssetCache[info.AssetId] = &AddrAssetCacheItem{
			Balance:    info.Balance,
			BlockIndex: 0,
		}
	}
}

// GetAddrAsset returns AddrAssetCacheItem by address and AssetMap.
func GetAddrAsset(addressId uint, assetId uint) (*AddrAssetCacheItem, bool) {
	addrCacheLock.RLock()
	defer addrCacheLock.RUnlock()

	cache, ok := addrCache[addressId]
	if !ok {
		return nil, false
	}

	addrAssetCache, ok := cache.AddrAssetCache[assetId]
	return addrAssetCache, ok
}

// GetAddrOrCreate gets or creates address cache.
func GetAddrOrCreate(address string, txTime uint64, addressId uint) (*AddrCacheItem, bool) {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if cache, ok := addrCache[addressId]; ok {
		return cache, false
	}

	cache := &AddrCacheItem{
		CreatedAt:           txTime,
		LastTransactionTime: txTime,
		AddrAssetCache:      make(map[uint]*AddrAssetCacheItem),
	}
	addrCache[addressId] = cache

	return cache, true
}

// UpdateCreatedTime updates address created time.
func (cache *AddrCacheItem) UpdateCreatedTime(blockTime uint64) bool {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if cache.CreatedAt > blockTime {
		cache.CreatedAt = blockTime
		return true
	}

	return false
}

// UpdateLastTxTime updates address last transaction.
func (cache *AddrCacheItem) UpdateLastTxTime(lastTxTime uint64) bool {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if cache.LastTransactionTime < lastTxTime {
		cache.LastTransactionTime = lastTxTime
		return true
	}

	return false
}

// GetAddrAsset returns AddrAssetCacheItem by AssetMap.
func (cache *AddrCacheItem) GetAddrAsset(assetID string) (*AddrAssetCacheItem, bool) {
	addrCacheLock.RLock()
	defer addrCacheLock.RUnlock()

	assetId, err := GetAssetId(assetID)
	if err != nil {
		panic(err)
	}

	addrAssetCache, ok := cache.AddrAssetCache[assetId]
	return addrAssetCache, ok
}

// GetAddrAssetOrCreate gets or creates address asset cache.
func (cache *AddrCacheItem) GetAddrAssetOrCreate(assetId uint, balance *big.Float) (*AddrAssetCacheItem, bool) {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if addrAssetCache, ok := cache.AddrAssetCache[assetId]; ok {
		return addrAssetCache, false
	}

	cache.AddrAssetCache[assetId] = &AddrAssetCacheItem{
		Balance:    balance,
		BlockIndex: 0,
	}

	return cache.AddrAssetCache[assetId], true
}

// UpdateBalance updates balance of address asset.
func (addrAssetCache *AddrAssetCacheItem) UpdateBalance(balance *big.Float, blockIndex uint) bool {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if blockIndex < addrAssetCache.BlockIndex {
		return false
	}

	addrAssetCache.BlockIndex = blockIndex

	if addrAssetCache.Balance.Cmp(balance) != 0 {
		addrAssetCache.Balance = balance
		return true
	}

	return false
}

// AddBalance increases balance at the given blockIndex.
func (addrAssetCache *AddrAssetCacheItem) AddBalance(delta *big.Float, blockIndex uint) bool {
	if delta.Cmp(big.NewFloat(0)) == 0 {
		return false
	}

	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if blockIndex < addrAssetCache.BlockIndex {
		return false
	}

	addrAssetCache.BlockIndex = blockIndex

	addrAssetCache.Balance = new(big.Float).Add(addrAssetCache.Balance, delta)
	return true
}

// SubtractBalance decreases balance at the given blockIndex.
func (addrAssetCache *AddrAssetCacheItem) SubtractBalance(delta *big.Float, blockIndex uint) bool {
	if delta.Cmp(big.NewFloat(0)) == 0 {
		return false
	}

	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	if blockIndex < addrAssetCache.BlockIndex {
		return false
	}

	addrAssetCache.BlockIndex = blockIndex

	addrAssetCache.Balance = new(big.Float).Sub(addrAssetCache.Balance, delta)
	return true
}

// CreateAddrAsset creates address asset cache.
func CreateAddrAsset(addressId uint, assetId uint, balance *big.Float, blockIndex uint) {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	cache := getAddrCache(addressId)
	cache.AddrAssetCache[assetId] = &AddrAssetCacheItem{
		Balance:    balance,
		BlockIndex: blockIndex,
	}
}

func getAddrCache(addressId uint) *AddrCacheItem {
	cache, ok := addrCache[addressId]
	if !ok {
		panic("Falied to find target addrCache. Make sure address data is cached first")
	}

	return cache
}

// MigrateNEP5 handles nep5 contract migration.
func MigrateNEP5(newAssetAdminId uint, oldAssetID, newAssetID string) (uint, uint) {
	addrCacheLock.Lock()
	defer addrCacheLock.Unlock()

	addrs := uint(0)
	holdingAddrs := uint(0)

	for addr, item := range addrCache {
		newAssetId, err := GetAssetId(newAssetID)
		if err != nil {
			panic(err)
		}
		if addr == newAssetAdminId {
			if _, ok := addrCache[newAssetAdminId].AddrAssetCache[newAssetId]; ok {
				continue
			}
		}
		oldAssetId, err := GetAssetId(oldAssetID)
		if err != nil {
			panic(err)
		}
		if old, ok := item.AddrAssetCache[oldAssetId]; ok {
			item.AddrAssetCache[newAssetId] = &AddrAssetCacheItem{
				Balance:    new(big.Float).Copy(old.Balance),
				BlockIndex: old.BlockIndex,
			}

			addrs++
			if old.Balance.Sign() > 0 {
				holdingAddrs++
			}

			delete(item.AddrAssetCache, oldAssetId)
		}
	}

	return addrs, holdingAddrs
}
