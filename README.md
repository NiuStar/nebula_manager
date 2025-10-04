# Nebula Manager Web UI

Nebula Manager 是一个基于 Go（Gin + Gorm + MySQL）和 Vue 3 的 Nebula 组网控制面板，支持证书生成、节点配置与安装脚本一站式管理。

---

## 1. 程序如何运行

### 1.1 环境准备
- 安装 Go 1.24 及以上版本
- 安装 Node.js 18 及以上版本（包含 npm）
- 准备一套可访问的 MySQL 8+ 数据库，并创建库：例如 `nebula_manager`

### 1.2 配置后端环境变量
1. 在项目根目录（`/Volumes/code/go_work/src/nebula_manager`）创建 `.env`（可选）：
   ```bash
   cat <<'ENV' > .env
   NEBULA_MYSQL_DSN="root:123150.wangzai7@tcp(10.10.10.1:3306)/nebula_manager?charset=utf8mb4&parseTime=True&loc=Local"
   NEBULA_SERVER_PORT=8080
   NEBULA_DATA_DIR=data
   NEBULA_API_BASE="http://localhost:8080"
   NEBULA_BINARY_VERSION=1.9.3
   NEBULA_BINARY_BASE="https://github.com/slackhq/nebula/releases/download"
   NEBULA_FRONTEND_DIR="frontend/dist"
   NEBULA_BINARY_PROXY_PREFIX=""  # 例如 https://proxy.529851.xyz/
   NEBULA_ADMIN_USERNAME="admin"
   NEBULA_ADMIN_PASSWORD="admin123"
   NEBULA_SESSION_SECRET=""
   NEBULA_SESSION_SECURE="false"
   NEBULA_STATIC_TOKEN=""
   ENV
   ```
2. 或者直接在终端导出变量：
   ```bash
   export NEBULA_MYSQL_DSN="root:123150.wangzai7@tcp(10.10.10.1:3306)/nebula_manager?charset=utf8mb4&parseTime=True&loc=Local"
   export NEBULA_SERVER_PORT=8080
   export NEBULA_DATA_DIR=data
   export NEBULA_API_BASE="http://localhost:8080"
   export NEBULA_BINARY_VERSION=1.9.3
   export NEBULA_BINARY_BASE="https://github.com/slackhq/nebula/releases/download"
   export NEBULA_FRONTEND_DIR="frontend/dist"
   export NEBULA_BINARY_PROXY_PREFIX=""
   export NEBULA_ADMIN_USERNAME="admin"
   export NEBULA_ADMIN_PASSWORD="admin123"
   export NEBULA_SESSION_SECRET=""
   export NEBULA_SESSION_SECURE="false"
   export NEBULA_STATIC_TOKEN=""
   ```

> 其中 `NEBULA_API_BASE` 用于生成安装脚本时填充控制面板访问地址，节点脚本执行时可通过 `NEBULA_MANAGER_API` 覆盖；若未设置，后端会尝试自动检测本机对外 IP 并组合默认地址。`NEBULA_BINARY_VERSION` / `NEBULA_BINARY_BASE` 可用于指定 Nebula 官方二进制的版本与下载源，默认指向 GitHub Releases。`NEBULA_BINARY_PROXY_PREFIX` 可选，用于指定代理前缀（示例：`https://proxy.529851.xyz/`），脚本会自动将其与下载地址拼接。`NEBULA_ADMIN_USERNAME` / `NEBULA_ADMIN_PASSWORD` 定义登录凭据，`NEBULA_SESSION_SECRET` 用于签发会话令牌（默认随机生成），`NEBULA_SESSION_SECURE` 为 `true` 时会在 HTTPS 下强制使用 `Secure` Cookie。`NEBULA_STATIC_TOKEN`（可选）提供一枚固定的访问令牌，适合脚本化部署；若设置，UI 中的安装命令会默认引用该 token。`NEBULA_FRONTEND_DIR` 指定静态文件目录，默认指向编译后的 `frontend/dist`。

### 1.3 启动后端 API
```bash
go run ./...
```
控制台看到 `Listening and serving HTTP on :8080` 表示后端已启动，API 根路径为 `http://localhost:8080/api`。

