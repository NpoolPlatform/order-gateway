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

	cconst "github.com/NpoolPlatform/cloud-hashing-order/pkg/message/const"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	corderent "github.com/NpoolPlatform/cloud-hashing-order/pkg/db/ent"

	"time"

	constant "github.com/NpoolPlatform/go-service-framework/pkg/mysql/const"
	"github.com/NpoolPlatform/order-manager/pkg/db"
	"github.com/NpoolPlatform/order-manager/pkg/db/ent"
)

func Migrate(ctx context.Context) error {
	return migrateFinishAmount(ctx)
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
func migrateFinishAmount(ctx context.Context) (err error) {
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

	paymentInfos, err := cloudOrderCli.
		Payment.
		Query().
		All(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}

	paymentMap := map[uuid.UUID]*corderent.Payment{}
	for _, payment := range paymentInfos {
		paymentMap[payment.OrderID] = payment
	}

	err = db.WithTx(ctx, func(_ctx context.Context, tx *ent.Tx) error {
		infos, err := tx.
			Payment.
			Query().
			All(_ctx)
		if err != nil {
			return err
		}

		for _, info := range infos {
			payment, ok := paymentMap[info.OrderID]
			if !ok {
				continue
			}

			_, err := tx.
				Payment.
				UpdateOneID(info.ID).
				SetFinishAmount(amount(payment.FinishAmount)).
				Save(_ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
