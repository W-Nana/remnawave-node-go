# Remnawave Node Go

[English](README.md)

高效能 Go 語言重寫的 [Remnawave Node](https://github.com/remnawave/node)，內嵌 xray-core。此節點連接至 Remnawave 面板，提供 VLESS、Trojan 和 Shadowsocks 代理服務。

## 特色功能

- **內嵌 xray-core** - 無需外部 xray 執行檔
- **多協議支援** - VLESS、Trojan、Shadowsocks
- **即時管理** - 新增/移除用戶無需重啟
- **流量統計** - 按用戶、入站、出站統計
- **自動更新 Geo** - 每週自動更新 geoip/geosite
- **單一目錄** - 所有檔案位於 `/etc/remnawave-node`

## 快速安裝

```bash
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) -s <SECRET_KEY> -p <NODE_PORT>
```

### 參數說明

| 參數 | 說明 |
|------|------|
| `-s, --secret` | 面板 SECRET_KEY（必填） |
| `-p, --port` | 節點端口（預設：3000） |
| `-v, --version` | 指定版本 |

### 使用範例

```bash
# 安裝並設定密鑰與端口
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) -s 你的SECRET_KEY -p 3000

# 更新至最新版本
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) update

# 僅更新 Geo 資料檔
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) update-geo

# 解除安裝
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) uninstall
```

## 安裝目錄

所有檔案安裝至 `/etc/remnawave-node/`：

```
/etc/remnawave-node/
├── remnawave-node-go    # 執行檔
├── .env                 # 設定檔
├── geoip.dat           # GeoIP 資料（每週自動更新）
└── geosite.dat         # GeoSite 資料（每週自動更新）
```

## 服務管理

```bash
# 啟動
systemctl start remnawave-node-go

# 停止
systemctl stop remnawave-node-go

# 狀態
systemctl status remnawave-node-go

# 檢視日誌
journalctl -u remnawave-node-go -f

# 開機自啟
systemctl enable remnawave-node-go
```

## 設定檔

編輯 `/etc/remnawave-node/.env`：

```bash
SECRET_KEY=your-secret-key-here
NODE_PORT=3000
XRAY_LOCATION_ASSET=/etc/remnawave-node
```

## 從原始碼編譯

```bash
# 複製專案
git clone https://github.com/W-Nana/remnawave-node-go.git
cd remnawave-node-go

# 編譯
make build

# 執行
./remnawave-node-go
```

## API 端點

### 主服務器（mTLS + JWT）

| 方法 | 路徑 | 說明 |
|------|------|------|
| `POST` | `/node/xray/start` | 啟動 xray |
| `GET` | `/node/xray/stop` | 停止 xray |
| `GET` | `/node/xray/status` | 取得狀態 |
| `GET` | `/node/xray/healthcheck` | 健康檢查 |
| `POST` | `/node/handler/add-user` | 新增用戶 |
| `POST` | `/node/handler/add-users` | 批次新增用戶 |
| `POST` | `/node/handler/remove-user` | 移除用戶 |
| `POST` | `/node/handler/remove-users` | 批次移除用戶 |
| `POST` | `/node/stats/get-users-stats` | 取得用戶統計 |
| `GET` | `/node/stats/get-system-stats` | 取得系統統計 |

### 內部服務器（僅限本機）

| 方法 | 路徑 | 說明 |
|------|------|------|
| `GET` | `/internal/get-config` | 取得 xray 設定 |
| `POST` | `/vision/block-ip` | 封鎖 IP |
| `POST` | `/vision/unblock-ip` | 解除封鎖 IP |

## 致謝

本專案是原始 [Remnawave Node](https://github.com/remnawave/node)（TypeScript/NestJS）的 Go 語言重寫版本。

## 授權條款

詳見 LICENSE 檔案。
