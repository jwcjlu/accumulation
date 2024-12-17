package bandwidth

import (
	"accumulation/framework/bandwidth/api"
	model2 "accumulation/framework/bandwidth/model"
	"context"
	"fmt"
)

type UseCase struct {
	bandwidthReportManager api.BandwidthReportManager
}

// NewBandWidthUseCase NewBandWidth .
func NewBandWidthUseCase(bandwidthReportManager api.BandwidthReportManager) *UseCase {
	return &UseCase{bandwidthReportManager: bandwidthReportManager}
}

func (useCase *UseCase) Start(ctx context.Context, gameStarted *model2.GameStarted) error {
	session := &model2.Session{
		Start:        gameStarted.Start,
		FlowID:       gameStarted.FlowID,
		BizID:        gameStarted.BizID,
		GID:          gameStarted.GID,
		UUID:         fmt.Sprintf("%d", gameStarted.UUID),
		VMid:         gameStarted.VMid,
		AreaType:     gameStarted.AreaType,
		InstanceId:   gameStarted.InstanceId,
		Idc:          gameStarted.Idc,
		StreamIp:     gameStarted.StreamIp,
		StreamPorts:  gameStarted.StreamPorts,
		EIP:          gameStarted.EIP(),
		ImageVersion: gameStarted.ImageVersion,
	}
	return useCase.bandwidthReportManager.StartReport(ctx, session)
}

func (useCase *UseCase) Stop(ctx context.Context, gameStop *model2.GameStop) error {
	session := &model2.Session{
		Start:      gameStop.Start,
		FlowID:     gameStop.FlowID,
		BizID:      gameStop.BizID,
		GID:        gameStop.GID,
		UUID:       gameStop.UUID,
		VMid:       gameStop.VMid,
		AreaType:   gameStop.AreaType,
		InstanceId: gameStop.InstanceId,
	}
	return useCase.bandwidthReportManager.EndReport(ctx, session)
}

/*func (useCase *UseCase) NotifyAccessInfo(ctx context.Context, req *v1.NotifyAccessInfoReq) error {
	var streamPorts []model.StreamPort
	err := json.Unmarshal([]byte(req.StreamPort), &streamPorts)
	if err != nil {
		return err
	}
	return useCase.bandwidthReportManager.NotifyAccessInfo(ctx, req.Vmid, req.StreamIp, streamPorts)
}*/