> 首次启动会在 `NEBULA_DATA_DIR`（默认 `./data`）创建目录并自动执行数据库迁移。

### 1.4 启动前端界面
```bash
cd frontend
npm install
npm run dev
```
浏览器访问 `http://localhost:5173`，即可打开 Nebula Manager 控制台。Vite 开发服务器已将 `/api` 请求代理到 `http://localhost:8080`。

> 生产环境可执行 `npm run build` 产出静态文件（位于 `frontend/dist`），再交由 Nginx 等静态服务器托管。

## 登录与访问控制

- 控制台及所有 `/api` 接口需要先登录，默认账号/密码见 `NEBULA_ADMIN_USERNAME` 与 `NEBULA_ADMIN_PASSWORD`（请尽快在生产环境中修改）。
- 登录方式：发送 `POST /api/login`，请求体示例：
  ```json
  {"username": "admin", "password": "admin123"}
  ```
  成功后会返回 `token`，并通过 `nebula_session` HttpOnly Cookie 维护会话。
- 若设置了 `NEBULA_STATIC_TOKEN`，控制台会将该值内置到安装命令和脚本，可直接复制执行；若未设置，需要先通过 `POST /api/login` 获取 token，并在目标主机 `export NEBULA_ACCESS_TOKEN=<token>` 后再运行命令。安装命令会使用 `Authorization: Bearer ...` 头部请求 `/install-script`，脚本内部也会复用该 token 访问 `/bundle`。
- 删除节点：`DELETE /api/nodes/<id>`；命令行可附加 `Authorization: Bearer <token>` 请求头，或在 URL 后追加 `?access_token=<token>`。
- 前端 SPA 会自动在路由切换时检测会话；退出登录可调用 `POST /api/logout`，或在页面右上角点击“退出登录”。
- 若需要在命令行下载节点脚本/配置，可携带登录返回的 token：
  ```bash
  curl -fsSL -H "Authorization: Bearer <TOKEN>" "http://<controller>/api/nodes/1/install-script"
  ```
  或在执行安装脚本前导出 `NEBULA_ACCESS_TOKEN`（脚本会自动把该变量转换为 `Authorization: Bearer ...` 请求头，用于访问 `/api/nodes/<id>/bundle` 等接口）。


---

## 2. 如何建立灯塔节点（Lighthouse）

灯塔节点用于协调其他节点的打洞与发现流程，通常需要拥有公网可达的 IP 或端口映射。

### 2.1 生成或更新 CA 证书
1. 打开前端，进入 **Dashboard** 页面。
2. 在「Certificate Authority」卡片填写：
   - Name：如 `Nebula Root`
   - Description：如 `生产环境根证书`
   - Validity (days)：建议 365 或更长
3. 点击 **Generate / Replace CA**。成功后页面会展示当前 CA 名称与创建时间。
4. 点击 **Download CA Cert** 可下载 `nebula-ca.crt`，供其他主机预信任。

### 2.2 配置全局网络参数
1. 在同一页面的「Network Settings」卡片填写：
   - Default Subnet：例如 `10.10.0.0/24`
   - Handshake Port：默认 `4242`
   - Certificate Validity：证书有效期天数（与节点证书关联）
   - Lighthouse Hosts：可填 `lighthouse.example.com`、`203.0.113.5` 等（逗号分隔，可选）
2. 点击 **Save Settings** 保存。

### 2.3 创建灯塔节点
1. 切换到 **Nodes** 页面。
2. 在「Provision Node」表单填写：
   - Name：如 `lighthouse-1`
   - Role：选择 `Lighthouse`
   - Subnet IP：节点在 Nebula 虚拟网中的地址，例如 `10.10.0.1`；可直接写 `/24`（如 `10.10.0.1/24`），不带掩码时默认使用 `/24`
   - Public IP / Host：灯塔对外可达的公网 IP 或域名（用于其他节点连接）
   - Listen Port：可留空使用全局 Handshake Port
   - 下载代理：按目标主机网络环境选择 `不使用代理`、`IPv4 代理` 或 `IPv6 代理`（目前统一使用 `https://proxy.529851.xyz/`），仅影响二进制下载，不会改变 `static_host_map`
   - Tags：可选，逗号分隔
