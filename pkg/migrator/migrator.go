package migrator

import (
	"context"
	"fmt"
	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	servicename "github.com/NpoolPlatform/order-gateway/pkg/servicename"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
	"github.com/NpoolPlatform/order-middleware/pkg/db/ent"
	entpowerrental "github.com/NpoolPlatform/order-middleware/pkg/db/ent/powerrental"
)

const (
	keyServiceID = "serviceid"
)

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(servicename.ServiceDomain, keyServiceID)
	return fmt.Sprintf("migrator:%v", serviceID)
}

func migratePowerRentals(ctx context.Context, tx *ent.Tx) error {
	logger.Sugar().Warnw("exec migratePowerRentals")
	now := uint32(time.Now().Unix())

	_, err := tx.PowerRental.
		Update().
		SetGoodStockMode(
			goodtypes.GoodStockMode_GoodStockByUnique.String(),
		).
		Where(
			entpowerrental.DeletedAtEQ(0),
			entpowerrental.GoodStockMode(goodtypes.GoodStockMode_DefaultGoodStockMode.String()),
		).
		SetUpdatedAt(now).
		Save(ctx)

	return err
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
		if err := migratePowerRentals(ctx, tx); err != nil {
			return err
		}
		logger.Sugar().Infow("Migrate", "Done", "success")
		return nil
	})
}
