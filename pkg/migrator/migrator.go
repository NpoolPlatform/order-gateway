package migrator

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	constant1 "github.com/NpoolPlatform/order-gateway/pkg/message/const"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
	"github.com/NpoolPlatform/order-middleware/pkg/db/ent"
	entorder "github.com/NpoolPlatform/order-middleware/pkg/db/ent/order"
	entpayment "github.com/NpoolPlatform/order-middleware/pkg/db/ent/payment"
)

const (
	keyServiceID = "serviceid"
)

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(constant1.ServiceName, keyServiceID)
	return fmt.Sprintf("migrator:%v", serviceID)
}

//nolint:funlen
func migrateState(ctx context.Context) error {
	return db.WithTx(ctx, func(_ctx context.Context, tx *ent.Tx) error {
		var err error
		OrderStateWaitPayment := "WaitPayment"
		OrderStatePaid := "Paid"
		OrderStatePaymentTimeout := "PaymentTimeout"
		OrderStateCanceled := "Canceled"
		OrderStateInService := "InService"
		OrderStateExpired := "Expired"
		_, err = tx.Order.Update().
			Where(
				entorder.StateV1(ordertypes.OrderState_DefaultOrderState.String()),
				entorder.State(OrderStateWaitPayment),
			).
			SetStateV1(ordertypes.OrderState_OrderStateWaitPayment.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Order.Update().
			Where(
				entorder.StateV1(ordertypes.OrderState_DefaultOrderState.String()),
				entorder.State(OrderStatePaid),
			).
			SetStateV1(ordertypes.OrderState_OrderStatePaid.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Order.Update().
			Where(
				entorder.StateV1(ordertypes.OrderState_DefaultOrderState.String()),
				entorder.State(OrderStatePaymentTimeout),
			).
			SetStateV1(ordertypes.OrderState_OrderStatePaymentTimeout.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Order.Update().
			Where(
				entorder.StateV1(ordertypes.OrderState_DefaultOrderState.String()),
				entorder.State(OrderStateCanceled),
			).
			SetStateV1(ordertypes.OrderState_OrderStateCanceled.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Order.Update().
			Where(
				entorder.StateV1(ordertypes.OrderState_DefaultOrderState.String()),
				entorder.State(OrderStateInService),
			).
			SetStateV1(ordertypes.OrderState_OrderStateInService.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Order.Update().
			Where(
				entorder.StateV1(ordertypes.OrderState_DefaultOrderState.String()),
				entorder.State(OrderStateExpired),
			).
			SetStateV1(ordertypes.OrderState_OrderStateExpired.String()).
			Save(_ctx)
		if err != nil {
			return err
		}

		PaymentStateWait := "Wait"
		PaymentStateDone := "Done"
		PaymentStateCanceled := "Canceled"
		PaymentStateTimeOut := "TimeOut"
		_, err = tx.Payment.Update().
			Where(
				entpayment.StateV1(ordertypes.PaymentState_DefaultPaymentState.String()),
				entpayment.State(PaymentStateWait),
			).
			SetStateV1(ordertypes.PaymentState_PaymentStateWait.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Payment.Update().
			Where(
				entpayment.StateV1(ordertypes.PaymentState_DefaultPaymentState.String()),
				entpayment.State(PaymentStateDone),
			).
			SetStateV1(ordertypes.PaymentState_PaymentStateDone.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Payment.Update().
			Where(
				entpayment.StateV1(ordertypes.PaymentState_DefaultPaymentState.String()),
				entpayment.State(PaymentStateCanceled),
			).
			SetStateV1(ordertypes.PaymentState_PaymentStateCanceled.String()).
			Save(_ctx)
		if err != nil {
			return err
		}
		_, err = tx.Payment.Update().
			Where(
				entpayment.StateV1(ordertypes.PaymentState_DefaultPaymentState.String()),
				entpayment.State(PaymentStateTimeOut),
			).
			SetStateV1(ordertypes.PaymentState_PaymentStateTimeOut.String()).
			Save(_ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func Migrate(ctx context.Context) error {
	if err := redis2.TryLock(lockKey(), 0); err != nil {
		return err
	}
	defer func() {
		_ = redis2.Unlock(lockKey())
	}()

	logger.Sugar().Infow("Migrate", "Start", "...")

	if err := db.Init(); err != nil {
		logger.Sugar().Errorw("Migrate", "error", err)
		return err
	}

	if err := migrateState(ctx); err != nil {
		logger.Sugar().Errorw("Migrate", "error", err)
		return err
	}

	logger.Sugar().Infow("Migrate", "Done", "success")

	return nil
}
