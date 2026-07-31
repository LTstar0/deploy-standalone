# 部署与配置手册 (Deploy Standalone & Platform Manual)

本手册详细介绍了发布系统的配置规范、脚本开发标准、产品发布包的打包格式，以及 Web 可视化部署平台的管理说明。

---

## 一、 应用配置文件规范 (`apps.json`)

系统的一切行为（包括依赖关系、部署顺序、健康检查、脚本路径）均由 JSON 配置文件驱动。

### 1. 完整字段结构示例

```json
{
  "project": {
    "name": "我的生产项目",
    "version": "1.2.0",
    "environment": "production"
  },
  "deploy_config": {
    "parallel_execution": false,
    "stop_on_failure": true,
    "backup_before_deploy": false
  },
  "apps": {
    "database": {
      "name": "MySQL 数据库",
      "description": "主业务数据库",
      "enabled": true,
      "dependencies": [],
      "deploy_order": 1,
      "health_check": {
        "command": "mysqladmin ping -h 127.0.0.1 --silent",
        "timeout": 30
      },
      "rollback_support": true,
      "scripts": {
        "pre_deploy": "",
        "deploy": "apps/database/deploy.sh",
        "post_deploy": "",
        "health_check": "",
        "rollback": ""
      }
    },
    "api-backend": {
      "name": "API 后端服务",
      "description": "基于 Spring Boot 的核心业务接口",
      "enabled": true,
      "dependencies": ["database"],
      "deploy_order": 2,
      "health_check": {
        "url": "http://localhost:8080/actuator/health",
        "timeout": 60
      },
      "rollback_support": true,
      "scripts": {
        "pre_deploy": "apps/api-backend/pre-deploy.sh",
        "deploy": "apps/api-backend/deploy.sh",
        "post_deploy": "apps/api-backend/post-deploy.sh",
        "health_check": "apps/api-backend/health-check.sh",
        "rollback": "apps/api-backend/rollback.sh"
      }
    }
  }
}
```

### 2. 核心字段详细说明

#### 全局配置部分 (`deploy_config`)
*   `stop_on_failure` (bool, 默认 `true`): 任意应用部署失败时是否立即中断后续部署。如果设为 `false`，则会继续部署其他没有依赖冲突的应用。
*   `backup_before_deploy` (bool, 默认 `false`): 预留配置，决定是否在部署前自动备份当前运行物。
*   `parallel_execution` (bool, 默认 `false`): 预留配置。

#### 应用配置部分 (`apps.<key>`)
*   `<key>`: 应用的唯一标识符（即 JSON 中的 Key），用于在依赖分析和回滚中定位应用。
*   `name` (string): 页面/终端交互展示的友好名称。
*   `enabled` (bool, 默认 `true`): 是否参与发布流程。若设为 `false`，它将不会出现在可选择的应用列表中。
*   `dependencies` (array[string]): 依赖的其他应用的 `<key>`。部署此应用时，系统会强制检查并补全依赖。
*   `deploy_order` (int): 决定部署先后顺序的权重值。值越小越先部署。相同权重的部署顺序不确定。
*   `rollback_support` (bool): 标识是否支持回滚。若为 `true`，失败时会执行对应应用的 `rollback` 脚本。
*   `health_check` (object): 健康检查配置（三种模式优先级：`scripts.health_check` > `health_check.url` > `health_check.command`）。
    *   `url`: 检测的 HTTP 服务地址，返回状态码在 `200-299` 之间即判定为就绪。
    *   `command`: 执行一段 shell 命令，当退出码为 `0` 时判定为就绪。
    *   `timeout`: 最大探测等待超时时间（秒）。
*   `scripts` (object): 各生命周期关联的脚本路径（路径需为相对于产品包根目录的相对路径，例如 `apps/redis/deploy.sh`）。

---

## 二、 应用部署生命周期脚本规范

应用部署需遵循标准的生命周期，分为以下五个钩子：

```
[开始部署] ────► 1. Pre-deploy (准备)
                       │
                       ▼
                 2. Deploy (执行部署)
                       │
                       ▼
                 3. Health Check (健康探测)
                       │
                       ▼
                 4. Post-deploy (后置处理) ────► [部署成功]
                       │
             (若任一阶段失败且确认回滚)
                       │
                       ▼
                 5. Rollback (逆序回滚)
```

