//nolint:dupl
package migrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	servicename "github.com/NpoolPlatform/order-gateway/pkg/servicename"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
	"github.com/NpoolPlatform/order-middleware/pkg/db/ent"
	entappconfig "github.com/NpoolPlatform/order-middleware/pkg/db/ent/appconfig"
	entfeeorder "github.com/NpoolPlatform/order-middleware/pkg/db/ent/feeorder"
	entfeeorderstate "github.com/NpoolPlatform/order-middleware/pkg/db/ent/feeorderstate"
	entordercoupon "github.com/NpoolPlatform/order-middleware/pkg/db/ent/ordercoupon"
	entorderstatebase "github.com/NpoolPlatform/order-middleware/pkg/db/ent/orderstatebase"
	entpaymentbalance "github.com/NpoolPlatform/order-middleware/pkg/db/ent/paymentbalance"
	entpaymentbalancelock "github.com/NpoolPlatform/order-middleware/pkg/db/ent/paymentbalancelock"
	entpaymentbase "github.com/NpoolPlatform/order-middleware/pkg/db/ent/paymentbase"
	entpaymenttransfer "github.com/NpoolPlatform/order-middleware/pkg/db/ent/paymenttransfer"
	entpowerrental "github.com/NpoolPlatform/order-middleware/pkg/db/ent/powerrental"
	entpowerrentalstate "github.com/NpoolPlatform/order-middleware/pkg/db/ent/powerrentalstate"
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

func validFieldExist(ctx context.Context, tx *ent.Tx, databaseName, tableName, fieldName string) (bool, error) {
	checkFieldSQL := fmt.Sprintf("show columns from %v.%v like '%v'", databaseName, tableName, fieldName)
	logger.Sugar().Warnw(
		"validFieldExist",
		"checkFieldSQL",
		checkFieldSQL,
	)
	checkRows, err := tx.QueryContext(ctx, checkFieldSQL)
	if err != nil {
		return false, err
	}
	count := 0
	for checkRows.Next() {
		count++
	}
	if count == 0 {
		return false, nil
	}
	return true, nil
}

