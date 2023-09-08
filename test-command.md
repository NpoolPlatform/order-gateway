# Order订单Mock

## 查找可用设备

```
mysql> select id from good_manager.device_infos where deleted_at=0 limit 3;
```

```
+--------------------------------------+
| id                                   |
+--------------------------------------+
| 00b7009e-34b2-4f55-88f8-dea3ac23e421 |
| 00b985d4-faa8-4994-b58a-e42d375f221a |
| 00bd89c7-1792-4182-b8b4-9547cce6de0c |
+--------------------------------------+
```

这里可以选用 id 00b7009e-34b2-4f55-88f8-dea3ac23e421



## 查找测试币种

```
mysql> select id,coin_type_id,name from chain_manager.app_coins where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and name like '%usdt%';
```

```
+--------------------------------------+--------------------------------------+------------+
| id                                   | coin_type_id                         | name       |
+--------------------------------------+--------------------------------------+------------+
| 4c8fb477-58da-49f5-aa1c-5e72120754c2 | 9ba0bfee-3dcf-4905-a95e-852436af748f | tusdttrc20 |
| 78e493d0-e78c-4efe-b537-32569987ee81 | 9c81fd06-ce50-466b-84e1-568018f00c8c | usdttrc20  |
| 80b846de-f89a-4543-9a37-5448465bbbb0 | 9ba0bfee-3dcf-4905-a95e-852436af748f | tusdttrc20 |
| d936fc39-c3ba-4d0f-aced-88c04bd8fb2e | aaf04c25-2c87-46e7-99d3-56814e40ec61 | tusdterc20 |
+--------------------------------------+--------------------------------------+------------+
```

这里可以选用 coin_type_id 9ba0bfee-3dcf-4905-a95e-852436af748f



## 查找location

```
mysql> select id from good_manager.vendor_locations where deleted_at=0 limit 3;
```

```
+--------------------------------------+
| id                                   |
+--------------------------------------+
| 001a98f0-ee9f-4d5e-a1f4-f4739221136b |
| 0082cdb6-cb51-4942-8c7d-661b528cc69c |
| 00ed8b24-0722-4188-9ddb-1284d68d56f3 |
+--------------------------------------+
```

这里可以选用 id 001a98f0-ee9f-4d5e-a1f4-f4739221136b



## 创建商品Good

这里可以指定ID，也可以不指定由程序自动生成，下一步创建appGood的时候需要设置goodID与这里指定或者生成的ID相同

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "1001a09e-34b2-4f55-88f8-dea3ac23e421",
        "DeviceInfoID": "00b7009e-34b2-4f55-88f8-dea3ac23e421",
        "DurationDays": "365",
        "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
        "VendorLocationID": "001a98f0-ee9f-4d5e-a1f4-f4739221136b",
        "Price": "10",
        "BenefitType": "BenefitTypePlatform",
        "GoodType": "PowerRenting",
        "Title": "test-20230907-01",
        "Unit": "TiB",
        "UnitAmount": "1",
        "SupportCoinTypeIDs": [
            "9ba0bfee-3dcf-4905-a95e-852436af748f"
        ],
        "DeliveryAt": "1694149200",
        "StartAt": "1693976400",
        "StartMode": "GoodStartModeConfirmed",
        "TestOnly": true,
        "Total": "100",
        "BenefitIntervalHours": "24",
        "UnitLockDeposit": "10"
    }
}' good-middleware:50531 good.middleware.good1.v1.Middleware.CreateGood
```



## 创建应用绑定商品AppGood

这里使用appID ff2c5d50-be56-413e-aba5-9c7ad888a769。或者在数据库查找其他的AppID也可以。

ID可以指定也可以程序自动生成

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2001aa50-be56-413e-aba5-9c7ad888a769",
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "GoodID": "1001a09e-34b2-4f55-88f8-dea3ac23e421",
        "Online": true,
        "Visible": true,
        "GoodName": "testbenefit20230906-15",
        "Price": "50",
        "DisplayIndex": "1",
        "PurchaseLimit": "100",
        "SaleStartAt": "1693976400",
        "SaleEndAt": "1694149200",
        "ServiceStartAt": "1693976400",
        "Descriptions": "1694149200",
        "GoodBanner": "",
        "EnablePurchase": true,
        "EnableProductPage": true,
        "CancelMode": "CancellableBeforeBenefit",
        "UserPurchaseLimit": "200",
        "DisplayColors": "",
        "CancellableBeforeStart": "0",
        "ProductPage": "",
        "EnableSetCommission": true,
        "Posters": "",
        "TechnicalFeeRatio": "20",
        "ElectricityFeeRatio": "0",
        "DisplayNames": ""
    }
}' good-middleware:50531 good.middleware.app.good1.v1.Middleware.CreateGood
```



