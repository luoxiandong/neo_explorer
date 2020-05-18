##  实现功能

1. 获取主币、合约币的utxo (`utxo`,`asset`等)

    ```
    SELECT
        utxo.address_id,
        utxo.value,
        utxo.asset_id,
        asset.name AS asset_symbol
    FROM
        utxo
        JOIN asset ON utxo. `asset_id` = asset. `id`
    WHERE
        utxo.address_id = 1;
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
        addr_asset.address_id,
        asset.name,
        asset.asset_id
   FROM
        addr_asset
        JOIN asset ON addr_asset. `asset_id` = asset. `id`
   WHERE
        addr_asset. `address_id` = 1;
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

## 配置使用

1. 数据库： `./sqls/create_table.sql`
2. 配置文件： `cp config.sample.json config.json` (根据本地情况修改配置参数)
3. 运行： `go build && ./neo_explorer`