func migrateAppConfigs(ctx context.Context, tx *ent.Tx) error {
	logger.Sugar().Warnw("exec migrateAppConfigs")
	rows, err := tx.QueryContext(ctx, "select ent_id from appuser_manager.apps where deleted_at = 0") //nolint
	if err != nil {
		return err
	}

	type App struct {
		EntID uuid.UUID `json:"ent_id"`
	}
	apps := []*App{}
	for rows.Next() {
		app := &App{}
		if err := rows.Scan(&app.EntID); err != nil {
			return err
		}
		apps = append(apps, app)
	}
	for _, app := range apps {
		appConfig, err := tx.
			AppConfig.
			Query().
			Where(
				entappconfig.AppID(app.EntID),
				entappconfig.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if appConfig != nil {
			logger.Sugar().Warnw(
				"appid exist",
				"appID", app.EntID,
			)
			continue
		}

		sendCouponProbability := decimal.NewFromInt32(0)
		cashableProfitProbability := decimal.NewFromInt32(0)
		now := uint32(time.Now().Unix())

		if _, err := tx.
			AppConfig.
			Create().
			SetAppID(app.EntID).
			SetSimulateOrderCouponProbability(sendCouponProbability).
			SetSimulateOrderCashableProfitProbability(cashableProfitProbability).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func migrateOrderLocks(ctx context.Context, tx *ent.Tx) error {
	logger.Sugar().Warnw("exec migrateOrderLocks")
	exist, err := validFieldExist(ctx, tx, "order_manager", "order_locks", "app_id")
	if err != nil {
		return err
	}
	if !exist {
		logger.Sugar().Warnw("unnecessary to exec migrateOrderLocks")
		return nil
	}
	orderLocksSQL := "alter table order_locks modify app_id varchar(36);"
	logger.Sugar().Warnw(
		"exec orderLocksSQL",
		"sql", orderLocksSQL,
	)
	rc, err := tx.ExecContext(ctx, orderLocksSQL)
	if err != nil {
		return err
	}
	_, err = rc.RowsAffected()
	if err != nil {
		return fmt.Errorf("fail modify order_locks: %v", err)
	}
	return nil
}

//nolint:funlen,gocyclo
func migratePowerRentals(ctx context.Context, tx *ent.Tx) error {
	logger.Sugar().Warnw("exec migratePowerRentals")
	selectOrderSql := fmt.Sprintf("select od.ent_id,od.app_id,od.user_id,od.good_id,od.app_good_id,od.parent_order_id,od.order_type,od.create_method,od.simulate,od.coupon_ids,od.payment_type,od.units_v1,od.good_value_usd,od.payment_amount,od.transfer_amount,od.discount_amount,od.promotion_id,od.investment_type,od.duration,od.payment_id,od.payment_coin_type_id,od.balance_amount,od.coin_usd_currency,od.local_coin_usd_currency,od.live_coin_usd_currency,od.created_at as order_created_at,od.updated_at as order_updated_at,os.cancel_state,os.paid_at,os.user_set_paid,os.user_set_canceled,os.admin_set_canceled,os.payment_state,os.renew_state,os.renew_notify_at,os.order_state,os.start_mode,os.start_at,os.last_benefit_at,os.benefit_state,os.payment_finish_amount,os.created_at as order_state_created_at,os.updated_at as order_state_updated_at,pm.ent_id as payment_ent_id,pm.account_id,pm.start_amount,pm.created_at as payment_created_at,pm.updated_at as payment_updated_at,ol.ent_id as ledger_lock_id,ol.created_at as lock_created_at,ol.updated_at as lock_updated_at from orders as od inner join order_states os on od.ent_id=os.order_id and os.deleted_at=0 left join payments as pm on od.ent_id=pm.order_id and pm.deleted_at=0 left join order_locks as ol on od.ent_id=ol.order_id and ol.deleted_at=0 and ol.lock_type='LockBalance' where od.parent_order_id='%v' and od.deleted_at=0", uuid.Nil.String()) //nolint
	logger.Sugar().Warnw(
		"exec selectOrderSql",
		"sql", selectOrderSql,
	)
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
		TransferAmount       string
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
			&od.PaymentType, &od.UnitsV1, &od.GoodValueUsd, &od.PaymentAmount, &od.TransferAmount, &od.DiscountAmount,
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
	logger.Sugar().Warnw(
		"len power rental orders",
		"len orders", len(orders),
	)
	logger.Sugar().Warnw(
		"len appGoodIDs",
		"len appGoodIDs", len(appGoodIDMap),
	)

	appGoodIDs := ""
	comm := ""
	for _, id := range appGoodIDMap {
		appGoodIDs += fmt.Sprintf("%v'%v'", comm, id)
		if comm == "" {
			comm = ","
		}
	}

	if appGoodIDs == "" {
		logger.Sugar().Warnw("unnecessary to exec migratePowerRentals")
		return nil
	}

	selectAppGoodStockSQL := fmt.Sprintf("select ent_id, app_good_id from good_manager.app_stocks where app_good_id in(%v) and deleted_at=0", appGoodIDs)
	logger.Sugar().Warnw(
		"exec selectAppGoodStockSQL",
		"sql", selectAppGoodStockSQL,
	)
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
		logger.Sugar().Warnw("exec order")
		appGoodStockID, ok := appStockMap[order.AppGoodID]
		if !ok {
			logger.Sugar().Warnw(
				"not found app good stock id",
				"appGoodID",
				order.AppGoodID,
			)
			continue
		}
		paymentID := order.PaymentID
		if order.PaymentID != uuid.Nil {
			checkPaymentBaseSQL := fmt.Sprintf("select 1 from payment_bases where ent_id='%v'", order.PaymentID)
			checkRows, err := tx.QueryContext(ctx, checkPaymentBaseSQL)
			if err != nil {
				return err
			}
			count := 0
			for checkRows.Next() {
				count++
			}
			if count == 0 {
				logger.Sugar().Warnw(
					"paymentbase not exist",
					"paymentID", order.PaymentID,
				)
				if order.PaymentCreatedAt == 0 {
					order.PaymentCreatedAt = order.OrderCreatedAt
					order.PaymentUpdatedAt = order.OrderUpdatedAt
				}
				// payment transter
				if _, err := tx.
					PaymentBase.
					Create().
					SetEntID(order.PaymentID).
					SetOrderID(order.EntID).
					SetObseleteState(defaultPaymentObseleteState).
					SetCreatedAt(order.PaymentCreatedAt).
					SetUpdatedAt(order.PaymentUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}

			paymentTransfer, err := tx.
				PaymentTransfer.
				Query().
				Where(
					entpaymenttransfer.PaymentID(order.PaymentID),
					entpaymenttransfer.DeletedAt(0),
				).
				Only(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}

			if paymentTransfer == nil {
				logger.Sugar().Warnw(
					"payment transfer not exist",
					"paymentID", order.PaymentID,
				)
				transferAmount, err := decimal.NewFromString(order.TransferAmount)
				if err != nil {
					return fmt.Errorf("invalid transferAmount")
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
				if _, err := tx.
					PaymentTransfer.
					Create().
					SetEntID(id).
					SetPaymentID(order.PaymentID).
					SetCoinTypeID(order.PaymentCoinTypeID).
					SetAccountID(order.AccountID).
					SetAmount(transferAmount).
					SetStartAmount(startAmount).
					SetFinishAmount(finishAmount).
					SetCoinUsdCurrency(coinUsdCurrency).
					SetLocalCoinUsdCurrency(localCoinUsdCurrency).
					SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
					SetCreatedAt(order.PaymentCreatedAt).
					SetUpdatedAt(order.PaymentUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}
		}

		orderBalanceAmount, err := decimal.NewFromString(order.BalanceAmount)
		if err != nil {
			return fmt.Errorf("invalid balanceamount")
		}

		if orderBalanceAmount.Cmp(decimal.NewFromInt(0)) > 0 {
			// payment balance
			if order.PaymentID == uuid.Nil {
				paymentBase, err := tx.
					PaymentBase.
					Query().
					Where(
						entpaymentbase.OrderID(order.EntID),
						entpaymentbase.DeletedAt(0),
					).
					Only(ctx)
				if err != nil && !ent.IsNotFound(err) {
					return err
				}
				if paymentBase == nil {
					logger.Sugar().Warnw(
						"paymentbase not exist",
						"paymentID", order.PaymentID,
					)
					paymentID = uuid.New()
					logger.Sugar().Warnw(
						"new paymentID",
						"paymentID", paymentID,
					)
					if order.PaymentCreatedAt == 0 {
						order.PaymentCreatedAt = order.OrderCreatedAt
						order.PaymentUpdatedAt = order.OrderUpdatedAt
					}
					if _, err := tx.
						PaymentBase.
						Create().
						SetEntID(paymentID).
						SetOrderID(order.EntID).
						SetObseleteState(defaultPaymentObseleteState).
						SetCreatedAt(order.PaymentCreatedAt).
						SetUpdatedAt(order.PaymentUpdatedAt).
						Save(ctx); err != nil {
						return err
					}
				} else {
					paymentID = paymentBase.EntID
				}
			}

			paymentBalance, err := tx.
				PaymentBalance.
				Query().
				Where(
					entpaymentbalance.PaymentID(paymentID),
					entpaymentbalance.DeletedAt(0),
				).
				Only(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}
			if paymentBalance == nil {
				logger.Sugar().Warnw(
					"payment balance not exist",
					"paymentID", paymentID,
				)
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
				if _, err := tx.
					PaymentBalance.
					Create().
					SetEntID(id).
					SetPaymentID(paymentID).
					SetCoinTypeID(order.PaymentCoinTypeID).
					SetAmount(orderBalanceAmount).
					SetCoinUsdCurrency(coinUsdCurrency).
					SetLocalCoinUsdCurrency(localCoinUsdCurrency).
					SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
					SetCreatedAt(order.OrderCreatedAt).
					SetUpdatedAt(order.OrderUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}

			if order.LedgerLockID != uuid.Nil {
				paymentBalanceLock, err := tx.
					PaymentBalanceLock.
					Query().
					Where(
						entpaymentbalancelock.PaymentID(paymentID),
						entpaymentbalancelock.LedgerLockID(order.LedgerLockID),
						entpaymentbalancelock.DeletedAt(0),
					).
					Only(ctx)
				if err != nil && !ent.IsNotFound(err) {
					return err
				}
				if paymentBalanceLock == nil {
					logger.Sugar().Warnw(
						"payment balance lock not exist",
						"paymentID", paymentID,
						"ledgerLockID", order.LedgerLockID,
					)
					id := uuid.New()
					if _, err := tx.
						PaymentBalanceLock.
						Create().
						SetEntID(id).
						SetPaymentID(paymentID).
						SetLedgerLockID(order.LedgerLockID).
						SetCreatedAt(order.LockCreatedAt).
						SetUpdatedAt(order.LockUpdatedAt).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}

		checkOrderBaseSQL := fmt.Sprintf("select 1 from order_bases where ent_id='%v'", order.EntID)
		checkRows, err := tx.QueryContext(ctx, checkOrderBaseSQL)
		if err != nil {
			return err
		}
		count := 0
		for checkRows.Next() {
			count++
		}
		if count == 0 {
			logger.Sugar().Warnw(
				"order base not exist",
				"orderID", order.EntID,
			)

			if order.OrderType == ordertypes.OrderType_Airdrop.String() || order.OrderType == ordertypes.OrderType_Offline.String() {
				order.CreateMethod = ordertypes.OrderCreateMethod_OrderCreatedByAdmin.String()
			}

			if _, err := tx.
				OrderBase.
				Create().
				SetEntID(order.EntID).
				SetAppID(order.AppID).
				SetUserID(order.UserID).
				SetGoodID(order.GoodID).
				SetAppGoodID(order.AppGoodID).
				SetGoodType(defaultGoodType).
				SetParentOrderID(order.ParentOrderID).
				SetOrderType(order.OrderType).
				SetCreateMethod(order.CreateMethod).
				SetSimulate(order.Simulate).
				SetCreatedAt(order.OrderCreatedAt).
				SetUpdatedAt(order.OrderUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		powerRental, err := tx.
			PowerRental.
			Query().
			Where(
				entpowerrental.OrderID(order.EntID),
				entpowerrental.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if powerRental == nil {
			logger.Sugar().Warnw(
				"power rental not exist",
				"orderID", order.EntID,
			)
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
			durationSeconds := order.Duration * timedef.SecondsPerDay
			if _, err := tx.
				PowerRental.
				Create().
				SetEntID(id).
				SetOrderID(order.EntID).
				SetAppGoodStockID(appGoodStockID).
				SetUnits(unit).
				SetDurationSeconds(durationSeconds).
				SetGoodValueUsd(goodValueUsd).
				SetPaymentAmountUsd(paymentAmount).
				SetDiscountAmountUsd(discountAmount).
				SetPromotionID(order.PromotionID).
				SetInvestmentType(order.InvestmentType).
				SetCreatedAt(order.OrderCreatedAt).
				SetUpdatedAt(order.OrderUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		orderStateBase, err := tx.
			OrderStateBase.
			Query().
			Where(
				entorderstatebase.OrderID(order.EntID),
				entorderstatebase.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}

		if orderStateBase == nil {
			logger.Sugar().Warnw(
				"order state base not exist",
				"orderID", order.EntID,
			)
			id := uuid.New()
			if _, err := tx.
				OrderStateBase.
				Create().
				SetEntID(id).
				SetOrderID(order.EntID).
				SetOrderState(order.OrderState).
				SetStartMode(order.StartMode).
				SetStartAt(order.StartAt).
				SetLastBenefitAt(order.LastBenefitAt).
				SetBenefitState(order.BenefitState).
				SetPaymentType(order.PaymentType).
				SetCreatedAt(order.OrderStateCreatedAt).
				SetUpdatedAt(order.OrderStateUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		powerRentalState, err := tx.
			PowerRentalState.
			Query().
			Where(
				entpowerrentalstate.OrderID(order.EntID),
				entpowerrentalstate.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if powerRentalState == nil {
			logger.Sugar().Warnw(
				"power rental state not exist",
				"orderID", order.EntID,
			)
			id := uuid.New()
			canceledAt := uint32(0)
			if order.CancelState == ordertypes.OrderState_OrderStateCanceled.String() {
				canceledAt = order.OrderStateUpdatedAt
			}
			if _, err := tx.
				PowerRentalState.
				Create().
				SetEntID(id).
				SetOrderID(order.EntID).
				SetCancelState(order.CancelState).
				SetCanceledAt(canceledAt).
				SetPaymentID(paymentID).
				SetPaidAt(order.PaidAt).
				SetUserSetPaid(order.UserSetPaid).
				SetUserSetCanceled(order.UserSetCanceled).
				SetAdminSetCanceled(order.AdminSetCanceled).
				SetPaymentState(order.PaymentState).
				SetOutofgasSeconds(uint32(0)).
				SetCompensateSeconds(uint32(0)).
				SetRenewState(order.RenewState).
				SetRenewNotifyAt(order.RenewNotifyAt).
				SetCreatedAt(order.OrderStateCreatedAt).
				SetUpdatedAt(order.OrderStateUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		couponIDs := []string{}
		_ = json.Unmarshal([]byte(order.CouponIDs), &couponIDs)
		for _, couponIDStr := range couponIDs {
			couponID := uuid.MustParse(couponIDStr)
			orderCoupon, err := tx.
				OrderCoupon.
				Query().
				Where(
					entordercoupon.OrderID(order.EntID),
					entordercoupon.CouponID(couponID),
					entordercoupon.DeletedAt(0),
				).
				Only(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}
			if orderCoupon == nil {
				logger.Sugar().Warnw(
					"order coupon not exist",
					"orderID", order.EntID,
					"couponID", couponID,
				)
				id := uuid.New()
				if _, err := tx.
					OrderCoupon.
					Create().
					SetEntID(id).
					SetOrderID(order.EntID).
					SetCouponID(couponID).
					SetCreatedAt(order.OrderCreatedAt).
					SetUpdatedAt(order.OrderUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

//nolint:funlen,gocyclo
func migrateFees(ctx context.Context, tx *ent.Tx) error {
	logger.Sugar().Warnw("exec migrateFees")
	selectOrderSql := fmt.Sprintf("select od.ent_id,od.app_id,od.user_id,od.good_id,od.app_good_id,od.parent_order_id,od.order_type,od.create_method,od.simulate,od.coupon_ids,od.payment_type,od.units_v1,od.good_value_usd,od.payment_amount,od.transfer_amount,od.discount_amount,od.promotion_id,od.investment_type,od.duration,od.payment_id,od.payment_coin_type_id,od.balance_amount,od.coin_usd_currency,od.local_coin_usd_currency,od.live_coin_usd_currency,od.created_at as order_created_at,od.updated_at as order_updated_at,os.cancel_state,os.paid_at,os.user_set_paid,os.user_set_canceled,os.admin_set_canceled,os.payment_state,os.renew_state,os.renew_notify_at,os.order_state,os.start_mode,os.start_at,os.last_benefit_at,os.benefit_state,os.payment_finish_amount,os.created_at as order_state_created_at,os.updated_at as order_state_updated_at,pm.ent_id as payment_ent_id,pm.account_id,pm.start_amount,pm.created_at as payment_created_at,pm.updated_at as payment_updated_at,ol.ent_id as ledger_lock_id,ol.created_at as lock_created_at,ol.updated_at as lock_updated_at from orders as od inner join order_states os on od.ent_id=os.order_id and os.deleted_at=0 left join payments as pm on od.ent_id=pm.order_id and pm.deleted_at=0 left join order_locks as ol on od.ent_id=ol.order_id and ol.deleted_at=0 and ol.lock_type='LockBalance' where od.parent_order_id!='%v' and od.deleted_at=0", uuid.Nil.String()) //nolint
	logger.Sugar().Warnw(
		"exec selectOrderSql",
		"sql", selectOrderSql,
	)
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
		TransferAmount       string
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
			&od.PaymentType, &od.UnitsV1, &od.GoodValueUsd, &od.PaymentAmount, &od.TransferAmount, &od.DiscountAmount,
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
	logger.Sugar().Warnw(
		"len fee orders",
		"len orders", len(orders),
	)

	defaultGoodType := "LegacyPowerRental"
	defaultPaymentObseleteState := "PaymentObseleteNone"
	for _, order := range orders {
		logger.Sugar().Warnw("exec order")
		paymentID := order.PaymentID
		if order.PaymentID != uuid.Nil {
			checkPaymentBaseSQL := fmt.Sprintf("select 1 from payment_bases where ent_id='%v'", order.PaymentID)
			checkRows, err := tx.QueryContext(ctx, checkPaymentBaseSQL)
			if err != nil {
				return err
			}
			count := 0
			for checkRows.Next() {
				count++
			}
			if count == 0 {
				logger.Sugar().Warnw(
					"paymentbase not exist",
					"paymentID", order.PaymentID,
				)
				if order.PaymentCreatedAt == 0 {
					order.PaymentCreatedAt = order.OrderCreatedAt
					order.PaymentUpdatedAt = order.OrderUpdatedAt
				}
				// payment transter
				if _, err := tx.
					PaymentBase.
					Create().
					SetEntID(order.PaymentID).
					SetOrderID(order.EntID).
					SetObseleteState(defaultPaymentObseleteState).
					SetCreatedAt(order.PaymentCreatedAt).
					SetUpdatedAt(order.PaymentUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}

			paymentTransfer, err := tx.
				PaymentTransfer.
				Query().
				Where(
					entpaymenttransfer.PaymentID(order.PaymentID),
					entpaymenttransfer.DeletedAt(0),
				).
				Only(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}

			if paymentTransfer == nil {
				logger.Sugar().Warnw(
					"payment transfer not exist",
					"paymentID:", order.PaymentID,
				)
				transferAmount, err := decimal.NewFromString(order.TransferAmount)
				if err != nil {
					return fmt.Errorf("invalid transferAmount")
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
				if _, err := tx.
					PaymentTransfer.
					Create().
					SetEntID(id).
					SetPaymentID(order.PaymentID).
					SetCoinTypeID(order.PaymentCoinTypeID).
					SetAccountID(order.AccountID).
					SetAmount(transferAmount).
					SetStartAmount(startAmount).
					SetFinishAmount(finishAmount).
					SetCoinUsdCurrency(coinUsdCurrency).
					SetLocalCoinUsdCurrency(localCoinUsdCurrency).
					SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
					SetCreatedAt(order.PaymentCreatedAt).
					SetUpdatedAt(order.PaymentUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}
		}

		orderBalanceAmount, err := decimal.NewFromString(order.BalanceAmount)
		if err != nil {
			return fmt.Errorf("invalid balanceamount")
		}

		if orderBalanceAmount.Cmp(decimal.NewFromInt(0)) > 0 {
			// payment balance
			if order.PaymentID == uuid.Nil {
				paymentBase, err := tx.
					PaymentBase.
					Query().
					Where(
						entpaymentbase.OrderID(order.EntID),
						entpaymentbase.DeletedAt(0),
					).
					Only(ctx)
				if err != nil && !ent.IsNotFound(err) {
					return err
				}
				if paymentBase == nil {
					logger.Sugar().Warnw(
						"paymentbase not exist",
						"paymentID", order.PaymentID,
					)
					paymentID = uuid.New()
					logger.Sugar().Warnw(
						"new paymentID",
						"paymentID", paymentID,
					)
					if order.PaymentCreatedAt == 0 {
						order.PaymentCreatedAt = order.OrderCreatedAt
						order.PaymentUpdatedAt = order.OrderUpdatedAt
					}
					if _, err := tx.
						PaymentBase.
						Create().
						SetEntID(paymentID).
						SetOrderID(order.EntID).
						SetObseleteState(defaultPaymentObseleteState).
						SetCreatedAt(order.PaymentCreatedAt).
						SetUpdatedAt(order.PaymentUpdatedAt).
						Save(ctx); err != nil {
						return err
					}
				} else {
					paymentID = paymentBase.EntID
				}
			}

			paymentBalance, err := tx.
				PaymentBalance.
				Query().
				Where(
					entpaymentbalance.PaymentID(paymentID),
					entpaymentbalance.DeletedAt(0),
				).
				Only(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}
			if paymentBalance == nil {
				logger.Sugar().Warnw(
					"payment balance not exist",
					"paymentID", paymentID,
				)
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
				if _, err := tx.
					PaymentBalance.
					Create().
					SetEntID(id).
					SetPaymentID(paymentID).
					SetCoinTypeID(order.PaymentCoinTypeID).
					SetAmount(orderBalanceAmount).
					SetCoinUsdCurrency(coinUsdCurrency).
					SetLocalCoinUsdCurrency(localCoinUsdCurrency).
					SetLiveCoinUsdCurrency(liveCoinUsdCurrency).
					SetCreatedAt(order.OrderCreatedAt).
					SetUpdatedAt(order.OrderUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}

			if order.LedgerLockID != uuid.Nil {
				paymentBalanceLock, err := tx.
					PaymentBalanceLock.
					Query().
					Where(
						entpaymentbalancelock.PaymentID(paymentID),
						entpaymentbalancelock.LedgerLockID(order.LedgerLockID),
						entpaymentbalancelock.DeletedAt(0),
					).
					Only(ctx)
				if err != nil && !ent.IsNotFound(err) {
					return err
				}
				if paymentBalanceLock == nil {
					logger.Sugar().Warnw(
						"payment balance lock not exist",
						"paymentID", paymentID,
						"ledgerLockID", order.LedgerLockID,
					)
					id := uuid.New()
					if _, err := tx.
						PaymentBalanceLock.
						Create().
						SetEntID(id).
						SetPaymentID(paymentID).
						SetLedgerLockID(order.LedgerLockID).
						SetCreatedAt(order.LockCreatedAt).
						SetUpdatedAt(order.LockUpdatedAt).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}

		checkOrderBaseSQL := fmt.Sprintf("select 1 from order_bases where ent_id='%v'", order.EntID)
		checkRows, err := tx.QueryContext(ctx, checkOrderBaseSQL)
		if err != nil {
			return err
		}
		count := 0
		for checkRows.Next() {
			count++
		}
		if count == 0 {
			logger.Sugar().Warnw(
				"order base not exist",
				"orderID", order.EntID,
			)

			if order.OrderType == ordertypes.OrderType_Airdrop.String() || order.OrderType == ordertypes.OrderType_Offline.String() {
				order.CreateMethod = ordertypes.OrderCreateMethod_OrderCreatedByAdmin.String()
			}

			if _, err := tx.
				OrderBase.
				Create().
				SetEntID(order.EntID).
				SetAppID(order.AppID).
				SetUserID(order.UserID).
				SetGoodID(order.GoodID).
				SetAppGoodID(order.AppGoodID).
				SetGoodType(defaultGoodType).
				SetParentOrderID(order.ParentOrderID).
				SetOrderType(order.OrderType).
				SetCreateMethod(order.CreateMethod).
				SetSimulate(order.Simulate).
				SetCreatedAt(order.OrderCreatedAt).
				SetUpdatedAt(order.OrderUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		feeOrder, err := tx.
			FeeOrder.
			Query().
			Where(
				entfeeorder.OrderID(order.EntID),
				entfeeorder.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if feeOrder == nil {
			logger.Sugar().Warnw(
				"fee order not exist",
				"orderID", order.EntID,
			)
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
			durationSeconds := order.Duration * timedef.SecondsPerDay
			if _, err := tx.
				FeeOrder.
				Create().
				SetEntID(id).
				SetOrderID(order.EntID).
				SetGoodValueUsd(goodValueUsd).
				SetPaymentAmountUsd(paymentAmount).
				SetDiscountAmountUsd(discountAmount).
				SetPromotionID(order.PromotionID).
				SetDurationSeconds(durationSeconds).
				SetCreatedAt(order.OrderCreatedAt).
				SetUpdatedAt(order.OrderUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		orderStateBase, err := tx.
			OrderStateBase.
			Query().
			Where(
				entorderstatebase.OrderID(order.EntID),
				entorderstatebase.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}

		if orderStateBase == nil {
			logger.Sugar().Warnw(
				"order state base not exist",
				"orderID", order.EntID,
			)
			id := uuid.New()
			if _, err := tx.
				OrderStateBase.
				Create().
				SetEntID(id).
				SetOrderID(order.EntID).
				SetOrderState(order.OrderState).
				SetStartMode(order.StartMode).
				SetStartAt(order.StartAt).
				SetLastBenefitAt(order.LastBenefitAt).
				SetBenefitState(order.BenefitState).
				SetPaymentType(order.PaymentType).
				SetCreatedAt(order.OrderStateCreatedAt).
				SetUpdatedAt(order.OrderStateUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		feeOrderState, err := tx.
			FeeOrderState.
			Query().
			Where(
				entfeeorderstate.OrderID(order.EntID),
				entfeeorderstate.DeletedAt(0),
			).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}

		if feeOrderState == nil {
			logger.Sugar().Warnw(
				"fee order state not exist",
				"orderID", order.EntID,
			)
			id := uuid.New()
			canceledAt := uint32(0)
			if order.CancelState == ordertypes.OrderState_OrderStateCanceled.String() {
				canceledAt = order.OrderStateUpdatedAt
			}
			if _, err := tx.
				FeeOrderState.
				Create().
				SetEntID(id).
				SetOrderID(order.EntID).
				SetPaymentID(paymentID).
				SetPaidAt(order.PaidAt).
				SetUserSetPaid(order.UserSetPaid).
				SetUserSetCanceled(order.UserSetCanceled).
				SetAdminSetCanceled(order.AdminSetCanceled).
				SetPaymentState(order.PaymentState).
				SetCancelState(order.CancelState).
				SetCanceledAt(canceledAt).
				SetCreatedAt(order.OrderStateCreatedAt).
				SetUpdatedAt(order.OrderStateUpdatedAt).
				Save(ctx); err != nil {
				return err
			}
		}

		couponIDs := []string{}
		_ = json.Unmarshal([]byte(order.CouponIDs), &couponIDs)
		for _, couponIDStr := range couponIDs {
			couponID := uuid.MustParse(couponIDStr)
			orderCoupon, err := tx.
				OrderCoupon.
				Query().
				Where(
					entordercoupon.OrderID(order.EntID),
					entordercoupon.CouponID(couponID),
					entordercoupon.DeletedAt(0),
				).
				Only(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return err
			}
			if orderCoupon == nil {
				logger.Sugar().Warnw(
					"order coupon not exist",
					"orderID", order.EntID,
					"couponID", couponID,
				)
				id := uuid.New()
				if _, err := tx.
					OrderCoupon.
					Create().
					SetEntID(id).
					SetOrderID(order.EntID).
					SetCouponID(couponID).
					SetCreatedAt(order.OrderCreatedAt).
					SetUpdatedAt(order.OrderUpdatedAt).
					Save(ctx); err != nil {
					return err
				}
			}
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
