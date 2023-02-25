//nolint
package migrator

import (
	"context"
	"fmt"
	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/order-manager/pkg/db/ent"
	orderent "github.com/NpoolPlatform/order-manager/pkg/db/ent/order"
	"github.com/shopspring/decimal"

	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	constant1 "github.com/NpoolPlatform/order-gateway/pkg/message/const"
	"github.com/NpoolPlatform/order-manager/pkg/db"
)

const keyServiceID = "serviceid"

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(constant1.ServiceName, keyServiceID)
	return fmt.Sprintf("migrator:%v", serviceID)
}

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
				"update orders set units_v1='0' where units_v1 is NULL",
			)
		if err != nil {
			return err
		}

		infos, err := tx.
			Order.
			Query().
			Where(
				orderent.UnitsV1(decimal.NewFromInt(0)),
			).
			All(_ctx)
		if err != nil {
			return err
		}

		for _, info := range infos {
			_, err := tx.
				Order.
				UpdateOneID(info.ID).
				SetUnitsV1(decimal.NewFromInt32(int32(info.Units))).
				Save(_ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
