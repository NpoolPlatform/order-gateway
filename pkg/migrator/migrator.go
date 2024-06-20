//nolint:dupl
package migrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	servicename "github.com/NpoolPlatform/order-gateway/pkg/servicename"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
	"github.com/NpoolPlatform/order-middleware/pkg/db/ent"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	keyServiceID = "serviceid"
)

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(servicename.ServiceDomain, keyServiceID)
	return fmt.Sprintf("migrator:%v", serviceID)
}

func migrateAppConfigs(ctx context.Context, tx *ent.Tx) error {
	fmt.Println("======================== exec migrateAppConfigs ==========================")
	rows, err := tx.QueryContext(ctx, "select ent_id,app_id,enabled,send_coupon_mode,send_coupon_probability,cashable_profit_probability,created_at,updated_at from simulate_configs where deleted_at = 0") //nolint
	if err != nil {
		return err
	}

	type SimulateConfig struct {
		EntID                     uuid.UUID `json:"ent_id"`
		AppID                     uuid.UUID `json:"app_id"`
		Enabled                   bool
		SendCouponMode            string
		SendCouponProbability     string
		CashableProfitProbability string
		CreatedAt                 uint32
		UpdatedAt                 uint32
	}
	simulateConfigs := []*SimulateConfig{}
	for rows.Next() {
		sc := &SimulateConfig{}
		if err := rows.Scan(&sc.EntID, &sc.AppID, &sc.Enabled, &sc.SendCouponMode, &sc.SendCouponProbability, &sc.CashableProfitProbability, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return err
		}
		simulateConfigs = append(simulateConfigs, sc)
	}
	for _, sc := range simulateConfigs {
		sendCouponProbability, err := decimal.NewFromString(sc.SendCouponProbability)
		if err != nil {
			return err
		}
		cashableProfitProbability, err := decimal.NewFromString(sc.CashableProfitProbability)
		if err != nil {
			return err
		}
		// if _, err := tx.
		// 	AppConfig.
		// 	Create().
		// 	SetAppID(sc.AppID).
		// 	SetEnableSimulateOrder(sc.Enabled).
		// 	SetSimulateOrderCouponMode(sc.SendCouponMode).
		// 	SetSimulateOrderCouponProbability(sendCouponProbability).
		// 	SetSimulateOrderCashableProfitProbability(cashableProfitProbability).
		// 	SetCreatedAt(sc.CreatedAt).
		// 	SetUpdatedAt(sc.UpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("1 -------------------------migrateAppConfigs-----------------------------")
		fmt.Println("sc.AppID: ", sc.AppID)
		fmt.Println("sc.Enabled: ", sc.Enabled)
		fmt.Println("sc.SendCouponMode: ", sc.SendCouponMode)
		fmt.Println("sendCouponProbability: ", sendCouponProbability)
		fmt.Println("cashableProfitProbability: ", cashableProfitProbability)
		fmt.Println("sc.CreatedAt: ", sc.CreatedAt)
		fmt.Println("sc.UpdatedAt: ", sc.UpdatedAt)
		fmt.Println("11 -------------------------migrateAppConfigs-----------------------------")
	}
	return nil
}

func migrateOrderLocks(ctx context.Context, tx *ent.Tx) error {
	fmt.Println("======================== exec migrateOrderLocks ==========================")
	sql := "alter table test_locks modify app_id varchar(36);"
	fmt.Println("exec sql: ", sql)
	// rc, err := tx.ExecContext(ctx, sql)
	// if err != nil {
	// 	return err
	// }
	// _, err = rc.RowsAffected()
	// if err != nil {
	// 	return fmt.Errorf("fail modify test_locks: %v", err)
	// }
	return nil
}