### 1. 脚本编写准则
1.  **幂等性 (Idempotency)**：部署脚本和回滚脚本必须能够安全地多次重复运行。例如：部署脚本在目录已存在时应该覆盖或升级，而不是直接抛出错误退出；端口占用时应先停掉旧进程。
2.  **正确的退出状态码**：
    *   脚本执行成功必须以 `exit 0` 退出。
    *   遇到非预期错误，必须以非 0 退出码（如 `exit 1`）退出，系统以此判定部署阶段失败并触发容灾逻辑。
3.  **日志输出**：脚本内部应该有明确的步骤输出（使用 `echo` 等标准输出）。在 Web 平台部署时，这些输出会实时通过 WebSocket 呈现在前端的控制台中。

---

## 三、 产品发布包打包标准 (Packaging Standard)

在可视化 Web 部署平台中，用户是通过上传一个产品包（`.tar.gz` 或 `.zip`）来注册并发布应用的。该产品包必须符合以下结构要求：

### 1. 发布包的标准目录结构

```
my-product-1.2.0.tar.gz/ (解压后结构)
├── config/
│   └── apps.json            # 必需！产品包中所有应用的配置清单
├── apps/
│   ├── api-backend/
│   │   ├── deploy.sh        # 必需！
│   │   ├── health-check.sh  # 可选，用于自定义复杂健康检查
│   │   └── rollback.sh      # 可选，用于支持一键回滚
│   └── web-frontend/
│       ├── deploy.sh
│       └── rollback.sh
├── scripts/
│   └── common.sh            # 可选，公共 Bash 脚本或依赖工具
└── dist/                    # 可选，存放预编译好的文件或应用程序本体
```

### 2. 打包命令示例
在包含 `config/` 和 `apps/` 的项目根目录下运行：
```bash
# 打包为 tar.gz 格式
tar -czvf my-product-v1.2.0.tar.gz config/ apps/ scripts/ [其他静态资源]

# 打包为 zip 格式
zip -r my-product-v1.2.0.zip config/ apps/ scripts/ [其他静态资源]
```

---

## 四、 产品包升级与重复部署机制 (Upgrades & Repeated Deployment)

在生产环境下，同一个产品包可能会被部署多次，或者用户会上传新的产品包版本（如从 `v1.2.0` 升级至 `v1.3.0`）。系统针对此类场景，通过 **“平台层加锁隔离”** 与 **“应用脚本层幂等设计”** 两个层面来保证安全性与正确性。

### 1. 平台层：并发部署锁与工作空间隔离
* **并发控制锁 (Concurrency Lock)**：
  为了防止多个用户同时操作部署、或对同一台服务器并发写入，Web 平台后端在执行部署/回滚时，会启用**全局并发锁**。在任意一个部署或回滚任务状态为 `running` 或 `rolling_back` 时，新的部署或回滚请求将被直接拒绝（返回 `409 Conflict` 或 `已有其他部署或回滚任务正在执行` 错误）。
* **多版本工作空间隔离 (Workspace Isolation)**：
  每个上传的产品包都会分配一个独立的 Package ID（如 `ab12cd34`），并解压到独立的目录 `data/workspaces/ab12cd34/` 下。多次上传或不同版本的应用部署完全隔离，互不影响。

### 2. 脚本层：应用升级与重复部署的幂等设计 (Idempotency)
平台仅作为部署脚本的宿主与编排引擎，具体的服务停止、文件覆盖、重复部署防错等升级逻辑由 **产品包中的 `deploy.sh` 脚本** 负责实现。

#### 重复部署/升级的脚本编写最佳实践：
以后端 API 服务部署为例，在编写 `deploy.sh` 时，应遵循以下幂等流程：

1. **优雅停止旧进程**：
   通过进程名称、端口或 PID 文件定位正在运行的旧实例，优雅停止它：
   ```bash
   # 根据端口优雅停止旧服务
   PORT=8080
   PID=$(lsof -t -i:$PORT || true)
   if [ -n "$PID" ]; then
       echo "发现旧版本服务运行中 (PID: $PID)，正在停止..."
       kill "$PID"
       sleep 3
       # 强杀以确保端口释放
       kill -9 "$PID" 2>/dev/null || true
   fi
   ```
