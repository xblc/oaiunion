package discovery

import (
	"go-oai-gateway/internal/config"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Model 代表從後端 API 返回的單一模型資訊
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelList 是 /v1/models API 端點的回應結構
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// ModelRegistry 負責儲存和管理所有發現的模型
// 它建立了從一個公開的模型名稱到一個或多個後端端點的映射
type ModelRegistry struct {
	// Models 是一個映射，key 是對外公開的模型名稱 (例如 "gpt-4-turbo")
	// value 是支援該模型的所有後端端點設定
	Models map[string][]config.EndpointConfig
	// client 是一個可用於發出 HTTP 請求的客戶端
	client *http.Client
	// lock 用於保護對 Models 映射的併發寫入
	lock sync.RWMutex
}

// register 是一個併發安全的方法，用於註冊一個模型和它對應的端點
func (r *ModelRegistry) register(modelName string, endpoint config.EndpointConfig) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Models[modelName] = append(r.Models[modelName], endpoint)
}

// NewModelRegistry 創建並初始化一個新的模型註冊表
func NewModelRegistry(cfg *config.Config) (*ModelRegistry, error) {
	registry := &ModelRegistry{
		Models: make(map[string][]config.EndpointConfig),
		client: &http.Client{
			Timeout: 10 * time.Second, // 為請求設定 10 秒超時
		},
	}

	var wg sync.WaitGroup
	for _, endpoint := range cfg.Endpoints {
		wg.Add(1)
		go func(ep config.EndpointConfig) {
			defer wg.Done()
			registry.fetchAndRegisterModels(ep, cfg.Routing.Mode)
		}(endpoint)
	}

	wg.Wait()

	if cfg.Routing.Mode == "merge" {
		registry.applyModelOverrides(cfg.Routing.ModelOverrides)
	}

	return registry, nil
}

// fetchAndRegisterModels 從單個端點獲取模型列表並進行註冊
// 注意：這是一個模擬實現，用於開發和測試
func (r *ModelRegistry) fetchAndRegisterModels(endpoint config.EndpointConfig, mode string) {
	// --- 模擬 API 請求 ---
	// 在真實場景中，這裡會發出 HTTP GET 請求到 endpoint.BaseURL + "/v1/models"
	// 並解析回應。為了測試，我們返回一個基於端點名稱的假模型列表。
	mockModels := getMockModels(endpoint.Name)
	// --- 模擬結束 ---

	for _, model := range mockModels.Data {
		modelID := model.ID
		if mode == "prefix" {
			modelID = endpoint.Name + "/" + model.ID
		}
		r.register(modelID, endpoint)
	}
}

// applyModelOverrides 應用手動模型重寫規則
func (r *ModelRegistry) applyModelOverrides(overrides map[string][]string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// 創建一個臨時的端點映射，按 provider name 進行索引，方便查找
	providerEndpoints := make(map[string]config.EndpointConfig)
	for _, endpoints := range r.Models {
		for _, ep := range endpoints {
			providerEndpoints[ep.Name] = ep
		}
	}

	for unifiedName, realModelRefs := range overrides {
		var endpointsToMerge []config.EndpointConfig
		for _, modelRef := range realModelRefs {
			// 解析 "provider/model" 格式
			parts := strings.SplitN(modelRef, "/", 2)
			if len(parts) != 2 {
				continue // 忽略格式不正確的條目
			}
			providerName, modelName := parts[0], parts[1]

			// 檢查原始模型是否存在於該 provider 的端點下
			if originalEndpoints, ok := r.Models[modelName]; ok {
				for _, ep := range originalEndpoints {
					if ep.Name == providerName {
						endpointsToMerge = append(endpointsToMerge, ep)
						break // 找到匹配的端點，處理下一個 modelRef
					}
				}
			}
		}

		if len(endpointsToMerge) > 0 {
			// 註冊統一的新名稱
			r.Models[unifiedName] = append(r.Models[unifiedName], endpointsToMerge...)

			// 清理被合併的舊模型
			// 我們只清理那些被成功合併的原始模型
			for _, modelRef := range realModelRefs {
				parts := strings.SplitN(modelRef, "/", 2)
				if len(parts) != 2 {
					continue
				}
				modelName := parts[1]

				// 如果這個模型的所有後端都被合併了，就刪除這個模型
				// 這裡的邏輯需要小心，避免誤刪。一個簡單的策略是，如果
				// 舊模型只有一個後端，且這個後端被合併了，就刪除它。
				if len(r.Models[modelName]) == 1 {
					delete(r.Models, modelName)
				}
			}
		}
	}
}

// getMockModels 是一個輔助函式，返回用於測試的模擬模型列表
func getMockModels(providerName string) ModelList {
	switch providerName {
	case "provider-a":
		return ModelList{
			Data: []Model{
				{ID: "gpt-4-turbo-preview"},
				{ID: "claude-3-opus-20240229"},
			},
		}
	case "provider-b":
		return ModelList{
			Data: []Model{
				{ID: "gpt-4"},
				{ID: "claude-v3-opus"},
			},
		}
	case "provider-c-custom":
		return ModelList{
			Data: []Model{
				{ID: "gpt-4-1106-preview"},
				{ID: "dall-e-3"},
			},
		}
	default:
		return ModelList{Data: []Model{}}
	}
}