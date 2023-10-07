//nolint
package migrator

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	servicename "github.com/NpoolPlatform/order-gateway/pkg/servicename"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
	"github.com/NpoolPlatform/order-middleware/pkg/db/ent"
)

const (
	keyServiceID = "serviceid"
)

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(servicename.ServiceDomain, keyServiceID)
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
		logger.Sugar().Infow("Migrate", "Done", "success")
		return nil
	})
}
