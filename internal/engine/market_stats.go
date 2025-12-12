package engine

import "github.com/fachebot/omni-grid-bot/internal/exchange"

// getMarketStrategyList 获取市场策略列表
func (engine *StrategyEngine) getMarketStrategyList(exchange, symbol string) []Strategy {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()

	strategyList := make([]Strategy, 0)
	for _, item := range engine.strategyMap {
		s := item.Get()
		if s.Exchange != exchange || s.Symbol != symbol {
			continue
		}
		strategyList = append(strategyList, item)
	}

	return strategyList
}

// processMarketStats 处理市场状态
func (engine *StrategyEngine) processMarketStats(exchange string, marketStats exchange.MarketStats) {
	strategyList := engine.getMarketStrategyList(exchange, marketStats.Symbol)
	for _, s := range strategyList {
		s.OnTicker(engine.ctx, marketStats.MarkPrice)
	}
}
