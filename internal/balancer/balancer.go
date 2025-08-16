package balancer

import (
	"go-oai-gateway/internal/config"
	"math/rand"
	"sync"
	"time"
)

// Balancer 負責為多後端的模型提供負載平衡
type Balancer struct {
	// counters 是一個映射，key 是模型名稱，value 是該模型的請求計數器
	counters map[string]uint64
	// lock 用於保護對 counters 映射的併發讀寫
	lock sync.Mutex
}

// NewBalancer 創建並初始化一個新的 Balancer
func NewBalancer() *Balancer {
	// 使用當前時間作為隨機種子，確保每次啟動時的隨機序列都不同
	rand.Seed(time.Now().UnixNano())
	return &Balancer{
		counters: make(map[string]uint64),
	}
}

// Next 使用加權輪詢策略，返回下一個可用的後端端點
func (b *Balancer) Next(modelName string, endpoints []config.EndpointConfig) *config.EndpointConfig {
	if len(endpoints) == 0 {
		return nil
	}
	if len(endpoints) == 1 {
		return &endpoints[0]
	}

	totalWeight := 0
	for _, ep := range endpoints {
		// 如果權重未設定或小於等於0，給予一個預設的基礎權重1
		if ep.Weight <= 0 {
			totalWeight += 1
		} else {
			totalWeight += ep.Weight
		}
	}

	// 如果總權重為0（例如所有權重都未設定），則退化為簡單輪詢
	if totalWeight == 0 {
		b.lock.Lock()
		count := b.counters[modelName]
		b.counters[modelName] = count + 1
		b.lock.Unlock()
		index := count % uint64(len(endpoints))
		return &endpoints[index]
	}

	// 加權輪詢邏輯
	b.lock.Lock()
	randomWeight := rand.Intn(totalWeight)
	b.lock.Unlock()

	for _, ep := range endpoints {
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		if randomWeight < weight {
			return &ep
		}
		randomWeight -= weight
	}

	// 理論上不應該執行到這裡，但作為一個保障，返回最後一個端點
	return &endpoints[len(endpoints)-1]
}