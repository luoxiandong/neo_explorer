##  实现功能

1. 获取主币、合约币的utxo (`utxo`,`asset`等)

    ```
	SELECT
	    utxo.address_id,
	    utxo.value,
		  utxo.n,
	    asset.name AS asset_symbol,
			asset.asset_id	as asset_hash,
			tx.txid
	FROM
	    utxo
	    LEFT JOIN asset ON utxo.`asset_id` = asset.`asset_id`
			LEFT JOIN tx  on utxo.tx_id=tx.id
			LEFT JOIN address on address.id = utxo.address_id
	WHERE
	    address.address = 'AM915nkDP6nDWCLuHTodmCHr5DCfb7XdY7' and ISNULL(utxo.used_in_tx)=1;
    ```
2. 根据address和symol获取余额 (`addr_asset`,`asset`等)

    ```
    SELECT
        addr_asset.address_id,
        asset.name,
        addr_asset.asset_id,
        addr_asset.balance
    FROM
        addr_asset
        JOIN asset ON addr_asset. `asset_id` = asset. `id`
    WHERE
        addr_asset. `address_id` = 1
        AND asset. `name` = "NEO";
    ```
3. 根据address获取symbol列表 (`addr_asset`,`asset`等)

    ```
	   SELECT
	    asset.name,
	    asset.asset_id as contract,
			asset.`precision`,
			addr_asset.balance,
			address.address
	FROM
	    addr_asset
	    LEFT JOIN asset ON addr_asset. `asset_id` = asset. `asset_id`
			LEFT JOIN address ON addr_asset.address_id = address.id
	WHERE
	    address.address='AM915nkDP6nDWCLuHTodmCHr5DCfb7XdY7';
    ```
4. 根据address和symbol获取历史交易记录 (`asset_tx`,`asset`等)

    ```
    SELECT
        asset_tx. `address_id`,
        tx. `txid`,
        asset_tx.id ,
        asset. `asset_id`,
        asset. `name`
    FROM
        asset_tx
        LEFT JOIN asset ON asset.id = asset_tx.asset_id
        LEFT JOIN tx ON tx. `id` = asset_tx. `tx_id`
    WHERE
        asset_tx. `address_id` = 1
        AND asset. `name` = "NEO"
    ```
 5.交易详情
 ```
	  -- trans
	 SELECT tx.* 
	 FROM tx 
	 WHERE tx.txid='0x78402ab5bc691ab51e8189808a03cf835a96f7ae5e0fe86c327a802402851a5b';

	 -- trans vin
	 SELECT tx_vin.vout AS n,tx.txid,utxo.`value`,asset.`name`,address.address 
	 FROM tx_vin
	 LEFT JOIN tx ON tx.id=tx_vin.tx_id
	 LEFT JOIN utxo ON utxo.tx_id=tx_vin.txid AND utxo.n=tx_vin.vout
	 LEFT JOIN asset ON asset.asset_id=utxo.asset_id
	 LEFT JOIN address ON address.id=utxo.address_id
	 WHERE tx_vin.tx_id='42752393'

	-- trans vout
	SELECT tx_vout.n,tx.txid,tx_vout.`value`,asset.`name`,tx_vout.address
	FROM tx_vout
	LEFT JOIN tx ON tx.id=tx_vout.tx_id
	LEFT JOIN asset ON asset.asset_id=tx_vout.asset_id
	WHERE tx_vout.tx_id='42752393'
	
	-- nep5
	SELECT nep5_tx.`from`,nep5_tx.`to`,nep5_tx.`value`,nep5.`name`,nep5.symbol,nep5.asset_id as contract
	FROM nep5_tx
	LEFT JOIN nep5 ON nep5_tx.asset_id=nep5.asset_id
	WHERE nep5_tx.tx_id='42834670';
 ```

## 配置使用

1. 数据库： `./sqls/create_table.sql`
2. 配置文件： `cp config.sample.json config.json` (根据本地情况修改配置参数)
3. 运行： `go build && ./neo_explorer`
