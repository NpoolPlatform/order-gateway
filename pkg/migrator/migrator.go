package migrator

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	servicename "github.com/NpoolPlatform/order-gateway/pkg/servicename"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
	"github.com/NpoolPlatform/order-middleware/pkg/db/ent"
	entorder "github.com/NpoolPlatform/order-middleware/pkg/db/ent/order"
	entorderstate "github.com/NpoolPlatform/order-middleware/pkg/db/ent/orderstate"
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

var orderStateMap = map[string]*types.OrderState{
	"DefaultState":   types.OrderState_DefaultOrderState.Enum(),
	"WaitPayment":    types.OrderState_OrderStateWaitPayment.Enum(),
	"Paid":           types.OrderState_OrderStatePaid.Enum(),
	"PaymentTimeout": types.OrderState_OrderStatePaymentTimeout.Enum(),
	"Canceled":       types.OrderState_OrderStateCanceled.Enum(),
	"InService":      types.OrderState_OrderStateInService.Enum(),
	"Expired":        types.OrderState_OrderStateExpired.Enum(),
}

var paymentStateMap = map[string]*types.PaymentState{
	"DefaultState": types.PaymentState_DefaultPaymentState.Enum(),
	"Wait":         types.PaymentState_PaymentStateWait.Enum(),
	"Done":         types.PaymentState_PaymentStateDone.Enum(),
	"Canceled":     types.PaymentState_PaymentStateCanceled.Enum(),
	"TimeOut":      types.PaymentState_PaymentStateTimeout.Enum(),
}

func resolveOrderState(oldState string) string {
	newState := orderStateMap[oldState]
	if newState == nil {
		return oldState
	}
	return newState.String()
}

func resolvePaymentState(oldState string) string {
	newState := paymentStateMap[oldState]
	if newState == nil {
		return oldState
	}
	return newState.String()
}

