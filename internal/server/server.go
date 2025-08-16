package server

import (
	"fmt"
	"go-oai-gateway/internal/balancer"
	"go-oai-gateway/internal/config"
	"go-oai-gateway/internal/discovery"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// Server 結構體包含了伺服器運行所需的所有依賴
type Server struct {
	config    *config.Config
	registry  *discovery.ModelRegistry
	balancer  *balancer.Balancer
	proxies   map[string]*httputil.ReverseProxy
	router    *gin.Engine
}

// NewServer 創建並初始化一個新的 Server
func NewServer(cfg *config.Config, registry *discovery.ModelRegistry) *Server {
	router := gin.Default()
	s := &Server{
		config:    cfg,
		registry:  registry,
		balancer:  balancer.NewBalancer(),
		proxies:   make(map[string]*httputil.ReverseProxy),
		router:    router,
	}

	// 為每個後端端點創建一個反向代理
	for _, endpoint := range cfg.Endpoints {
		target, err := url.Parse(endpoint.BaseURL)
		if err != nil {
			log.Printf("警告：無法解析後端 URL %s: %v", endpoint.BaseURL, err)
			continue
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		
		// 修改請求，以設定正確的 Host 和認證資訊
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			req.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
		}

		s.proxies[endpoint.Name] = proxy
	}

	s.setupRoutes()
	return s
}

// setupRoutes 設定所有 API 路由
func (s *Server) setupRoutes() {
	v1 := s.router.Group("/v1")
	{
		v1.GET("/models", s.handleGetModels)
		v1.POST("/chat/completions", s.handleChatCompletions)
	}
}

// Start 啟動 HTTP 伺服器
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	fmt.Printf("伺服器正在監聽 %s\n", addr)
	return s.router.Run(addr)
}

// handleGetModels 處理 /v1/models 請求
func (s *Server) handleGetModels(c *gin.Context) {
	// 這裡我們需要將 discovery.ModelRegistry.Models 轉換為 OpenAI API 的標準格式
	// 為了簡化，我們先直接返回一個包含模型名稱的列表
	
	type ModelInfo struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}

	type ModelsResponse struct {
		Object string      `json:"object"`
		Data   []ModelInfo `json:"data"`
	}

	var models []ModelInfo
	for name := range s.registry.Models {
		models = append(models, ModelInfo{
			ID:      name,
			Object:  "model",
			OwnedBy: "go-oai-gateway", // 使用自訂的擁有者名稱
		})
	}

	c.JSON(http.StatusOK, ModelsResponse{
		Object: "list",
		Data:   models,
	})
}

// ChatCompletionRequest 是 /v1/chat/completions 請求體的簡化結構
type ChatCompletionRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// handleChatCompletions 處理 /v1/chat/completions 的轉發請求
func (s *Server) handleChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	// 使用 Bind 對象，這樣即使在流式請求下也能讀取到 model 欄位
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求格式"})
		return
	}

	// 從註冊表中查找支援該模型的後端
	endpoints, ok := s.registry.Models[req.Model]
	if !ok || len(endpoints) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("模型 '%s' 不存在", req.Model)})
		return
	}

	// 使用負載平衡器選擇下一個後端
	selectedEndpoint := s.balancer.Next(req.Model, endpoints)
	if selectedEndpoint == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法為模型選擇可用的後端"})
		return
	}

	// 從代理池中獲取對應的代理
	proxy, ok := s.proxies[selectedEndpoint.Name]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("找不到提供商 '%s' 的代理", selectedEndpoint.Name)})
		return
	}

	log.Printf("請求模型 '%s'，轉發到 -> %s", req.Model, selectedEndpoint.Name)
	// 使用代理來處理請求
	proxy.ServeHTTP(c.Writer, c.Request)
}