## 找个有钱的用户

```
mysql> select user_id,spendable from ledger_manager.generals where coin_type_id='9ba0bfee-3dcf-4905-a95e-852436af748f' and app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and deleted_at=0 limit 10;
```

```
+--------------------------------------+-----------------------------------+
| user_id                              | spendable                         |
+--------------------------------------+-----------------------------------+
| fba0bd90-99b2-44e1-88e8-5fdfad2dc9f0 |              0.000000000000000000 |
| 15cf1283-634a-4008-9913-c9a9235316a9 | 20000000000350.294000000000000000 |
| c48cf817-0b54-476f-9962-6379203a562a |              0.000000000000000000 |
| 628db1e7-2fd9-4468-a785-a434ba5849bc |          49995.000000000000000000 |
| 732d2a46-3c71-448f-8f92-067ff11634e1 |            980.000000000000000000 |
| b36df48a-3581-442b-b5ad-83ecf6effcdd |          99506.000000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce1234 |            280.000000000000000000 |
| 297d7c7c-ea54-4843-8502-2c3b925f2749 |            107.888000000000000000 |
| 8c14fb2f-14f9-4656-84c3-e7ef104e9d58 |              5.000000000000000000 |
| 9bfb1441-7090-4cff-9451-7ada311bf736 |            223.126000000000000000 |
+--------------------------------------+-----------------------------------+
```

这里可以选用 

有钱的user_id b36df48a-3581-442b-b5ad-83ecf6effcdd

没钱的user_id 8c14fb2f-14f9-4656-84c3-e7ef104e9d58



## 创建订单

1. 没有用余额支付

```
grpcurl --plaintext -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "b36df48a-3581-442b-b5ad-83ecf6effcdd",
    "AppGoodID": "2001aa50-be56-413e-aba5-9c7ad888a769",
    "Units": "10",
    "PaymentCoinID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
    "InvestmentType": "FullPayment"
}' localhost:50431 order.gateway.order1.v1.Gateway.CreateOrder
```

2. 有余额支付

```
grpcurl --plaintext -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "b36df48a-3581-442b-b5ad-83ecf6effcdd",
    "AppGoodID": "2001aa50-be56-413e-aba5-9c7ad888a769",
    "Units": "20",
    "PaymentCoinID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
    "PayWithBalanceAmount": "10000",
    "InvestmentType": "FullPayment"
}' localhost:50431 order.gateway.order1.v1.Gateway.CreateOrder
```





## 批量创建订单（父子订单）

场景描述：

在good中有一个goodRequired模块，设置了主商品与关联商品的绑定关系，如MainGood是A商品，RequiredGood是B和C商品，其中B商品的must属性为true，则表示在购买A商品时必须购买B商品，而C商品为可选商品，在下单时， 会批量创建order，A商品的order为父订单，B商品的order为子订单，如果C商品也同时购买，则也是且支付金额会汇总到A商品的order中计算