3. 点击 **Create**。
4. 在列表中找到刚创建的灯塔节点，使用“安装命令”列提供的 `curl ... | bash` 指令，并复制备用（会实时引用最新的安装脚本）；若节点不再需要，可点击“删除”按钮回收。

### 2.4 在目标主机安装灯塔
1. 将复制的安装命令在目标主机执行（需已安装 Nebula 二进制且能够访问控制面板 API）：
   ```bash
   export NEBULA_ACCESS_TOKEN="<登录接口返回的 token>"
   # 如控制面板地址非默认，可先设置
   export NEBULA_MANAGER_API="http://控制面板主机:8080"

   curl -fsSL -H "Authorization: Bearer ${NEBULA_ACCESS_TOKEN}" "${NEBULA_MANAGER_API:-http://<controller>:8080}/api/nodes/<id>/install-script" | bash
   ```
   - `NEBULA_ACCESS_TOKEN` 可通过 `POST /api/login` 获得，脚本执行过程中也会使用该 token 下载节点归档；若不想修改命令，可直接在复制的命令中将 `<ACCESS_TOKEN>` 替换为实际 token。
   - 若在同一浏览器内下载，可复用 Cookie，命令中的 `NEBULA_ACCESS_TOKEN` 可省略。
   - 命令会自动访问 `/api/nodes/<id>/bundle` 接口下载归档，并写入 `/etc/nebula` 下的 `ca.crt`、节点证书/私钥与 `config.yml`。
   - 安装脚本会识别 Linux CPU 架构，按 `NEBULA_BINARY_VERSION` 指定的版本下载 Nebula 官方二进制，并安装到 `/usr/local/bin/nebula`。
   - 脚本会生成 `nebula.service` systemd 单元，执行 `systemctl enable --now nebula.service` 实现开机自启。
   > 需确保目标主机已安装 `curl` 与 `tar`，脚本内部会调用二者完成下载与解压。
3. 将 Nebula 可执行文件与 `config.yml` 配合 systemd 或直接命令启动：
   ```bash
   sudo nebula -config /etc/nebula/config.yml
   ```
4. 确保防火墙开放 Handshake Port，并在需要时配置端口转发。

---

## 3. 如何建立普通服务节点（Standard）

普通节点加入同一虚拟子网，使用灯塔进行服务发现。

### 3.1 前置条件
- 已完成上文的 CA 生成与网络设置。
- 至少有一个在线的灯塔节点，其配置中的 `static_host_map` 和 `lighthouse.hosts` 会自动包含已创建的灯塔信息。

### 3.2 创建普通节点
1. 在 **Nodes** 页面点击「Provision Node」。
2. 填写：
   - Name：如 `web-1`
   - Role：保持 `Standard`
   - Subnet IP：例如 `10.10.0.11` 或 `10.10.0.11/24`（未写掩码时默认 `/24`）
   - Public IP / Host：可选，若节点仅作为客户端可留空
   - Listen Port：通常留空使用默认值
   - 下载代理：可按需选择 `IPv4` / `IPv6` 代理或关闭
   - Tags：如 `prod,web`
3. 点击 **Create**，等待成功提示，并确认节点出现在列表。
4. 同灯塔步骤，在节点列表中复制“安装命令”并在目标主机执行；如需移除节点，可直接点击“删除”。

### 3.3 在目标主机部署普通节点
1. 在目标主机执行刚才复制的命令：
   ```bash
   export NEBULA_ACCESS_TOKEN="<登录接口返回的 token>"
   export NEBULA_MANAGER_API="http://控制面板主机:8080"

   curl -fsSL -H "Authorization: Bearer ${NEBULA_ACCESS_TOKEN}" "${NEBULA_MANAGER_API}/api/nodes/<id>/install-script" | bash
   ```
   - 命令会根据节点配置自动选择下载代理，并通过 API 下载证书、密钥、配置以及 Nebula 二进制。
   - 若使用浏览器下载（已登录），也可以手动在命令中将 `<ACCESS_TOKEN>` 占位符替换为实际 token。
   > 请确认节点主机具备 `curl` 与 `tar`。
