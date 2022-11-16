//nolint:nolintlint
package migrator

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"

	entsql "entgo.io/ent/dialect/sql"

	stant "github.com/NpoolPlatform/cloud-hashing-order/pkg/const"
	cconst "github.com/NpoolPlatform/cloud-hashing-order/pkg/message/const"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	paymentmgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/payment"

	corderent "github.com/NpoolPlatform/cloud-hashing-order/pkg/db/ent"
	peymentent "github.com/NpoolPlatform/cloud-hashing-order/pkg/db/ent/payment"

	"time"

	constant "github.com/NpoolPlatform/go-service-framework/pkg/mysql/const"
	"github.com/NpoolPlatform/order-manager/pkg/db"
	"github.com/NpoolPlatform/order-manager/pkg/db/ent"
)

func Migrate(ctx context.Context) error {
	return migrationCloudGoods(ctx)
}

const (
	keyUsername  = "username"
	keyPassword  = "password"
	keyDBName    = "database_name"
	maxOpen      = 10
	maxIdle      = 10
	MaxLife      = 3
	priceScale12 = 1000000000000
)

func dsn(hostname string) (string, error) {
	username := config.GetStringValueWithNameSpace(constant.MysqlServiceName, keyUsername)
	password := config.GetStringValueWithNameSpace(constant.MysqlServiceName, keyPassword)
	dbname := config.GetStringValueWithNameSpace(hostname, keyDBName)

	svc, err := config.PeekService(constant.MysqlServiceName)
	if err != nil {
		logger.Sugar().Warnw("dsb", "error", err)
		return "", err
	}

	return fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true&interpolateParams=true",
		username, password,
		svc.Address,
		svc.Port,
		dbname,
	), nil
}

func open(hostname string) (conn *sql.DB, err error) {
	hdsn, err := dsn(hostname)
	if err != nil {
		return nil, err
	}

	conn, err = sql.Open("mysql", hdsn)
	if err != nil {
		return nil, err
	}

	// https://github.com/go-sql-driver/mysql
	// See "Important settings" section.

	conn.SetConnMaxLifetime(time.Minute * MaxLife)
	conn.SetMaxOpenConns(maxOpen)
	conn.SetMaxIdleConns(maxIdle)

	return conn, nil
}