## 创建一组商品Good

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2001a09e-34b2-4f55-88f8-dea3ac23e421",
        "DeviceInfoID": "00b7009e-34b2-4f55-88f8-dea3ac23e421",
        "DurationDays": "365",
        "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
        "VendorLocationID": "001a98f0-ee9f-4d5e-a1f4-f4739221136b",
        "Price": "10",
        "BenefitType": "BenefitTypePlatform",
        "GoodType": "PowerRenting",
        "Title": "test-20230907-21",
        "Unit": "TiB",
        "UnitAmount": "1",
        "SupportCoinTypeIDs": [
            "9ba0bfee-3dcf-4905-a95e-852436af748f"
        ],
        "DeliveryAt": "1694149200",
        "StartAt": "1693976400",
        "StartMode": "GoodStartModeConfirmed",
        "TestOnly": true,
        "Total": "100",
        "BenefitIntervalHours": "24",
        "UnitLockDeposit": "10"
    }
}' good-middleware:50531 good.middleware.good1.v1.Middleware.CreateGood
```

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2002a09e-34b2-4f55-88f8-dea3ac23e421",
        "DeviceInfoID": "00b7009e-34b2-4f55-88f8-dea3ac23e421",
        "DurationDays": "365",
        "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
        "VendorLocationID": "001a98f0-ee9f-4d5e-a1f4-f4739221136b",
        "Price": "10",
        "BenefitType": "BenefitTypePlatform",
        "GoodType": "PowerRenting",
        "Title": "test-20230907-22",
        "Unit": "TiB",
        "UnitAmount": "1",
        "SupportCoinTypeIDs": [
            "9ba0bfee-3dcf-4905-a95e-852436af748f"
        ],
        "DeliveryAt": "1694149200",
        "StartAt": "1693976400",
        "StartMode": "GoodStartModeConfirmed",
        "TestOnly": true,
        "Total": "100",
        "BenefitIntervalHours": "24",
        "UnitLockDeposit": "10"
    }
}' good-middleware:50531 good.middleware.good1.v1.Middleware.CreateGood
```

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2003a09e-34b2-4f55-88f8-dea3ac23e421",
        "DeviceInfoID": "00b7009e-34b2-4f55-88f8-dea3ac23e421",
        "DurationDays": "365",
        "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
        "VendorLocationID": "001a98f0-ee9f-4d5e-a1f4-f4739221136b",
        "Price": "10",
        "BenefitType": "BenefitTypePlatform",
        "GoodType": "PowerRenting",
        "Title": "test-20230907-23",
        "Unit": "TiB",
        "UnitAmount": "1",
        "SupportCoinTypeIDs": [
            "9ba0bfee-3dcf-4905-a95e-852436af748f"
        ],
        "DeliveryAt": "1694149200",
        "StartAt": "1693976400",
        "StartMode": "GoodStartModeConfirmed",
        "TestOnly": true,
        "Total": "100",
        "BenefitIntervalHours": "24",
        "UnitLockDeposit": "10"
    }
}' good-middleware:50531 good.middleware.good1.v1.Middleware.CreateGood
```



## 创建一组应用绑定商品AppGood

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2201aa50-be56-413e-aba5-9c7ad888a769",
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "GoodID": "2001a09e-34b2-4f55-88f8-dea3ac23e421",
        "Online": true,
        "Visible": true,
        "GoodName": "testbenefit20230906-15",
        "Price": "50",
        "DisplayIndex": "1",
        "PurchaseLimit": "100",
        "SaleStartAt": "1693976400",
        "SaleEndAt": "1694149200",
        "ServiceStartAt": "1693976400",
        "Descriptions": "1694149200",
        "GoodBanner": "",
        "EnablePurchase": true,
        "EnableProductPage": true,
        "CancelMode": "CancellableBeforeBenefit",
        "UserPurchaseLimit": "200",
        "DisplayColors": "",
        "CancellableBeforeStart": "0",
        "ProductPage": "",
        "EnableSetCommission": true,
        "Posters": "",
        "TechnicalFeeRatio": "20",
        "ElectricityFeeRatio": "0",
        "DisplayNames": ""
    }
}' good-middleware:50531 good.middleware.app.good1.v1.Middleware.CreateGood
```

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2202aa50-be56-413e-aba5-9c7ad888a769",
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "GoodID": "2002a09e-34b2-4f55-88f8-dea3ac23e421",
        "Online": true,
        "Visible": true,
        "GoodName": "testbenefit20230906-15",
        "Price": "50",
        "DisplayIndex": "1",
        "PurchaseLimit": "100",
        "SaleStartAt": "1693976400",
        "SaleEndAt": "1694149200",
        "ServiceStartAt": "1693976400",
        "Descriptions": "1694149200",
        "GoodBanner": "",
        "EnablePurchase": true,
        "EnableProductPage": true,
        "CancelMode": "CancellableBeforeBenefit",
        "UserPurchaseLimit": "200",
        "DisplayColors": "",
        "CancellableBeforeStart": "0",
        "ProductPage": "",
        "EnableSetCommission": true,
        "Posters": "",
        "TechnicalFeeRatio": "20",
        "ElectricityFeeRatio": "0",
        "DisplayNames": ""
    }
}' good-middleware:50531 good.middleware.app.good1.v1.Middleware.CreateGood
```

```
grpcurl --plaintext -d '{
    "Info": {
    	"ID": "2203aa50-be56-413e-aba5-9c7ad888a769",
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "GoodID": "2003a09e-34b2-4f55-88f8-dea3ac23e421",
        "Online": true,
        "Visible": true,
        "GoodName": "testbenefit20230906-15",
        "Price": "50",
        "DisplayIndex": "1",
        "PurchaseLimit": "100",
        "SaleStartAt": "1693976400",
        "SaleEndAt": "1694149200",
        "ServiceStartAt": "1693976400",
        "Descriptions": "1694149200",
        "GoodBanner": "",
        "EnablePurchase": true,
        "EnableProductPage": true,
        "CancelMode": "CancellableBeforeBenefit",
        "UserPurchaseLimit": "200",
        "DisplayColors": "",
        "CancellableBeforeStart": "0",
        "ProductPage": "",
        "EnableSetCommission": true,
        "Posters": "",
        "TechnicalFeeRatio": "20",
        "ElectricityFeeRatio": "0",
        "DisplayNames": ""
    }
}' good-middleware:50531 good.middleware.app.good1.v1.Middleware.CreateGood
```



## 创建关联商品

这里设置第一个商品为主商品，第二个商品在第一个商品购买时必须同时购买，第三个商品为可选商品

```
grpcurl --plaintext -d '{
    "Info": {
        "MainGoodID": "2001a09e-34b2-4f55-88f8-dea3ac23e421",
        "RequiredGoodID": "2002a09e-34b2-4f55-88f8-dea3ac23e421",
        "Must": true
    }
}' localhost:50531 good.middleware.good1.required1.v1.Middleware.CreateRequired
```

```
grpcurl --plaintext -d '{
    "Info": {
        "MainGoodID": "2001a09e-34b2-4f55-88f8-dea3ac23e421",
        "RequiredGoodID": "2003a09e-34b2-4f55-88f8-dea3ac23e421",
        "Must": false
    }
}' localhost:50531 good.middleware.good1.required1.v1.Middleware.CreateRequired
```



## 批量创建订单

```
grpcurl --plaintext -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "b36df48a-3581-442b-b5ad-83ecf6effcdd",
    "PaymentCoinID": "2ab32b87-0696-46f2-92f6-a02c9b39fe4c",
    "InvestmentType": "FullPayment",
    "Orders": [
        {
            "AppGoodID": "2201aa50-be56-413e-aba5-9c7ad888a769",
            "Units": "10",
            "Parent": true
        },
        {
            "AppGoodID": "2202aa50-be56-413e-aba5-9c7ad888a769",
            "Units": "20",
            "Parent": false
        },
        {
            "AppGoodID": "2203aa50-be56-413e-aba5-9c7ad888a769",
            "Units": "20",
            "Parent": false
        }
    ]
}' localhost:50431 order.gateway.order1.v1.Gateway.CreateOrders
```