3. 运行 Nebula：
   ```bash
   sudo nebula -config /etc/nebula/config.yml
   ```
4. 通过 `nebula status` 或 `systemctl status nebula`（如果配置了服务）检查运行状态。
5. 确认节点可以 Ping 通虚拟网中的其他节点，例如：
   ```bash
   ping 10.10.0.1  # 测试是否能到达灯塔
   ```

---

## 附：常见排查提示
- 创建节点前必须先生成 CA，否则后端会返回 “CA not generated yet”。
- `NEBULA_DATA_DIR` 目录需对后端进程可写，否则文件写入会失败。
- MySQL 用户需具备创建表、插入、更新权限；如遇连接问题，请检查 DSN、网络或防火墙设置。
- 运行脚本时如提示权限不足，可在命令前加 `sudo`。

---

## 多平台打包脚本

如需批量生成不同系统/架构的发布包，可执行：

```bash
./scripts/package_release.sh v1.2.3
```

- 目标包含 Linux（amd64/arm64/386）、Windows（amd64）与 macOS（amd64/arm64）。
- 脚本会编译后端、拷贝 `README.md` 与 `config.yaml.default`（若存在）、打包 `frontend/dist`，最终产物位于 `build/packages/`。
- 版本号参数可省略，默认为 `git describe --tags` 的结果或当前日期。
- 需具备 Go 工具链与 Node/npm；若缺少前端构建产物，会自动执行 `npm install && npm run build`。

---

## 二进制安装脚本（非 Docker）

若希望直接在宿主机上部署 Nebula Manager，可在仓库根目录执行：

```bash
sudo ./scripts/install_binary.sh
```

- 脚本会检测/构建前端产物与后端二进制，安装到 `/opt/nebula-manager`（可通过 `INSTALL_DIR` 环境变量覆盖）。
- 运行过程中会提示输入 MySQL DSN、监听端口、数据目录等信息，并生成 `/etc/nebula-manager.env`。
- 自动写入并启用 `nebula-manager.service` systemd 单元，方便 `systemctl status|restart nebula-manager` 管理。
- 如需修改配置，请编辑 `/etc/nebula-manager.env` 后执行 `sudo systemctl restart nebula-manager`。

按以上步骤，即可完成后端 API 与前端控制台的部署，并通过 Web UI 快速生成灯塔节点与普通节点的证书、配置和安装脚本，实现 Nebula 组网的集中管理。

---

## 节点间网络质量采集

为实现“节点 ↔ 节点”级别的延迟监控，需要在每个节点上部署一个轻量探针脚本，由节点自行对其他节点发起 `ping` 并把结果上报到控制面板。后端已提供以下接口：

- `GET  /api/nodes/:id/network?range=1h|6h|24h`：查询指定节点在最近一段时间内对其它节点的延迟曲线（前端图表使用的接口，不需要额外操作）。
- `POST /api/nodes/:id/network/samples`：由节点自报数据，JSON 请求体形如：
  ```json
  {
    "samples": [
      {"peer_id": 2, "latency_ms": 23.7, "success": true, "timestamp": "2024-11-27T10:15:00Z"},
      {"peer_id": 3, "latency_ms": 0, "success": false, "timestamp": "2024-11-27T10:15:02Z"}
    ]
  }
  ```
  - `peer_id`：被测节点在控制面板中的 ID。
  - `latency_ms`：以毫秒为单位的往返延迟，失败时可置为 0。
  - `success`：本次探测是否成功。
  - `timestamp`：ISO8601 / RFC3339 格式时间戳，可选；未提供时后端会使用接收时间。
- `GET /api/nodes/:id/network/targets`：返回推荐的探测目标（包含节点 ID、名称与地址），便于探针自动获取最新列表。
- `POST /api/nodes/:id/status`：上报节点运行状态，字段包括 CPU/Load、内存、磁盘、Swap、网络累计字节、进程数、Uptime 等，`reported_at` 可选。
- `GET /api/public/status`：无需登录即可获取节点状态概览，适合对外只读展示。