//nolint
func migrationCloudGoods(ctx context.Context) (err error) {
	cli, err := db.Client()
	if err != nil {
		return err
	}

	order, err := cli.Order.Query().Limit(1).All(ctx)
	if err != nil {
		return err
	}
	if len(order) != 0 {
		return nil
	}

	cloudOrder, err := open(cconst.ServiceName)
	if err != nil {
		return err
	}

	defer cloudOrder.Close()

	cloudOrderCli := corderent.NewClient(corderent.Driver(entsql.OpenDB(dialect.MySQL, cloudOrder)))

	logger.Sugar().Infow("Migrate order", "Start", "...")

	defer func() {
		logger.Sugar().Infow("Migrate order", "Done", "...", "error", err)
	}()

	amount := func(samount uint64) decimal.Decimal {
		if samount == 0 {
			return decimal.NewFromInt(0)
		}
		return decimal.NewFromInt(int64(samount)).Div(decimal.NewFromInt(priceScale12))
	}

	// Order
	paymentInfos, err := cloudOrderCli.
		Payment.
		Query().
		All(ctx)
	if err != nil {
		return err
	}

	paymentMap := map[uuid.UUID]*corderent.Payment{}
	for _, payment := range paymentInfos {
		paymentMap[payment.OrderID] = payment
	}

	orderInfos, err := cloudOrderCli.
		Order.
		Query().
		All(ctx)
	if err != nil {
		return err
	}

	err = db.WithTx(ctx, func(_ctx context.Context, tx *ent.Tx) error {
		bulk := make([]*ent.OrderCreate, len(orderInfos))
		for i, info := range orderInfos {
			userSetCanceled := false
			payState := peymentent.StateWait
			if _, ok := paymentMap[info.ID]; ok {
				userSetCanceled = paymentMap[info.ID].UserSetCanceled
				payState = paymentMap[info.ID].State
			}
			bulk[i] = tx.Order.
				Create().
				SetID(info.ID).
				SetCreatedAt(info.CreateAt).
				SetUpdatedAt(info.UpdateAt).
				SetDeletedAt(info.DeleteAt).
				SetGoodID(info.GoodID).
				SetAppID(info.AppID).
				SetUserID(info.UserID).
				SetParentOrderID(info.ParentOrderID).
				SetPayWithParent(info.PayWithParent).
				SetUnits(info.Units).
				SetPromotionID(info.PromotionID).
				SetDiscountCouponID(info.DiscountCouponID).
				SetUserSpecialReductionID(info.UserSpecialReductionID).
				SetStartAt(info.Start).
				SetEndAt(info.End).
				SetFixAmountCouponID(info.CouponID).
				SetType(getOrderType(info.OrderType)).
				SetState(getOrderState(payState.String(), userSetCanceled, info.Start, info.End))
		}
		_, err = tx.Order.CreateBulk(bulk...).Save(_ctx)
		return err
	})

	if err != nil {
		return err
	}

	// Payment
	err = db.WithTx(ctx, func(_ctx context.Context, tx *ent.Tx) error {
		bulk := make([]*ent.PaymentCreate, len(paymentInfos))
		for i, info := range paymentInfos {
			payWithBalance := decimal.NewFromInt(0)
			if info.PayWithBalanceAmount != nil {
				payWithBalance = *info.PayWithBalanceAmount
			}

			bulk[i] = tx.Payment.
				Create().
				SetID(info.ID).
				SetCreatedAt(info.CreateAt).
				SetUpdatedAt(info.UpdateAt).
				SetDeletedAt(info.DeleteAt).
				SetAppID(info.AppID).
				SetUserID(info.UserID).
				SetGoodID(info.GoodID).
				SetOrderID(info.OrderID).
				SetAccountID(info.AccountID).
				SetStartAmount(amount(info.StartAmount)).
				SetAmount(amount(info.Amount)).
				SetPayWithBalanceAmount(payWithBalance).
				SetFinishAmount(amount(info.Amount)).
				SetCoinUsdCurrency(amount(info.CoinUsdCurrency)).
				SetLocalCoinUsdCurrency(amount(info.LocalCoinUsdCurrency)).
				SetLiveCoinUsdCurrency(amount(info.LiveCoinUsdCurrency)).
				SetCoinInfoID(info.CoinInfoID).
				SetState(getPaymentState(info.State.String())).
				SetChainTransactionID(info.ChainTransactionID).
				SetUserSetPaid(info.UserSetPaid).
				SetUserSetCanceled(info.UserSetCanceled).
				SetFakePayment(info.FakePayment)
		}
		_, err = tx.Payment.CreateBulk(bulk...).Save(_ctx)
		return err
	})

	if err != nil {
		return err
	}

	return nil
}

func getPaymentState(payState string) string {
	switch payState {
	case "wait":
		return paymentmgrpb.PaymentState_Wait.String()
	case "done":
		return paymentmgrpb.PaymentState_Done.String()
	case "canceled":
		return paymentmgrpb.PaymentState_Canceled.String()
	default:
		return paymentmgrpb.PaymentState_DefaultState.String()
	}
}

func getOrderType(orderType string) string {
	switch orderType {
	case stant.OrderTypeNormal:
		return ordermgrpb.OrderType_Normal.String()
	case ordermgrpb.OrderType_Normal.String():
		return ordermgrpb.OrderType_Normal.String()

	case stant.OrderTypeOffline:
		return ordermgrpb.OrderType_Offline.String()
	case ordermgrpb.OrderType_Offline.String():
		return ordermgrpb.OrderType_Offline.String()

	case stant.OrderTypeAirdrop:
		return ordermgrpb.OrderType_Airdrop.String()
	case ordermgrpb.OrderType_Airdrop.String():
		return ordermgrpb.OrderType_Airdrop.String()

	default:
		return ordermgrpb.OrderType_DefaultOrderType.String()
	}
}

func getOrderState(payState string, userCanceled bool, start, end uint32) string {
	state := ordermgrpb.OrderState_DefaultState.String()
	switch payState {
	case stant.PaymentStateTimeout:
		state = ordermgrpb.OrderState_PaymentTimeout.String()
	case stant.PaymentStateWait:
		state = ordermgrpb.OrderState_WaitPayment.String()
	case stant.PaymentStateDone:
		state = ordermgrpb.OrderState_Paid.String()
	case stant.PaymentStateCanceled:
		state = ordermgrpb.OrderState_Canceled.String()
	}

	if state == ordermgrpb.OrderState_WaitPayment.String() && userCanceled {
		state = ordermgrpb.OrderState_UserCanceled.String()
	}

	now := uint32(time.Now().Unix())
	if state == ordermgrpb.OrderState_Paid.String() {
		if start < now {
			state = ordermgrpb.OrderState_InService.String()
		}
		if now > end {
			state = ordermgrpb.OrderState_Expired.String()
		}
	}

	return state
}
