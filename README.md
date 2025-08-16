# Go OAI Gateway - Golang OpenAI 相容 API 聚合器

這是一個使用 Golang 開發的本地、高效、可配置的 OpenAI API 相容聚合與轉發服務。

## ✨ 核心功能

*   **多後端聚合**: 在單一的本地端點 (`127.0.0.1:8080`) 聚合多個不同的 OpenAI 相容 API 服務。
*   **統一 API 入口**: 無需在客戶端應用中管理多個 API Key 和 URL，所有請求都發送到本服務。
*   **智慧路由**: 根據請求的模型名稱，自動將請求轉發到支援該模型的後端服務。
*   **負載平衡**: 當多個後端支援同一個模型時，自動進行加權輪詢負載平衡。
*   **模型別名與合併**: 可通過設定檔將不同後端的相似模型（如 `gpt-4-turbo-preview` 和 `gpt-4-1106-preview`）合併為一個統一的名稱（如 `gpt-4-turbo`），並對其進行輪詢。
*   **流式傳輸支援**: 完全支援 `stream: true`，能夠像原生 API 一樣實現打字機效果。
*   **高效率與低延遲**: 基於 Golang 的高效能特性，為您的 AI 應用提供低延遲的本地轉發。

## 🚀 如何開始

### 1. 前提條件

*   安裝 [Go](https://go.dev/doc/install) (建議版本 1.21 或更高)。

### 2. 安裝

```bash
# 下載專案
git clone https://github.com/your-username/go-oai-gateway.git
cd go-oai-gateway

# 安裝依賴項
go mod tidy
```

### 3. 設定

複製或重命名 `config.yaml.example` 為 `config.yaml`，並根據您的需求進行修改。

```yaml
# config.yaml

# 服務監聽的本地地址和端口
server:
  host: "127.0.0.1"
  port: 8080

# 後端 API 端點列表
# 在這裡填入您自己的 OpenAI 相容服務提供商
endpoints:
  - name: "your-provider-1" # 端點的唯一標識符
    base_url: "https://api.your-provider-1.com/v1" # 服務的基礎 URL
    api_key: "sk-your-provider-1-key" # 您的 API Key
    weight: 10 # 加權輪詢的權重

  - name: "your-provider-2"
    base_url: "https://api.your-provider-2.com/v1"
    api_key: "sk-your-provider-2-key"
    weight: 5

# 模型路由與合併策略
routing:
  # 'merge': (推薦) 合併所有端點的同名模型，進行輪詢。
  # 'prefix': 為每個模型加上端點名稱前綴，例如 'your-provider-1/gpt-4'。
  mode: "merge"

  # 手動重寫和合併規則 (僅在 mode: 'merge' 下生效)
  model_overrides:
    "claude-3-opus":
      - "your-provider-1/claude-3-opus-20240229"
      - "your-provider-2/claude-v3-opus"
    "gpt-4-turbo":
      - "your-provider-1/gpt-4-turbo-preview"
```

### 4. 啟動服務

```bash
go run main.go
```

服務啟動後，將監聽在 `http://127.0.0.1:8080`。

### 5. 使用

現在，您可以將您的 AI 應用程式的 API 端點指向 `http://127.0.0.1:8080/v1`，並像使用單一 OpenAI 服務一樣使用它。

例如，使用 `curl` 測試：

```bash
# 獲取可用的模型列表
curl http://127.0.0.1:8080/v1/models

# 發送一個聊天請求
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4-turbo",
    "messages": [{"role": "user", "content": "你好！"}],
    "stream": false
  }'