//nolint:funlen,gocyclo
func migratePowerRentals(ctx context.Context, tx *ent.Tx) error {
	fmt.Println("======================== exec migratePowerRentals ==========================")
	selectOrderSql := fmt.Sprintf("select od.ent_id,od.app_id,od.user_id,od.good_id,od.app_good_id,od.parent_order_id,od.order_type,od.create_method,od.simulate,od.coupon_ids,od.payment_type,od.units_v1,od.good_value_usd,od.payment_amount,od.discount_amount,od.promotion_id,od.investment_type,od.duration,od.payment_id,od.payment_coin_type_id,od.balance_amount,od.coin_usd_currency,od.local_coin_usd_currency,od.live_coin_usd_currency,od.created_at as order_created_at,od.updated_at as order_updated_at,os.cancel_state,os.paid_at,os.user_set_paid,os.user_set_canceled,os.admin_set_canceled,os.payment_state,os.renew_state,os.renew_notify_at,os.order_state,os.start_mode,os.start_at,os.last_benefit_at,os.benefit_state,os.payment_finish_amount,os.created_at as order_state_created_at,os.updated_at as order_state_updated_at,pm.ent_id as payment_ent_id,pm.account_id,pm.start_amount,pm.created_at as payment_created_at,pm.updated_at as payment_updated_at,ol.ent_id as ledger_lock_id,ol.created_at as lock_created_at,ol.updated_at as lock_updated_at from orders as od inner join order_states os on od.ent_id=os.order_id and os.deleted_at=0 left join payments as pm on od.ent_id=pm.order_id and pm.deleted_at=0 left join order_locks as ol on od.ent_id=ol.order_id and ol.deleted_at=0 and ol.lock_type='LockBalance' where od.parent_order_id='%v' and od.deleted_at=0", uuid.Nil.String()) //nolint
	fmt.Println("exec selectOrderSql: ", selectOrderSql)
	orderRows, err := tx.QueryContext(ctx, selectOrderSql)
	if err != nil {
		return err
	}

	type Order struct {
		EntID                uuid.UUID `json:"ent_id"`
		AppID                uuid.UUID `json:"app_id"`
		UserID               uuid.UUID `json:"user_id"`
		GoodID               uuid.UUID `json:"good_id"`
		AppGoodID            uuid.UUID `json:"app_good_id"`
		ParentOrderID        uuid.UUID `json:"parent_order_id"`
		OrderType            string
		CreateMethod         string
		Simulate             bool
		CouponIDsStr         sql.NullString
		CouponIDs            string
		PaymentType          string
		UnitsV1              string
		GoodValueUsd         string
		PaymentAmount        string
		DiscountAmount       string
		PromotionID          uuid.UUID `json:"promotion_id"`
		InvestmentType       string
		Duration             uint32
		PaymentID            uuid.UUID `json:"payment_id"`
		PaymentCoinTypeID    uuid.UUID `json:"payment_coin_type_id"`
		BalanceAmount        string
		CoinUsdCurrency      string
		LocalCoinUsdCurrency string
		LiveCoinUsdCurrency  string
		OrderCreatedAt       uint32
		OrderUpdatedAt       uint32
		CancelState          string
		PaidAt               uint32
		UserSetPaid          bool
		UserSetCanceled      bool
		AdminSetCanceled     bool
		PaymentState         string
		RenewState           string
		RenewNotifyAt        uint32
		OrderState           string
		StartMode            string
		StartAt              uint32
		LastBenefitAt        uint32
		BenefitState         string
		PaymentFinishAmount  string
		OrderStateCreatedAt  uint32
		OrderStateUpdatedAt  uint32
		PaymentEntID         uuid.UUID `json:"payment_ent_id"`
		AccountID            uuid.UUID `json:"account_id"`
		StartAmountStr       sql.NullString
		StartAmount          string
		PaymentCreatedAtInt  sql.NullInt32
		PaymentUpdatedAtInt  sql.NullInt32
		PaymentCreatedAt     uint32
		PaymentUpdatedAt     uint32
		LedgerLockID         uuid.UUID `json:"ledger_lock_id"`
		LockCreatedAtInt     sql.NullInt32
		LockUpdatedAtInt     sql.NullInt32
		LockCreatedAt        uint32
		LockUpdatedAt        uint32
	}
	orders := map[uuid.UUID]*Order{}
	appGoodIDMap := map[uuid.UUID]string{}
	for orderRows.Next() {
		od := &Order{}
		if err := orderRows.Scan(&od.EntID, &od.AppID, &od.UserID, &od.GoodID, &od.AppGoodID,
			&od.ParentOrderID, &od.OrderType, &od.CreateMethod, &od.Simulate, &od.CouponIDsStr,
			&od.PaymentType, &od.UnitsV1, &od.GoodValueUsd, &od.PaymentAmount, &od.DiscountAmount,
			&od.PromotionID, &od.InvestmentType, &od.Duration, &od.PaymentID, &od.PaymentCoinTypeID,
			&od.BalanceAmount, &od.CoinUsdCurrency, &od.LocalCoinUsdCurrency, &od.LiveCoinUsdCurrency,
			&od.OrderCreatedAt, &od.OrderUpdatedAt, &od.CancelState, &od.PaidAt, &od.UserSetPaid,
			&od.UserSetCanceled, &od.AdminSetCanceled, &od.PaymentState, &od.RenewState, &od.RenewNotifyAt,
			&od.OrderState, &od.StartMode, &od.StartAt, &od.LastBenefitAt, &od.BenefitState, &od.PaymentFinishAmount,
			&od.OrderStateCreatedAt, &od.OrderStateUpdatedAt, &od.PaymentEntID, &od.AccountID, &od.StartAmountStr,
			&od.PaymentCreatedAtInt, &od.PaymentUpdatedAtInt, &od.LedgerLockID, &od.LockCreatedAtInt, &od.LockUpdatedAtInt,
		); err != nil {
			return err
		}
		od.StartAmount = decimal.NewFromInt(0).String()
		if od.StartAmountStr.Valid {
			od.StartAmount = od.StartAmountStr.String
		}
		od.CouponIDs = "[]"
		if od.CouponIDsStr.Valid && od.CouponIDsStr.String != "null" {
			od.CouponIDs = od.CouponIDsStr.String
		}
		od.PaymentCreatedAt = uint32(0)
		if od.PaymentCreatedAtInt.Valid {
			od.PaymentCreatedAt = uint32(od.PaymentCreatedAtInt.Int32)
		}
		od.PaymentUpdatedAt = uint32(0)
		if od.PaymentUpdatedAtInt.Valid {
			od.PaymentUpdatedAt = uint32(od.PaymentUpdatedAtInt.Int32)
		}
		od.LockCreatedAt = uint32(0)
		if od.LockCreatedAtInt.Valid {
			od.LockCreatedAt = uint32(od.LockCreatedAtInt.Int32)
		}
		od.LockUpdatedAt = uint32(0)
		if od.LockUpdatedAtInt.Valid {
			od.LockUpdatedAt = uint32(od.LockUpdatedAtInt.Int32)
		}
		orders[od.EntID] = od
		appGoodIDMap[od.AppGoodID] = od.AppGoodID.String()
	}
	fmt.Println("2 len orders=========== ", len(orders))
	fmt.Println("2 len appGoodIDs======= ", len(appGoodIDMap))

	appGoodIDs := ""
	comm := ""
	for _, id := range appGoodIDMap {
		appGoodIDs += fmt.Sprintf("%v'%v'", comm, id)
		if comm == "" {
			comm = ","
		}
	}

	selectAppGoodStockSQL := fmt.Sprintf("select ent_id, app_good_id from good_manager.app_stocks where app_good_id in(%v) and deleted_at=0", appGoodIDs)
	fmt.Println("exec selectAppGoodStockSQL: ", selectAppGoodStockSQL)
	appGoodStockRows, err := tx.QueryContext(ctx, selectAppGoodStockSQL)
	if err != nil {
		return err
	}

	type AppGoodStock struct {
		EntID     uuid.UUID `json:"ent_id"`
		AppGoodID uuid.UUID `json:"app_good_id"`
	}
	appStockMap := map[uuid.UUID]uuid.UUID{}
	for appGoodStockRows.Next() {
		ags := &AppGoodStock{}
		if err := appGoodStockRows.Scan(&ags.EntID, &ags.AppGoodID); err != nil {
			return err
		}
		appStockMap[ags.AppGoodID] = ags.EntID
	}

	defaultGoodType := "LegacyPowerRental"
	defaultPaymentObseleteState := "PaymentObseleteNone"
	for _, order := range orders {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>><<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
		fmt.Println("============== order ================")
		fmt.Println("EntID: ", order.EntID)
		fmt.Println("AppID: ", order.AppID)
		fmt.Println("UserID: ", order.UserID)
		fmt.Println("GoodID: ", order.GoodID)
		fmt.Println("AppGoodID: ", order.AppGoodID)
		fmt.Println("ParentOrderID: ", order.ParentOrderID)
		fmt.Println("OrderType: ", order.OrderType)
		fmt.Println("CreateMethod: ", order.CreateMethod)
		fmt.Println("Simulate: ", order.Simulate)
		fmt.Println("CouponIDsStr: ", order.CouponIDsStr)
		fmt.Println("CouponIDs: ", order.CouponIDs)
		fmt.Println("PaymentType: ", order.PaymentType)
		fmt.Println("UnitsV1: ", order.UnitsV1)
		fmt.Println("GoodValueUsd: ", order.GoodValueUsd)
		fmt.Println("PaymentAmount: ", order.PaymentAmount)
		fmt.Println("DiscountAmount: ", order.DiscountAmount)
		fmt.Println("PromotionID: ", order.PromotionID)
		fmt.Println("InvestmentType: ", order.InvestmentType)
		fmt.Println("Duration: ", order.Duration)
		fmt.Println("PaymentID: ", order.PaymentID)
		fmt.Println("PaymentCoinTypeID: ", order.PaymentCoinTypeID)
		fmt.Println("BalanceAmount: ", order.BalanceAmount)
		fmt.Println("CoinUsdCurrency: ", order.CoinUsdCurrency)
		fmt.Println("LocalCoinUsdCurrency: ", order.LocalCoinUsdCurrency)
		fmt.Println("LiveCoinUsdCurrency: ", order.LiveCoinUsdCurrency)
		fmt.Println("OrderCreatedAt: ", order.OrderCreatedAt)
		fmt.Println("OrderUpdatedAt: ", order.OrderUpdatedAt)
		fmt.Println("CancelState: ", order.CancelState)
		fmt.Println("PaidAt: ", order.PaidAt)
		fmt.Println("UserSetPaid: ", order.UserSetPaid)
		fmt.Println("UserSetCanceled: ", order.UserSetCanceled)
		fmt.Println("AdminSetCanceled: ", order.AdminSetCanceled)
		fmt.Println("PaymentState: ", order.PaymentState)
		fmt.Println("RenewState: ", order.RenewState)
		fmt.Println("RenewNotifyAt: ", order.RenewNotifyAt)
		fmt.Println("OrderState: ", order.OrderState)
		fmt.Println("StartMode: ", order.StartMode)
		fmt.Println("StartAt: ", order.StartAt)
		fmt.Println("LastBenefitAt: ", order.LastBenefitAt)
		fmt.Println("BenefitState: ", order.BenefitState)
		fmt.Println("PaymentFinishAmount: ", order.PaymentFinishAmount)
		fmt.Println("OrderStateCreatedAt: ", order.OrderStateCreatedAt)
		fmt.Println("OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
		fmt.Println("PaymentEntID: ", order.PaymentEntID)
		fmt.Println("AccountID: ", order.AccountID)
		fmt.Println("StartAmountStr: ", order.StartAmountStr)
		fmt.Println("StartAmount: ", order.StartAmount)
		fmt.Println("PaymentCreatedAtInt: ", order.PaymentCreatedAtInt)
		fmt.Println("PaymentCreatedAt: ", order.PaymentCreatedAt)
		fmt.Println("PaymentUpdatedAtInt: ", order.PaymentUpdatedAtInt)
		fmt.Println("PaymentUpdatedAt: ", order.PaymentUpdatedAt)
		fmt.Println("LedgerLockID: ", order.LedgerLockID)
		fmt.Println("LockCreatedAtInt: ", order.LockCreatedAtInt)
		fmt.Println("LockCreatedAt: ", order.LockCreatedAt)
		fmt.Println("LockUpdatedAtInt: ", order.LockUpdatedAtInt)
		fmt.Println("LockUpdatedAt: ", order.LockUpdatedAt)
		fmt.Println("============== order ================")
		appGoodStockID, ok := appStockMap[order.AppGoodID]
		if !ok {
			fmt.Println("-------------------------------- not found app good stock id --------------------------------: appGoodID: ", order.AppGoodID)
			continue
		}
		paymentID := order.PaymentID
		if order.PaymentID != uuid.Nil {
			// payment transter
			// if _, err := tx.
			// 	PaymentBase.
			// 	Create().
			// 	SetEntID(order.PaymentID).
			// 	SetOrderID(order.EntID).
			// 	SetObseleteState(defaultPaymentObseleteState).
			// 	SetCreatedAt(order.PaymentCreatedAt).
			// 	SetUpdatedAt(order.PaymentUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("2 -------- create payment base ---------")
			fmt.Println("order.PaymentID: ", order.PaymentID)
			fmt.Println("order.EntID: ", order.EntID)
			fmt.Println("defaultPaymentObseleteState: ", defaultPaymentObseleteState)
			fmt.Println("order.PaymentCreatedAt: ", order.PaymentCreatedAt)
			fmt.Println("order.PaymentUpdatedAt: ", order.PaymentUpdatedAt)
			fmt.Println("22 ------- create payment base ---------")

			paymentAmount, err := decimal.NewFromString(order.PaymentAmount)
			if err != nil {
				return fmt.Errorf("invalid paymentAmount")
			}
			startAmount, err := decimal.NewFromString(order.StartAmount)
			if err != nil {
				return fmt.Errorf("invalid startAmount")
			}
			finishAmount, err := decimal.NewFromString(order.PaymentFinishAmount)
			if err != nil {
				return fmt.Errorf("invalid finishAmount")
			}
			coinUsdCurrency, err := decimal.NewFromString(order.CoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid coinUsdCurrency")
			}
			localCoinUsdCurrency, err := decimal.NewFromString(order.LocalCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid localCoinUsdCurrency")
			}
			liveCoinUsdCurrency, err := decimal.NewFromString(order.LiveCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid liveCoinUsdCurrency")
			}
			id := uuid.New()
			// if _, err := tx.
			// 	PaymentTransfer.
			// 	Create().
			// 	SetEntID(id).
			// 	SetPaymentID(order.PaymentID).
			// 	SetCoinTypeID(order.PaymentCoinTypeID).
			// 	SetAccountID(order.AccountID).
			// 	SetAmount(paymentAmount).
			// 	SetStartAmount(startAmount).
			// 	SetFinishAmount(finishAmount).
			// 	SetCoinUsdCurrency(coinUsdCurrency).
			// 	SetLocalCoinUsdCurrency(localCoinUsdCurrency).
			// 	SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
			// 	SetCreatedAt(order.PaymentCreatedAt).
			// 	SetUpdatedAt(order.PaymentUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("2 -------- create payment transter ---------")
			fmt.Println("id: ", id)
			fmt.Println("order.PaymentID: ", order.PaymentID)
			fmt.Println("order.PaymentCoinTypeID: ", order.PaymentCoinTypeID)
			fmt.Println("order.AccountID: ", order.AccountID)
			fmt.Println("paymentAmount: ", paymentAmount)
			fmt.Println("startAmount: ", startAmount)
			fmt.Println("finishAmount: ", finishAmount)
			fmt.Println("coinUsdCurrency: ", coinUsdCurrency)
			fmt.Println("localCoinUsdCurrency: ", localCoinUsdCurrency)
			fmt.Println("liveCoinUsdCurrency: ", liveCoinUsdCurrency)
			fmt.Println("order.PaymentCreatedAt: ", order.PaymentCreatedAt)
			fmt.Println("order.PaymentUpdatedAt: ", order.PaymentUpdatedAt)
			fmt.Println("22 ------- create payment transter ---------")
		}

		orderBalanceAmount, err := decimal.NewFromString(order.BalanceAmount)
		if err != nil {
			return fmt.Errorf("invalid balanceamount")
		}

		if orderBalanceAmount.Cmp(decimal.NewFromInt(0)) > 0 {
			// payment balance
			if order.PaymentID == uuid.Nil {
				paymentID = uuid.New()
				fmt.Println("new paymentID: ", paymentID)
				// if _, err := tx.
				// 	PaymentBase.
				// 	Create().
				// 	SetEntID(paymentID).
				// 	SetOrderID(order.EntID).
				// 	SetObseleteState(defaultPaymentObseleteState).
				// 	SetCreatedAt(order.PaymentCreatedAt).
				// 	SetUpdatedAt(order.PaymentUpdatedAt).
				// 	Save(ctx); err != nil {
				// 	return err
				// }
				fmt.Println("2 -------- create payment base ---------")
				fmt.Println("paymentID: ", paymentID)
				fmt.Println("order.EntID: ", order.EntID)
				fmt.Println("defaultPaymentObseleteState: ", defaultPaymentObseleteState)
				fmt.Println("order.PaymentCreatedAt: ", order.PaymentCreatedAt)
				fmt.Println("order.PaymentUpdatedAt: ", order.PaymentUpdatedAt)
				fmt.Println("22 ------- create payment base ---------")
			}

			coinUsdCurrency, err := decimal.NewFromString(order.CoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid coinusdcurrency")
			}
			localCoinUsdCurrency, err := decimal.NewFromString(order.LocalCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid localcoinusdcurrency")
			}
			liveCoinUsdCurrency, err := decimal.NewFromString(order.LiveCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid livecoinusdcurrency")
			}
			id := uuid.New()
			// if _, err := tx.
			// 	PaymentBalance.
			// 	Create().
			// 	SetEntID(id).
			// 	SetPaymentID(paymentID).
			// 	SetCoinTypeID(order.PaymentCoinTypeID).
			// 	SetAmount(orderBalanceAmount).
			// 	SetCoinUsdCurrency(coinUsdCurrency).
			// 	SetLocalCoinUsdCurrency(localCoinUsdCurrency).
			// 	SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
			// 	SetCreatedAt(order.OrderCreatedAt).
			// 	SetUpdatedAt(order.OrderUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("2 -------- create payment balance ---------")
			fmt.Println("id: ", id)
			fmt.Println("order.PaymentID: ", paymentID)
			fmt.Println("order.PaymentCoinTypeID: ", order.PaymentCoinTypeID)
			fmt.Println("orderBalanceAmount: ", orderBalanceAmount)
			fmt.Println("coinUsdCurrency: ", coinUsdCurrency)
			fmt.Println("localCoinUsdCurrency: ", localCoinUsdCurrency)
			fmt.Println("liveCoinUsdCurrency: ", liveCoinUsdCurrency)
			fmt.Println("order.OrderCreatedAt: ", order.OrderCreatedAt)
			fmt.Println("order.OrderUpdatedAt: ", order.OrderUpdatedAt)
			fmt.Println("22 ------- create payment balance ---------")

			if order.LedgerLockID != uuid.Nil {
				id = uuid.New()
				// if _, err := tx.
				// 	PaymentBalanceLock.
				// 	Create().
				// 	SetEntID(id).
				// 	SetPaymentID(paymentID).
				// 	SetLedgerLockID(order.LedgerLockID).
				// 	SetCreatedAt(order.LockCreatedAt).
				// 	SetUpdatedAt(order.LockUpdatedAt).
				// 	Save(ctx); err != nil {
				// 	return err
				// }
				fmt.Println("2 -------- create payment balance lock ---------")
				fmt.Println("id: ", id)
				fmt.Println("order.PaymentID: ", paymentID)
				fmt.Println("order.LedgerLockID: ", order.LedgerLockID)
				fmt.Println("order.LockCreatedAt: ", order.LockCreatedAt)
				fmt.Println("order.LockUpdatedAt: ", order.LockUpdatedAt)
				fmt.Println("22 ------- create payment balance lock ---------")
			}
		}

		// if _, err := tx.
		// 	OrderBase.
		// 	Create().
		// 	SetEntID(order.EntID).
		// 	SetAppID(order.AppID).
		// 	SetUserID(order.UserID).
		// 	SetGoodID(order.GoodID).
		// 	SetAppGoodID(order.AppGoodID).
		// 	SetGoodType(defaultGoodType).
		// 	SetParentOrderID(order.ParentOrderID).
		// 	SetOrderType(order.OrderType).
		// 	SetCreateMethod(order.CreateMethod).
		// 	SetSimulate(order.Simulate).
		// 	SetCreatedAt(order.OrderCreatedAt).
		// 	SetUpdatedAt(order.OrderUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("2 -------- create order base ---------")
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("order.AppID: ", order.AppID)
		fmt.Println("order.UserID: ", order.UserID)
		fmt.Println("order.GoodID: ", order.GoodID)
		fmt.Println("order.AppGoodID: ", order.AppGoodID)
		fmt.Println("defaultGoodType: ", defaultGoodType)
		fmt.Println("order.ParentOrderID: ", order.ParentOrderID)
		fmt.Println("order.OrderType: ", order.OrderType)
		fmt.Println("order.CreateMethod: ", order.CreateMethod)
		fmt.Println("order.Simulate: ", order.Simulate)
		fmt.Println("order.OrderCreatedAt: ", order.OrderCreatedAt)
		fmt.Println("order.OrderUpdatedAt: ", order.OrderUpdatedAt)
		fmt.Println("22 ------- create order base ---------")

		id := uuid.New()
		unit, err := decimal.NewFromString(order.UnitsV1)
		if err != nil {
			return err
		}
		goodValueUsd, err := decimal.NewFromString(order.GoodValueUsd)
		if err != nil {
			return err
		}
		paymentAmount, err := decimal.NewFromString(order.PaymentAmount)
		if err != nil {
			return err
		}
		discountAmount, err := decimal.NewFromString(order.DiscountAmount)
		if err != nil {
			return err
		}
		// if _, err := tx.
		// 	PowerRental.
		// 	Create().
		// 	SetEntID(id).
		// 	SetOrderID(order.EntID).
		// 	SetAppGoodStockID(appGoodStockID).
		// 	SetUnits(unit).
		// 	SetGoodValueUsd(goodValueUsd).
		// 	SetPaymentAmountUsd(paymentAmount).
		// 	SetDiscountAmountUsd(discountAmount).
		// 	SetPromotionID(order.PromotionID).
		// 	SetInvestmentType(order.InvestmentType).
		// 	SetCreatedAt(order.OrderCreatedAt).
		// 	SetUpdatedAt(order.OrderUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("2 -------- create power rental ---------")
		fmt.Println("id: ", id)
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("appGoodStockID: ", appGoodStockID)
		fmt.Println("unit: ", unit)
		fmt.Println("goodValueUsd: ", goodValueUsd)
		fmt.Println("paymentAmount: ", paymentAmount)
		fmt.Println("discountAmount: ", discountAmount)
		fmt.Println("order.PromotionID: ", order.PromotionID)
		fmt.Println("order.InvestmentType: ", order.InvestmentType)
		fmt.Println("order.OrderCreatedAt: ", order.OrderCreatedAt)
		fmt.Println("order.OrderUpdatedAt: ", order.OrderUpdatedAt)
		fmt.Println("22 ------- create power rental ---------")

		id = uuid.New()
		// if _, err := tx.
		// 	OrderStateBase.
		// 	Create().
		// 	SetEntID(id).
		// 	SetOrderID(order.EntID).
		// 	SetOrderState(order.OrderState).
		// 	SetStartMode(order.StartMode).
		// 	SetStartAt(order.StartAt).
		// 	SetLastBenefitAt(order.LastBenefitAt).
		// 	SetBenefitState(order.BenefitState).
		// 	SetPaymentType(order.PaymentType).
		// 	SetCreatedAt(order.OrderStateCreatedAt).
		// 	SetUpdatedAt(order.OrderStateUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("2 -------- create order state base ---------")
		fmt.Println("id: ", id)
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("order.OrderState: ", order.OrderState)
		fmt.Println("order.StartMode: ", order.StartMode)
		fmt.Println("order.StartAt: ", order.StartAt)
		fmt.Println("order.LastBenefitAt: ", order.LastBenefitAt)
		fmt.Println("order.BenefitState: ", order.BenefitState)
		fmt.Println("order.PaymentType: ", order.PaymentType)
		fmt.Println("order.OrderStateCreatedAt: ", order.OrderStateCreatedAt)
		fmt.Println("order.OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
		fmt.Println("22 ------- create order state base ---------")

		id = uuid.New()
		canceledAt := uint32(0)
		if order.CancelState == ordertypes.OrderState_OrderStateCanceled.String() {
			canceledAt = order.OrderStateUpdatedAt
		}
		// if _, err := tx.
		// 	PowerRentalState.
		// 	Create().
		// 	SetEntID(id).
		// 	SetOrderID(order.EntID).
		// 	SetCancelState(order.CancelState).
		// 	SetCanceledAt(canceledAt).
		// 	SetDurationSeconds(order.Duration).
		// 	SetPaymentID(paymentID).
		// 	SetPaidAt(order.PaidAt).
		// 	SetUserSetPaid(order.UserSetPaid).
		// 	SetUserSetCanceled(order.UserSetCanceled).
		// 	SetAdminSetCanceled(order.AdminSetCanceled).
		// 	SetPaymentState(order.PaymentState).
		// 	SetOutofgasSeconds(uint32(0)).
		// 	SetCompensateSeconds(uint32(0)).
		// 	SetRenewState(order.RenewState).
		// 	SetRenewNotifyAt(order.RenewNotifyAt).
		// 	SetCreatedAt(order.OrderStateCreatedAt).
		// 	SetUpdatedAt(order.OrderStateUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("2 -------- create power rental state ---------")
		fmt.Println("id: ", id)
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("order.CancelState: ", order.CancelState)
		fmt.Println("canceledAt: ", canceledAt)
		fmt.Println("order.Duration: ", order.Duration)
		fmt.Println("paymentID: ", paymentID)
		fmt.Println("order.PaidAt: ", order.PaidAt)
		fmt.Println("order.UserSetPaid: ", order.UserSetPaid)
		fmt.Println("order.UserSetCanceled: ", order.UserSetCanceled)
		fmt.Println("order.AdminSetCanceled: ", order.AdminSetCanceled)
		fmt.Println("order.PaymentState: ", order.PaymentState)
		fmt.Println("OutofgasSeconds: ", uint32(0))
		fmt.Println("CompensateSeconds: ", uint32(0))
		fmt.Println("order.RenewState: ", order.RenewState)
		fmt.Println("order.RenewNotifyAt: ", order.RenewNotifyAt)
		fmt.Println("order.OrderStateCreatedAt: ", order.OrderStateCreatedAt)
		fmt.Println("order.OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
		fmt.Println("22 ------- create power rental state ---------")

		couponIDs := []string{}
		_ = json.Unmarshal([]byte(order.CouponIDs), &couponIDs)
		for _, couponIDStr := range couponIDs {
			couponID := uuid.MustParse(couponIDStr)
			id = uuid.New()
			// if _, err := tx.
			// 	OrderCoupon.
			// 	Create().
			// 	SetEntID(id).
			// 	SetOrderID(order.EntID).
			// 	SetCouponID(couponID).
			// 	SetCreatedAt(order.OrderStateCreatedAt).
			// 	SetUpdatedAt(order.OrderStateUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("2 -------- create order coupon ---------")
			fmt.Println("id: ", id)
			fmt.Println("order.EntID: ", order.EntID)
			fmt.Println("couponID: ", couponID)
			fmt.Println("order.OrderStateCreatedAt: ", order.OrderStateCreatedAt)
			fmt.Println("order.OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
			fmt.Println("22 ------- create order coupon ---------")
		}
	}

	return nil
}

//nolint:funlen,gocyclo
func migrateFees(ctx context.Context, tx *ent.Tx) error {
	fmt.Println("======================== exec migrateFees ==========================")
	selectOrderSql := fmt.Sprintf("select od.ent_id,od.app_id,od.user_id,od.good_id,od.app_good_id,od.parent_order_id,od.order_type,od.create_method,od.simulate,od.coupon_ids,od.payment_type,od.units_v1,od.good_value_usd,od.payment_amount,od.discount_amount,od.promotion_id,od.investment_type,od.duration,od.payment_id,od.payment_coin_type_id,od.balance_amount,od.coin_usd_currency,od.local_coin_usd_currency,od.live_coin_usd_currency,od.created_at as order_created_at,od.updated_at as order_updated_at,os.cancel_state,os.paid_at,os.user_set_paid,os.user_set_canceled,os.admin_set_canceled,os.payment_state,os.renew_state,os.renew_notify_at,os.order_state,os.start_mode,os.start_at,os.last_benefit_at,os.benefit_state,os.payment_finish_amount,os.created_at as order_state_created_at,os.updated_at as order_state_updated_at,pm.ent_id as payment_ent_id,pm.account_id,pm.start_amount,pm.created_at as payment_created_at,pm.updated_at as payment_updated_at,ol.ent_id as ledger_lock_id,ol.created_at as lock_created_at,ol.updated_at as lock_updated_at from orders as od inner join order_states os on od.ent_id=os.order_id and os.deleted_at=0 left join payments as pm on od.ent_id=pm.order_id and pm.deleted_at=0 left join order_locks as ol on od.ent_id=ol.order_id and ol.deleted_at=0 and ol.lock_type='LockBalance' where od.parent_order_id!='%v' and od.deleted_at=0", uuid.Nil.String()) //nolint
	fmt.Println("exec selectOrderSql: ", selectOrderSql)
	orderRows, err := tx.QueryContext(ctx, selectOrderSql)
	if err != nil {
		return err
	}

	type Order struct {
		EntID                uuid.UUID `json:"ent_id"`
		AppID                uuid.UUID `json:"app_id"`
		UserID               uuid.UUID `json:"user_id"`
		GoodID               uuid.UUID `json:"good_id"`
		AppGoodID            uuid.UUID `json:"app_good_id"`
		ParentOrderID        uuid.UUID `json:"parent_order_id"`
		OrderType            string
		CreateMethod         string
		Simulate             bool
		CouponIDsStr         sql.NullString
		CouponIDs            string
		PaymentType          string
		UnitsV1              string
		GoodValueUsd         string
		PaymentAmount        string
		DiscountAmount       string
		PromotionID          uuid.UUID `json:"promotion_id"`
		InvestmentType       string
		Duration             uint32
		PaymentID            uuid.UUID `json:"payment_id"`
		PaymentCoinTypeID    uuid.UUID `json:"payment_coin_type_id"`
		BalanceAmount        string
		CoinUsdCurrency      string
		LocalCoinUsdCurrency string
		LiveCoinUsdCurrency  string
		OrderCreatedAt       uint32
		OrderUpdatedAt       uint32
		CancelState          string
		PaidAt               uint32
		UserSetPaid          bool
		UserSetCanceled      bool
		AdminSetCanceled     bool
		PaymentState         string
		RenewState           string
		RenewNotifyAt        uint32
		OrderState           string
		StartMode            string
		StartAt              uint32
		LastBenefitAt        uint32
		BenefitState         string
		PaymentFinishAmount  string
		OrderStateCreatedAt  uint32
		OrderStateUpdatedAt  uint32
		PaymentEntID         uuid.UUID `json:"payment_ent_id"`
		AccountID            uuid.UUID `json:"account_id"`
		StartAmountStr       sql.NullString
		StartAmount          string
		PaymentCreatedAtInt  sql.NullInt32
		PaymentUpdatedAtInt  sql.NullInt32
		PaymentCreatedAt     uint32
		PaymentUpdatedAt     uint32
		LedgerLockID         uuid.UUID `json:"ledger_lock_id"`
		LockCreatedAtInt     sql.NullInt32
		LockUpdatedAtInt     sql.NullInt32
		LockCreatedAt        uint32
		LockUpdatedAt        uint32
	}
	orders := map[uuid.UUID]*Order{}
	for orderRows.Next() {
		od := &Order{}
		if err := orderRows.Scan(&od.EntID, &od.AppID, &od.UserID, &od.GoodID, &od.AppGoodID,
			&od.ParentOrderID, &od.OrderType, &od.CreateMethod, &od.Simulate, &od.CouponIDsStr,
			&od.PaymentType, &od.UnitsV1, &od.GoodValueUsd, &od.PaymentAmount, &od.DiscountAmount,
			&od.PromotionID, &od.InvestmentType, &od.Duration, &od.PaymentID, &od.PaymentCoinTypeID,
			&od.BalanceAmount, &od.CoinUsdCurrency, &od.LocalCoinUsdCurrency, &od.LiveCoinUsdCurrency,
			&od.OrderCreatedAt, &od.OrderUpdatedAt, &od.CancelState, &od.PaidAt, &od.UserSetPaid,
			&od.UserSetCanceled, &od.AdminSetCanceled, &od.PaymentState, &od.RenewState, &od.RenewNotifyAt,
			&od.OrderState, &od.StartMode, &od.StartAt, &od.LastBenefitAt, &od.BenefitState, &od.PaymentFinishAmount,
			&od.OrderStateCreatedAt, &od.OrderStateUpdatedAt, &od.PaymentEntID, &od.AccountID, &od.StartAmountStr,
			&od.PaymentCreatedAtInt, &od.PaymentUpdatedAtInt, &od.LedgerLockID, &od.LockCreatedAtInt, &od.LockUpdatedAtInt,
		); err != nil {
			return err
		}
		od.StartAmount = decimal.NewFromInt(0).String()
		if od.StartAmountStr.Valid {
			od.StartAmount = od.StartAmountStr.String
		}
		od.CouponIDs = "[]"
		if od.CouponIDsStr.Valid && od.CouponIDsStr.String != "null" {
			od.CouponIDs = od.CouponIDsStr.String
		}
		od.PaymentCreatedAt = uint32(0)
		if od.PaymentCreatedAtInt.Valid {
			od.PaymentCreatedAt = uint32(od.PaymentCreatedAtInt.Int32)
		}
		od.PaymentUpdatedAt = uint32(0)
		if od.PaymentUpdatedAtInt.Valid {
			od.PaymentUpdatedAt = uint32(od.PaymentUpdatedAtInt.Int32)
		}
		od.LockCreatedAt = uint32(0)
		if od.LockCreatedAtInt.Valid {
			od.LockCreatedAt = uint32(od.LockCreatedAtInt.Int32)
		}
		od.LockUpdatedAt = uint32(0)
		if od.LockUpdatedAtInt.Valid {
			od.LockUpdatedAt = uint32(od.LockUpdatedAtInt.Int32)
		}
		orders[od.EntID] = od
	}
	fmt.Println("3 len orders=========== ", len(orders))

	defaultGoodType := "LegacyPowerRental"
	defaultPaymentObseleteState := "PaymentObseleteNone"
	for _, order := range orders {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>><<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
		fmt.Println("============== order ================")
		fmt.Println("EntID: ", order.EntID)
		fmt.Println("AppID: ", order.AppID)
		fmt.Println("UserID: ", order.UserID)
		fmt.Println("GoodID: ", order.GoodID)
		fmt.Println("AppGoodID: ", order.AppGoodID)
		fmt.Println("ParentOrderID: ", order.ParentOrderID)
		fmt.Println("OrderType: ", order.OrderType)
		fmt.Println("CreateMethod: ", order.CreateMethod)
		fmt.Println("Simulate: ", order.Simulate)
		fmt.Println("CouponIDsStr: ", order.CouponIDsStr)
		fmt.Println("CouponIDs: ", order.CouponIDs)
		fmt.Println("PaymentType: ", order.PaymentType)
		fmt.Println("UnitsV1: ", order.UnitsV1)
		fmt.Println("GoodValueUsd: ", order.GoodValueUsd)
		fmt.Println("PaymentAmount: ", order.PaymentAmount)
		fmt.Println("DiscountAmount: ", order.DiscountAmount)
		fmt.Println("PromotionID: ", order.PromotionID)
		fmt.Println("InvestmentType: ", order.InvestmentType)
		fmt.Println("Duration: ", order.Duration)
		fmt.Println("PaymentID: ", order.PaymentID)
		fmt.Println("PaymentCoinTypeID: ", order.PaymentCoinTypeID)
		fmt.Println("BalanceAmount: ", order.BalanceAmount)
		fmt.Println("CoinUsdCurrency: ", order.CoinUsdCurrency)
		fmt.Println("LocalCoinUsdCurrency: ", order.LocalCoinUsdCurrency)
		fmt.Println("LiveCoinUsdCurrency: ", order.LiveCoinUsdCurrency)
		fmt.Println("OrderCreatedAt: ", order.OrderCreatedAt)
		fmt.Println("OrderUpdatedAt: ", order.OrderUpdatedAt)
		fmt.Println("CancelState: ", order.CancelState)
		fmt.Println("PaidAt: ", order.PaidAt)
		fmt.Println("UserSetPaid: ", order.UserSetPaid)
		fmt.Println("UserSetCanceled: ", order.UserSetCanceled)
		fmt.Println("AdminSetCanceled: ", order.AdminSetCanceled)
		fmt.Println("PaymentState: ", order.PaymentState)
		fmt.Println("RenewState: ", order.RenewState)
		fmt.Println("RenewNotifyAt: ", order.RenewNotifyAt)
		fmt.Println("OrderState: ", order.OrderState)
		fmt.Println("StartMode: ", order.StartMode)
		fmt.Println("StartAt: ", order.StartAt)
		fmt.Println("LastBenefitAt: ", order.LastBenefitAt)
		fmt.Println("BenefitState: ", order.BenefitState)
		fmt.Println("PaymentFinishAmount: ", order.PaymentFinishAmount)
		fmt.Println("OrderStateCreatedAt: ", order.OrderStateCreatedAt)
		fmt.Println("OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
		fmt.Println("PaymentEntID: ", order.PaymentEntID)
		fmt.Println("AccountID: ", order.AccountID)
		fmt.Println("StartAmountStr: ", order.StartAmountStr)
		fmt.Println("StartAmount: ", order.StartAmount)
		fmt.Println("PaymentCreatedAtInt: ", order.PaymentCreatedAtInt)
		fmt.Println("PaymentCreatedAt: ", order.PaymentCreatedAt)
		fmt.Println("PaymentUpdatedAtInt: ", order.PaymentUpdatedAtInt)
		fmt.Println("PaymentUpdatedAt: ", order.PaymentUpdatedAt)
		fmt.Println("LedgerLockID: ", order.LedgerLockID)
		fmt.Println("LockCreatedAtInt: ", order.LockCreatedAtInt)
		fmt.Println("LockCreatedAt: ", order.LockCreatedAt)
		fmt.Println("LockUpdatedAtInt: ", order.LockUpdatedAtInt)
		fmt.Println("LockUpdatedAt: ", order.LockUpdatedAt)
		fmt.Println("============== order ================")
		paymentID := order.PaymentID
		if order.PaymentID != uuid.Nil {
			// payment transter
			// if _, err := tx.
			// 	PaymentBase.
			// 	Create().
			// 	SetEntID(order.PaymentID).
			// 	SetOrderID(order.EntID).
			// 	SetObseleteState(defaultPaymentObseleteState).
			// 	SetCreatedAt(order.PaymentCreatedAt).
			// 	SetUpdatedAt(order.PaymentUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("3 -------- create payment base ---------")
			fmt.Println("order.PaymentID: ", order.PaymentID)
			fmt.Println("order.EntID: ", order.EntID)
			fmt.Println("defaultPaymentObseleteState: ", defaultPaymentObseleteState)
			fmt.Println("order.PaymentCreatedAt: ", order.PaymentCreatedAt)
			fmt.Println("order.PaymentUpdatedAt: ", order.PaymentUpdatedAt)
			fmt.Println("33 ------- create payment base ---------")

			paymentAmount, err := decimal.NewFromString(order.PaymentAmount)
			if err != nil {
				return fmt.Errorf("invalid paymentAmount")
			}
			startAmount, err := decimal.NewFromString(order.StartAmount)
			if err != nil {
				return fmt.Errorf("invalid startAmount")
			}
			finishAmount, err := decimal.NewFromString(order.PaymentFinishAmount)
			if err != nil {
				return fmt.Errorf("invalid finishAmount")
			}
			coinUsdCurrency, err := decimal.NewFromString(order.CoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid coinUsdCurrency")
			}
			localCoinUsdCurrency, err := decimal.NewFromString(order.LocalCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid localCoinUsdCurrency")
			}
			liveCoinUsdCurrency, err := decimal.NewFromString(order.LiveCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid liveCoinUsdCurrency")
			}
			id := uuid.New()
			// if _, err := tx.
			// 	PaymentTransfer.
			// 	Create().
			// 	SetEntID(id).
			// 	SetPaymentID(order.PaymentID).
			// 	SetCoinTypeID(order.PaymentCoinTypeID).
			// 	SetAccountID(order.AccountID).
			// 	SetAmount(paymentAmount).
			// 	SetStartAmount(startAmount).
			// 	SetFinishAmount(finishAmount).
			// 	SetCoinUsdCurrency(coinUsdCurrency).
			// 	SetLocalCoinUsdCurrency(localCoinUsdCurrency).
			// 	SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
			// 	SetCreatedAt(order.PaymentCreatedAt).
			// 	SetUpdatedAt(order.PaymentUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("3 -------- create payment transter ---------")
			fmt.Println("id: ", id)
			fmt.Println("order.PaymentID: ", order.PaymentID)
			fmt.Println("order.PaymentCoinTypeID: ", order.PaymentCoinTypeID)
			fmt.Println("order.AccountID: ", order.AccountID)
			fmt.Println("paymentAmount: ", paymentAmount)
			fmt.Println("startAmount: ", startAmount)
			fmt.Println("finishAmount: ", finishAmount)
			fmt.Println("coinUsdCurrency: ", coinUsdCurrency)
			fmt.Println("localCoinUsdCurrency: ", localCoinUsdCurrency)
			fmt.Println("liveCoinUsdCurrency: ", liveCoinUsdCurrency)
			fmt.Println("order.PaymentCreatedAt: ", order.PaymentCreatedAt)
			fmt.Println("order.PaymentUpdatedAt: ", order.PaymentUpdatedAt)
			fmt.Println("33 ------- create payment transter ---------")
		}

		orderBalanceAmount, err := decimal.NewFromString(order.BalanceAmount)
		if err != nil {
			return fmt.Errorf("invalid balanceamount")
		}

		if orderBalanceAmount.Cmp(decimal.NewFromInt(0)) > 0 {
			// payment balance
			if order.PaymentID == uuid.Nil {
				paymentID = uuid.New()
				fmt.Println("new paymentID: ", paymentID)
				// if _, err := tx.
				// 	PaymentBase.
				// 	Create().
				// 	SetEntID(paymentID).
				// 	SetOrderID(order.EntID).
				// 	SetObseleteState(defaultPaymentObseleteState).
				// 	SetCreatedAt(order.PaymentCreatedAt).
				// 	SetUpdatedAt(order.PaymentUpdatedAt).
				// 	Save(ctx); err != nil {
				// 	return err
				// }
				fmt.Println("3 -------- create payment base ---------")
				fmt.Println("paymentID: ", paymentID)
				fmt.Println("order.EntID: ", order.EntID)
				fmt.Println("defaultPaymentObseleteState: ", defaultPaymentObseleteState)
				fmt.Println("order.PaymentCreatedAt: ", order.PaymentCreatedAt)
				fmt.Println("order.PaymentUpdatedAt: ", order.PaymentUpdatedAt)
				fmt.Println("33 ------- create payment base ---------")
			}

			coinUsdCurrency, err := decimal.NewFromString(order.CoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid coinusdcurrency")
			}
			localCoinUsdCurrency, err := decimal.NewFromString(order.LocalCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid localcoinusdcurrency")
			}
			liveCoinUsdCurrency, err := decimal.NewFromString(order.LiveCoinUsdCurrency)
			if err != nil {
				return fmt.Errorf("invalid livecoinusdcurrency")
			}
			id := uuid.New()
			// if _, err := tx.
			// 	PaymentBalance.
			// 	Create().
			// 	SetEntID(id).
			// 	SetPaymentID(paymentID).
			// 	SetCoinTypeID(order.PaymentCoinTypeID).
			// 	SetAmount(orderBalanceAmount).
			// 	SetCoinUsdCurrency(coinUsdCurrency).
			// 	SetLocalCoinUsdCurrency(localCoinUsdCurrency).
			// 	SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
			// 	SetCreatedAt(order.OrderCreatedAt).
			// 	SetUpdatedAt(order.OrderUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("3 -------- create payment balance ---------")
			fmt.Println("id: ", id)
			fmt.Println("order.PaymentID: ", paymentID)
			fmt.Println("order.PaymentCoinTypeID: ", order.PaymentCoinTypeID)
			fmt.Println("orderBalanceAmount: ", orderBalanceAmount)
			fmt.Println("coinUsdCurrency: ", coinUsdCurrency)
			fmt.Println("localCoinUsdCurrency: ", localCoinUsdCurrency)
			fmt.Println("liveCoinUsdCurrency: ", liveCoinUsdCurrency)
			fmt.Println("order.OrderCreatedAt: ", order.OrderCreatedAt)
			fmt.Println("order.OrderUpdatedAt: ", order.OrderUpdatedAt)
			fmt.Println("33 ------- create payment balance ---------")

			if order.LedgerLockID != uuid.Nil {
				id = uuid.New()
				// if _, err := tx.
				// 	PaymentBalanceLock.
				// 	Create().
				// 	SetEntID(id).
				// 	SetPaymentID(paymentID).
				// 	SetLedgerLockID(order.LedgerLockID).
				// 	SetCreatedAt(order.LockCreatedAt).
				// 	SetUpdatedAt(order.LockUpdatedAt).
				// 	Save(ctx); err != nil {
				// 	return err
				// }
				fmt.Println("3 -------- create payment balance lock ---------")
				fmt.Println("id: ", id)
				fmt.Println("order.PaymentID: ", paymentID)
				fmt.Println("order.LedgerLockID: ", order.LedgerLockID)
				fmt.Println("order.LockCreatedAt: ", order.LockCreatedAt)
				fmt.Println("order.LockUpdatedAt: ", order.LockUpdatedAt)
				fmt.Println("33 ------- create payment balance lock ---------")
			}
		}

		// if _, err := tx.
		// 	OrderBase.
		// 	Create().
		// 	SetEntID(order.EntID).
		// 	SetAppID(order.AppID).
		// 	SetUserID(order.UserID).
		// 	SetGoodID(order.GoodID).
		// 	SetAppGoodID(order.AppGoodID).
		// 	SetGoodType(defaultGoodType).
		// 	SetParentOrderID(order.ParentOrderID).
		// 	SetOrderType(order.OrderType).
		// 	SetCreateMethod(order.CreateMethod).
		// 	SetSimulate(order.Simulate).
		// 	SetCreatedAt(order.OrderCreatedAt).
		// 	SetUpdatedAt(order.OrderUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("3 -------- create order base ---------")
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("order.AppID: ", order.AppID)
		fmt.Println("order.UserID: ", order.UserID)
		fmt.Println("order.GoodID: ", order.GoodID)
		fmt.Println("order.AppGoodID: ", order.AppGoodID)
		fmt.Println("defaultGoodType: ", defaultGoodType)
		fmt.Println("order.ParentOrderID: ", order.ParentOrderID)
		fmt.Println("order.OrderType: ", order.OrderType)
		fmt.Println("order.CreateMethod: ", order.CreateMethod)
		fmt.Println("order.Simulate: ", order.Simulate)
		fmt.Println("order.OrderCreatedAt: ", order.OrderCreatedAt)
		fmt.Println("order.OrderUpdatedAt: ", order.OrderUpdatedAt)
		fmt.Println("33 ------- create order base ---------")

		id := uuid.New()
		goodValueUsd, err := decimal.NewFromString(order.GoodValueUsd)
		if err != nil {
			return err
		}
		paymentAmount, err := decimal.NewFromString(order.PaymentAmount)
		if err != nil {
			return err
		}
		discountAmount, err := decimal.NewFromString(order.DiscountAmount)
		if err != nil {
			return err
		}
		// if _, err := tx.
		// 	FeeOrder.
		// 	Create().
		// 	SetEntID(id).
		// 	SetOrderID(order.EntID).
		// 	SetGoodValueUsd(goodValueUsd).
		// 	SetPaymentAmountUsd(paymentAmount).
		// 	SetDiscountAmountUsd(discountAmount).
		// 	SetPromotionID(order.PromotionID).
		// 	SetDurationSeconds(order.Duration).
		// 	SetCreatedAt(order.OrderCreatedAt).
		// 	SetUpdatedAt(order.OrderUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("3 -------- create fee order ---------")
		fmt.Println("id: ", id)
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("goodValueUsd: ", goodValueUsd)
		fmt.Println("paymentAmount: ", paymentAmount)
		fmt.Println("discountAmount: ", discountAmount)
		fmt.Println("order.PromotionID: ", order.PromotionID)
		fmt.Println("order.Duration: ", order.Duration)
		fmt.Println("order.OrderCreatedAt: ", order.OrderCreatedAt)
		fmt.Println("order.OrderUpdatedAt: ", order.OrderUpdatedAt)
		fmt.Println("33 ------- create fee order ---------")

		id = uuid.New()
		// if _, err := tx.
		// 	OrderStateBase.
		// 	Create().
		// 	SetEntID(id).
		// 	SetOrderID(order.EntID).
		// 	SetOrderState(order.OrderState).
		// 	SetStartMode(order.StartMode).
		// 	SetStartAt(order.StartAt).
		// 	SetLastBenefitAt(order.LastBenefitAt).
		// 	SetBenefitState(order.BenefitState).
		// 	SetPaymentType(order.PaymentType).
		// 	SetCreatedAt(order.OrderStateCreatedAt).
		// 	SetUpdatedAt(order.OrderStateUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("3 -------- create order state base ---------")
		fmt.Println("id: ", id)
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("order.OrderState: ", order.OrderState)
		fmt.Println("order.StartMode: ", order.StartMode)
		fmt.Println("order.StartAt: ", order.StartAt)
		fmt.Println("order.LastBenefitAt: ", order.LastBenefitAt)
		fmt.Println("order.BenefitState: ", order.BenefitState)
		fmt.Println("order.PaymentType: ", order.PaymentType)
		fmt.Println("order.OrderStateCreatedAt: ", order.OrderStateCreatedAt)
		fmt.Println("order.OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
		fmt.Println("33 ------- create order state base ---------")

		id = uuid.New()
		canceledAt := uint32(0)
		if order.CancelState == ordertypes.OrderState_OrderStateCanceled.String() {
			canceledAt = order.OrderStateUpdatedAt
		}
		// if _, err := tx.
		// 	FeeOrderState.
		// 	Create().
		// 	SetEntID(id).
		// 	SetOrderID(order.EntID).
		// 	SetPaymentID(paymentID).
		// 	SetPaidAt(order.PaidAt).
		// 	SetUserSetPaid(order.UserSetPaid).
		// 	SetUserSetCanceled(order.UserSetCanceled).
		// 	SetAdminSetCanceled(order.AdminSetCanceled).
		// 	SetPaymentState(order.PaymentState).
		// 	SetCancelState(order.CancelState).
		// 	SetCanceledAt(canceledAt).
		// 	SetCreatedAt(order.OrderStateCreatedAt).
		// 	SetUpdatedAt(order.OrderStateUpdatedAt).
		// 	Save(ctx); err != nil {
		// 	return err
		// }
		fmt.Println("3 -------- create fee order state ---------")
		fmt.Println("id: ", id)
		fmt.Println("order.EntID: ", order.EntID)
		fmt.Println("paymentID: ", paymentID)
		fmt.Println("order.PaidAt: ", order.PaidAt)
		fmt.Println("order.UserSetPaid: ", order.UserSetPaid)
		fmt.Println("order.UserSetCanceled: ", order.UserSetCanceled)
		fmt.Println("order.AdminSetCanceled: ", order.AdminSetCanceled)
		fmt.Println("order.PaymentState: ", order.PaymentState)
		fmt.Println("order.CancelState: ", order.CancelState)
		fmt.Println("canceledAt: ", canceledAt)
		fmt.Println("order.OrderStateCreatedAt: ", order.OrderStateCreatedAt)
		fmt.Println("order.OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
		fmt.Println("33 ------- create fee order state ---------")

		couponIDs := []string{}
		_ = json.Unmarshal([]byte(order.CouponIDs), &couponIDs)
		for _, couponIDStr := range couponIDs {
			couponID := uuid.MustParse(couponIDStr)
			id = uuid.New()
			// if _, err := tx.
			// 	OrderCoupon.
			// 	Create().
			// 	SetEntID(id).
			// 	SetOrderID(order.EntID).
			// 	SetCouponID(couponID).
			// 	SetCreatedAt(order.OrderStateCreatedAt).
			// 	SetUpdatedAt(order.OrderStateUpdatedAt).
			// 	Save(ctx); err != nil {
			// 	return err
			// }
			fmt.Println("3 -------- create order coupon ---------")
			fmt.Println("id: ", id)
			fmt.Println("order.EntID: ", order.EntID)
			fmt.Println("couponID: ", couponID)
			fmt.Println("order.OrderStateCreatedAt: ", order.OrderStateCreatedAt)
			fmt.Println("order.OrderStateUpdatedAt: ", order.OrderStateUpdatedAt)
			fmt.Println("33 ------- create order coupon ---------")
		}
	}
	return nil
}

func Migrate(ctx context.Context) error {
	var err error

	if err := db.Init(); err != nil {
		return err
	}

	err = redis2.TryLock(lockKey(), 0)
	if err != nil {
		return err
	}
	logger.Sugar().Infow("Migrate order", "Start", "...")
	defer func() {
		_ = redis2.Unlock(lockKey())
		logger.Sugar().Infow("Migrate order", "Done", "...", "error", err)
	}()

	return db.WithTx(ctx, func(_ctx context.Context, tx *ent.Tx) error {
		if err := migrateAppConfigs(ctx, tx); err != nil {
			return err
		}
		if err := migrateOrderLocks(ctx, tx); err != nil {
			return err
		}
		if err := migratePowerRentals(ctx, tx); err != nil {
			return err
		}
		if err := migrateFees(ctx, tx); err != nil {
			return err
		}
		logger.Sugar().Infow("Migrate", "Done", "success")
		return nil
	})
}
