package strategy

import (
	"context"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
)

type GridStrategy struct {
	svcCtx   *svc.ServiceContext
	strategy *ent.Strategy
}

func NewGridStrategy(svcCtx *svc.ServiceContext, s *ent.Strategy) *GridStrategy {
	return &GridStrategy{svcCtx: svcCtx, strategy: s}
}

func (s *GridStrategy) Get() *ent.Strategy {
	return s.strategy
}

func (s *GridStrategy) OnUpdate(ctx context.Context) error {
	state, err := LoadGridStrategyState(ctx, s.svcCtx, s.strategy)
	if err != nil {
		logger.Errorf("[GridStrategy] 加载策略状态失败, id: %s, symbol: %s, account: %s, %v",
			s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, err)
		return err
	}

	if err = state.Rebalance(); err != nil {
		logger.Errorf("[GridStrategy] 处理网格再平衡失败, id: %s, symbol: %s, account: %s, %v",
			s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, err)
		return err
	}

	return nil
}