### 推荐的探针部署方式

通过安装命令部署节点时，脚本会自动：

1. 下载并安装 `/usr/local/bin/nebula-network-agent.sh`；
2. 写入 `/etc/nebula/nebula-network-agent.env`（自动使用后端 `NEBULA_API_BASE` 以及 `NEBULA_STATIC_TOKEN`，若存在）；
3. 安装 `nebula-net-probe.service` 和 `nebula-net-probe.timer`，默认在启动 60 秒后运行并每分钟触发一次：
   - 执行前通过 `GET /api/nodes/:id/network/targets` 自动同步最新节点列表（可通过 `NEBULA_DYNAMIC_TARGETS=0` 关闭）；
   - 采集 `CPU/内存/磁盘/Swap/进程/负载/网络流量/运行时长` 等信息，连同 Ping 样本一起上报控制端。

若控制端未配置 `NEBULA_STATIC_TOKEN`，请在执行安装命令前导出 `NEBULA_ACCESS_TOKEN`，或稍后补充后运行：

```bash
sudo tee /etc/nebula/nebula-network-agent.env <<'ENV'
NEBULA_MANAGER_API="https://controller.example.com"
NEBULA_ACCESS_TOKEN="<token>"
NEBULA_NODE_ID=<your-node-id>
NEBULA_PEERS="2:10.10.0.12,3:10.10.0.13"
ENV

sudo systemctl enable --now nebula-net-probe.timer
```

仓库仍提供一个可独立运行的脚本 `scripts/node-network-agent.sh`，便于手动或自定义部署：

```bash
export NEBULA_MANAGER_API="https://controller.example.com"          # 控制面板地址
export NEBULA_ACCESS_TOKEN="<静态 token 或登录获得的 token>"
export NEBULA_NODE_ID=1                                              # 当前节点 ID
export NEBULA_PEERS="2:10.10.0.12,3:10.10.0.13"                     # 目标 ID:IP 列表

/opt/nebula/node-network-agent.sh
```

脚本要点：
- 使用 `ping` 测试每个目标（默认 1 包，3 秒超时，可通过 `NEBULA_AGENT_PING_COUNT` 与 `NEBULA_AGENT_PING_TIMEOUT` 调整）。
- 默认会拉取 `GET /api/nodes/:id/network/targets`，实时刷新 `NEBULA_PEERS`（可设置 `NEBULA_DYNAMIC_TARGETS=0` 关闭）。
- 自动汇总节点运行状态（CPU、内存、磁盘、Swap、网络累计字节、平均负载、进程数、Uptime），并调用 `POST /api/nodes/:id/status` 上报。
- 推荐使用全局 `NEBULA_STATIC_TOKEN` 作为访问凭据，避免会话 token 过期导致探针上报失败。
- 支持在 Nebula overlay 内使用子网 IP 直接探测，也可以配置公网地址或任意可达的探测目标。
- 脚本依赖 `python3` 用于解析 `/proc` 指标，若节点缺少 python 会提示“跳过运行状态上报”。

示例 systemd timer：

```ini
# /etc/systemd/system/nebula-net-probe.service
[Unit]
Description=Nebula node latency reporter

[Service]
Type=oneshot
Environment=NEBULA_MANAGER_API=https://controller.example.com
Environment=NEBULA_ACCESS_TOKEN=<token>
Environment=NEBULA_NODE_ID=1
Environment=NEBULA_PEERS=2:10.10.0.12,3:10.10.0.13
ExecStart=/opt/nebula/node-network-agent.sh

# /etc/systemd/system/nebula-net-probe.timer
[Unit]
Description=Run Nebula latency reporter every minute

[Timer]
OnUnitActiveSec=60s
Unit=nebula-net-probe.service

[Install]
WantedBy=timers.target
```

启用：

```bash
sudo systemctl enable --now nebula-net-probe.timer
```

只要各节点按计划上报，前端“网络情况”页面即可展示真实的节点间延迟曲线，并在表格中列出最近一次探测的结果。
