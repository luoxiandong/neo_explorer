package cache

import (
	"fmt"
	"neo_explorer/neo/asset"
)

var AssetMap map[string]uint
var AssetPairMap map[uint]string

var assetMaxID = uint(0)

func LoadAssetsInfo(assets []asset.Asset) {
	AssetMap = make(map[string]uint)
	AssetPairMap = make(map[uint]string)
	for _, asset := range assets {
		AssetMap[asset.AssetID] = asset.ID
		AssetPairMap[asset.ID] = asset.AssetID
	}

	assetMaxID += uint(len(assets))
}

func GetAssetId(asset_id string) (uint, error) {
	assetLock.Lock()
	defer assetLock.Unlock()

	if assetId, ok := AssetMap[asset_id]; ok {
		return assetId, nil
	}

	assetMaxID++

	if succeed := setAsset(asset.Asset{ID: assetMaxID, AssetID: asset_id}); succeed {
		return assetMaxID, nil
	}

	return 0, fmt.Errorf("asset_id :%s created failed", asset_id)
}

func GetAssetID(id uint) (string, error) {
	if assetID, ok := AssetPairMap[id]; ok {
		return assetID, nil
	}

	return "", fmt.Errorf("asset id :%d get failed", id)
}

func setAsset(asset asset.Asset) bool {
	if AssetMap == nil {
		AssetMap = map[string]uint{}
	}
	if AssetPairMap == nil {
		AssetPairMap = map[uint]string{}
	}
	AssetMap[asset.AssetID] = asset.ID
	AssetPairMap[asset.ID] = asset.AssetID

	return true
}

func GetAssetMaxID() uint {
	return assetMaxID
}
