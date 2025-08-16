package main

import (
	"fmt"
	"log"

	"go-oai-gateway/internal/config"
	"go-oai-gateway/internal/discovery"
	"go-oai-gateway/internal/server"
)

func main() {
	// 讀取設定檔
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("無法讀取設定檔: %v", err)
	}

	// 進行模型發現與註冊
	registry, err := discovery.NewModelRegistry(cfg)
	if err != nil {
		log.Fatalf("模型發現失敗: %v", err)
	}

	// 打印最終的模型列表以進行驗證
	fmt.Println("模型註冊完成！")

	// 創建並啟動伺服器
	srv := server.NewServer(cfg, registry)
	if err := srv.Start(); err != nil {
		log.Fatalf("伺服器啟動失敗: %v", err)
	}
}