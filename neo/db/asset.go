package db

import (
	"neo_explorer/neo/asset"
)

func GetAssetInfo() []asset.Asset {
	const query = "SELECT `id`, `asset_id` FROM `asset`"

	result := []asset.Asset{}
	rows, err := wrappedQuery(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		m := asset.Asset{}

		err := rows.Scan(
			&m.ID,
			&m.AssetID,
		)

		if err != nil {
			panic(err)
		}


		result = append(result, m)
	}

	return result
}