2. **安全覆盖目标运行物**：
   在复制文件时，使用 `cp -f` 或 `rsync` 覆盖，并保证目标目录结构正确，而不是直接新建：
   ```bash
   # 准备运行目录并安全复制
   TARGET_DIR="/opt/my-app"
   mkdir -p "$TARGET_DIR"
   cp -rf dist/* "$TARGET_DIR/"
   ```
3. **拉起新进程并记录 PID**：
   在后台启动服务并重定向日志：
   ```bash
   cd "$TARGET_DIR"
   nohup ./my-backend-binary > stdout.log 2>&1 &
   echo $! > app.pid
   echo "新版本服务已拉起"
   ```

通过上述幂等性脚本编写，即可支持新旧版本的平滑升级与多次重复部署。

---

## 五、 Web 可视化平台运维指南

可视化平台（`src/platform`）的后端完全是用 Go 编写的轻量服务。

### 1. 数据存储与目录说明
运行平台后，会在配置的 `DATA_DIR`（默认 `./data`）下生成以下文件和目录：
*   `data/packages/`：保存上传的原始打包文件。
*   `data/workspaces/<package_id>/`：产品包解压后的工作空间目录。部署脚本会在此目录下执行。
*   `data/packages.json`：持久化保存的所有已上传包的数据元信息。
*   `data/history.json`：持久化保存的部署与回滚历史记录。

### 2. Makefile 核心指令
在 `src/platform/` 目录下：
*   `make dev-backend`：在本地开发模式启动 Go 后端（port 9090）。
*   `make dev-frontend`：在本地开发模式启动 Vue 3 前端（port 3000）。
*   `make build`：编译打包前后端，并把前端静态资源存入发布区。
*   `make clean`：清理所有构建的二进制和临时数据。

### 3. 界面操作示意

以下为 Web 可视化平台主要操作界面截图，便于对照使用流程。

#### 3.1 产品包管理

上传产品包（`.tar.gz` / `.tgz` / `.zip`），解析后即可在列表中查看应用组成，并进入部署或删除：

![产品包管理 — 上传与包列表](docs/image.png)

#### 3.2 部署控制台 — 选择应用

进入某产品包的部署控制台后，可按拓扑顺序勾选需要发布的应用（如 MySQL、Redis、API 后端、Web 前端），确认后点击「开始部署」：

![部署控制台 — 选择待部署应用](docs/image2.png)

#### 3.3 部署控制台 — 实时日志

部署过程中，平台通过 WebSocket 将各应用的 `deploy.sh` 输出与健康检查结果实时推送到终端面板：

![部署控制台 — 实时部署日志](docs/image4.png)

#### 3.4 部署失败时的日志与排障

若脚本缺失、权限不足或健康检查失败，控制台会标记任务为「失败」，并在日志中输出具体错误（例如脚本路径不存在、`exit status 127`），便于对照 FAQ 排查：

![部署控制台 — 部署失败日志](docs/image3.png)

---

## 六、 Web 平台安全与认证机制 (Security & Authentication)

为保障生产环境下的部署安全，Web 可视化平台内置了发布者身份校验机制，拦截未授权的应用包上传、发布执行、回滚及删除请求。

### 1. Token 配置与生成规则
系统支持两种 Token 注入方式，优先级从高到低：
1. **环境变量**：在后台服务启动前设置 `DEPLOY_TOKEN` 环境变量进行覆盖：
   ```bash
   export DEPLOY_TOKEN="your-custom-secure-token"
   ./deploy-platform
   ```
2. **本地文件自动生成**：若未配置环境变量，服务启动时将尝试读取 `data/token.txt`。如果该文件不存在，系统将自动生成一个 16 位强随机的安全发布 Token（形如 `deploy_163cdb113fbe3d9f`）持久化写入该文件，并在终端启动日志中清晰输出。

> [!NOTE]
> 本地开发或首次部署时，建议直接查看控制台打印的 Token，或者读取服务器上的 `data/token.txt` 文件的内容。

### 2. 认证传参规范 (API & WebSocket)
客户端必须通过以下方式之一传递有效的 Token 才能通过安全网关：
* **HTTP 请求头（二选一）**：
  * 使用自定义头：`X-Deploy-Token: <token>`
  * 使用标准授权头：`Authorization: Bearer <token>` (或直接 `Authorization: <token>`)
