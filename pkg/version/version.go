package version

import (
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/NpoolPlatform/go-service-framework/pkg/version"
	"github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func Version() (*basetypes.VersionResponse, error) {
	info, err := version.GetVersion()
	if err != nil {
		logger.Sugar().Errorf("get service version error: %+w", err)
		return nil, wlog.Errorf("get service version error: %w", err)
	}
	return &basetypes.VersionResponse{
		Info: info,
	}, nil
}