//nolint:funlen,gocyclo
func migrateOrder(ctx context.Context, tx *ent.Tx) error {
	var err error
	r, err := tx.QueryContext(ctx, "select id,app_id,user_id,good_id,units_v1,start_at,end_at,type,state,last_benefit_at,deleted_at from orders where app_good_id=''")
	if err != nil {
		return err
	}
	type od struct {
		ID            uuid.UUID
		AppID         uuid.UUID
		UserID        uuid.UUID
		GoodID        uuid.UUID
		UnitsV1       decimal.Decimal
		StartAt       uint32
		EndAt         uint32
		Type          string
		State         string
		LastBenefitAt uint32
		DeletedAt     uint32
	}
	orders := []*od{}
	for r.Next() {
		order := &od{}
		if err := r.Scan(&order.ID, &order.AppID, &order.UserID, &order.GoodID, &order.UnitsV1, &order.StartAt, &order.EndAt,
			&order.Type, &order.State, &order.LastBenefitAt, &order.DeletedAt); err != nil {
			return err
		}
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		return nil
	}

	r, err = tx.QueryContext(ctx, "select id,app_id,good_id,price,deleted_at from good_manager.app_goods")
	if err != nil {
		return err
	}
	type ag struct {
		ID        uuid.UUID
		AppID     uuid.UUID
		GoodID    uuid.UUID
		Price     decimal.Decimal
		DeletedAt uint32
	}
	appgoods := map[uuid.UUID]*ag{}
	for r.Next() {
		good := &ag{}
		if err := r.Scan(&good.ID, &good.AppID, &good.GoodID, &good.Price, &good.DeletedAt); err != nil {
			return err
		}
		appgoods[good.GoodID] = good
	}

	r, err = tx.QueryContext(ctx, "select id,coin_type_id,duration_days,deleted_at from good_manager.goods")
	if err != nil {
		return err
	}
	type g struct {
		ID           uuid.UUID
		CoinTypeID   uuid.UUID
		DurationDays uint32
		DeletedAt    uint32
	}
	goods := map[uuid.UUID]*g{}
	for r.Next() {
		good := &g{}
		if err := r.Scan(&good.ID, &good.CoinTypeID, &good.DurationDays, &good.DeletedAt); err != nil {
			return err
		}
		goods[good.ID] = good
	}

	selectOrderText := "select id,app_id,user_id,order_id,amount,pay_with_balance_amount,finish_amount," +
		"coin_usd_currency,local_coin_usd_currency,live_coin_usd_currency,coin_info_id,state,chain_transaction_id," +
		"user_set_paid,user_set_canceled,updated_at,deleted_at from payments"
	r, err = tx.QueryContext(ctx, selectOrderText)
	if err != nil {
		return err
	}

	type p struct {
		ID                   uuid.UUID
		AppID                uuid.UUID
		UserID               uuid.UUID
		OrderID              uuid.UUID
		Amount               decimal.Decimal
		PayWithBalanceAmount decimal.Decimal
		FinishAmount         decimal.Decimal
		CoinUsdCurrency      decimal.Decimal
		LocalCoinUsdCurrency decimal.Decimal
		LiveCoinUsdCurrency  decimal.Decimal
		CoinInfoID           uuid.UUID
		State                string
		ChainTransactionID   string
		UserSetPaid          bool
		UserSetCanceled      bool
		UpdatedAt            uint32
		DeletedAt            uint32
	}
	payments := map[uuid.UUID]*p{}
	for r.Next() {
		payment := &p{}
		if err := r.Scan(&payment.ID, &payment.AppID, &payment.UserID, &payment.OrderID, &payment.Amount, &payment.PayWithBalanceAmount, &payment.FinishAmount,
			&payment.CoinUsdCurrency, &payment.LocalCoinUsdCurrency, &payment.LiveCoinUsdCurrency, &payment.CoinInfoID, &payment.State, &payment.ChainTransactionID,
			&payment.UserSetPaid, &payment.UserSetCanceled, &payment.UpdatedAt, &payment.DeletedAt); err != nil {
			return err
		}
		payments[payment.OrderID] = payment
	}
	for _, order := range orders {
		appGood, ok := appgoods[order.GoodID]
		if !ok {
			continue
		}
		payment, ok := payments[order.ID]
		if !ok {
			continue
		}
		good, ok := goods[order.GoodID]
		if !ok {
			continue
		}
		// discount_amount
		discountAmount := decimal.NewFromInt(0)
		// good_valud_usd
		goodValueUSD := appGood.Price.Mul(order.UnitsV1)
		// good_value
		goodValue := goodValueUSD.Div(payment.CoinUsdCurrency)
		// payment_amount
		paymentAmount := payment.Amount.Add(payment.PayWithBalanceAmount)

		_, err = tx.Order.Update().
			Where(
				entorder.ID(order.ID),
			).
			SetAppGoodID(appGood.ID).
			SetPaymentID(payment.ID).
			SetGoodValueUsd(goodValueUSD).
			SetGoodValue(goodValue).
			SetPaymentAmount(paymentAmount).
			SetDurationDays(good.DurationDays).
			SetOrderType(order.Type).
			SetCoinTypeID(good.CoinTypeID).
			SetPaymentCoinTypeID(payment.CoinInfoID).
			SetTransferAmount(payment.Amount).
			SetBalanceAmount(payment.PayWithBalanceAmount).
			SetCoinUsdCurrency(payment.CoinUsdCurrency).
			SetLocalCoinUsdCurrency(payment.LocalCoinUsdCurrency).
			SetLiveCoinUsdCurrency(payment.LiveCoinUsdCurrency).
			SetDiscountAmount(discountAmount).
			Save(ctx)
		if err != nil {
			return err
		}

		// check order state
		exist, err := tx.OrderState.
			Query().
			Where(
				entorderstate.OrderID(order.ID),
			).
			ForUpdate().
			Exist(ctx)
		if err != nil {
			return err
		}
		if exist {
			continue
		}
		// create orderstates
		// order_state
		_orderState := resolveOrderState(order.State)
		// payment_state
		paymentState := resolvePaymentState(payment.State)
		// paid_at
		paidAt := uint32(0)
		if paymentState == types.PaymentState_PaymentStateDone.String() {
			paidAt = payment.UpdatedAt
		}

		if _, err := tx.
			OrderState.
			Create().
			SetOrderID(order.ID).
			SetOrderState(_orderState).
			SetStartAt(order.StartAt).
			SetEndAt(order.EndAt).
			SetLastBenefitAt(order.LastBenefitAt).
			SetUserSetPaid(payment.UserSetPaid).
			SetUserSetCanceled(payment.UserSetCanceled).
			SetPaymentTransactionID(payment.ChainTransactionID).
			SetPaymentFinishAmount(payment.FinishAmount).
			SetPaymentState(paymentState).
			SetPaidAt(paidAt).
			Save(ctx); err != nil {
			return err
		}
	}

	return nil
}

//nolint:funlen
func Migrate(ctx context.Context) error {
	var err error

	if err := db.Init(); err != nil {
		return err
	}
	logger.Sugar().Infow("Migrate order", "Start", "...")
	defer func() {
		_ = redis2.Unlock(lockKey())
		logger.Sugar().Infow("Migrate order", "Done", "...", "error", err)
	}()

	err = redis2.TryLock(lockKey(), 0)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, func(_ctx context.Context, tx *ent.Tx) error {
		_, err := tx.
			ExecContext(
				ctx,
				"update payments set amount='0' where amount is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update payments set pay_with_balance_amount='0' where pay_with_balance_amount is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update payments set finish_amount='0' where finish_amount is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update payments set coin_usd_currency='0' where coin_usd_currency is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update payments set local_coin_usd_currency='0' where local_coin_usd_currency is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update payments set live_coin_usd_currency='0' where live_coin_usd_currency is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update orders set good_value_usd='0' where good_value_usd is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update orders set transfer_amount='0' where transfer_amount is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update orders set balance_amount='0' where balance_amount is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update orders set coin_usd_currency='0' where coin_usd_currency is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update orders set live_coin_usd_currency='0' where live_coin_usd_currency is NULL",
			)
		if err != nil {
			return err
		}
		_, err = tx.
			ExecContext(
				ctx,
				"update orders set local_coin_usd_currency='0' where local_coin_usd_currency is NULL",
			)
		if err != nil {
			return err
		}

		if err := migrateOrder(_ctx, tx); err != nil {
			logger.Sugar().Errorw("Migrate", "error", err)
			return err
		}
		logger.Sugar().Infow("Migrate", "Done", "success")

		return nil
	})
}