* **WebSocket 握手查询参数**：
  由于浏览器不支持在 WebSocket 握手时直接附带 Headers，因此在进行 WS 日志订阅时需使用 URL 参数传值：
  `ws://localhost:9090/ws/deploy/:taskId?token=<your_token>`

### 3. 前端认证与自动静默登录
* Vue 前端系统在首次挂载时会调用 `/api/verify` 接口对 `localStorage.getItem('deploy_token')` 中的凭证进行静默校验。
* 若校验不通过或本地无凭据，页面会渲染出“发布 Token 校验”卡片，阻止未授权用户进行任何页面操作。
* 成功输入并验证 Token 后，会写入本地 `localStorage`，后续所有 Axios API 请求均会经由请求拦截器自动附带验证 Header。
* 用户可以在侧边栏底部点击“退出登录”清空本地缓存以重新锁定终端。

---

## 七、 Web 平台 API 参考

以下是 Web 可视化部署平台提供的 API 接口，可用于二次开发或 CI/CD 对接（均受上述 Token 机制保护）。

### 1. 认证与验证
*   **验证 Token 有效性**
    *   **Method**: `GET`
    *   **Path**: `/api/verify`
    *   **Headers**: 需要携带有效的 `X-Deploy-Token` 或 `Authorization`
    *   **Response**: `200 OK` 且内容为 `{"status":"ok"}`

### 2. 基础包管理
*   **上传产品包**
    *   **Method**: `POST`
    *   **Path**: `/api/packages/upload`
    *   **Content-Type**: `multipart/form-data`
    *   **Payload**: `file` (File)
    *   **Response**: `200 OK` 返回包 ID 及解析到的应用列表。
*   **获取包列表**
    *   **Method**: `GET`
    *   **Path**: `/api/packages`
*   **删除包**
    *   **Method**: `DELETE`
    *   **Path**: `/api/packages/:id`

### 3. 部署控制
*   **启动部署任务**
    *   **Method**: `POST`
    *   **Path**: `/api/deploy/start`
    *   **JSON Payload**:
        ```json
        {
          "package_id": "ab12cd34",
          "selected_apps": ["api-backend", "web-frontend"]
        }
        ```
    *   **Response**: `200 OK` 返回 `task_id` 和任务初始状态。
*   **获取部署历史**
    *   **Method**: `GET`
    *   **Path**: `/api/deploy/history`
*   **一键回滚历史部署**
    *   **Method**: `POST`
    *   **Path**: `/api/deploy/rollback/:historyId`

### 4. 实时日志通道 (WebSocket)
*   **WS 握手地址**: `ws://<host>:<port>/ws/deploy/:taskId?token=<your_token>`
*   **数据流向**:
    *   连接建立后，后端会自动向客户端推送当前任务已产生的历史日志。
    *   随后有新的 shell 输出时，会实时推送格式如下的 JSON 帧：
        ```json
        {
          "type": "log",
          "data": {
            "time": "2026-06-15T17:00:00Z",
            "level": "info",
            "message": "  执行 deploy.sh 成功"
          }
        }
        ```
    *   部署完成时，发送结束帧：
        ```json
        {
          "type": "done",
          "data": "success"
        }
        ```

---

## 八、 常见问题与排障 (FAQ)

#### 1. 部署过程中健康检查一直等待并超时？
*   **排查**：请确认是否配置了正确的端口。若使用 URL 检测模式，检查目标服务启动后是否能被部署平台所在的主机正常访问（特别注意防火墙及 `localhost` 的解析）。
*   **调整**：可在 `apps.json` 中适当调大该应用的 `health_check.timeout` 探测时间。

#### 2. 在 Web 平台上点击部署，控制台报“脚本权限不足”？
*   **排查**：Go 后端在解压包后会自动给脚本赋予可执行权限，但如果仍然报错，请确保上传的产品包在打包时，脚本的权限是正确无误的，或者在打包前运行 `chmod +x`。

#### 3. 部署失败后回滚没有生效？
*   **排查**：请确认在 `apps.json` 中配置了 `"rollback_support": true` 并且对应的 `"scripts.rollback"` 指向了正确的回滚脚本。如果未配置或者为 `false`，系统在回滚时会选择直接跳过该应用。