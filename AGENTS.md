# go-nx-admin — Fiber + Casbin + Refine + Shadcn/ui 全栈脚手架

## 二开扩展接口

`backend/app/` 是公开入口，二开项目只能 import `go-nx-admin/app`。不要把需要给二开项目 import 的包放在 `backend/internal/` 下，Go 的 `internal` 规则会阻止跨 module 引用。

### 后端：`app.Options`

```go
// 二开项目 cmd/main.go
app.Run(app.Options{
    EmbedFS: sub,                                   // 前端 dist 的 embed.FS
    ExtraModels: []interface{}{&MyModel{}},          // 额外需要 AutoMigrate 的模型
    ExtraRoutes: func(api fiber.Router, db *gorm.DB, e *casbin.Enforcer) {
        api.Get("/orders", handlers.ListOrders(db))  // 二开路由（自动经过 JWT + Casbin）
    },
    AfterMigrate: func(db *gorm.DB) {
        // 底座 seed 之后执行，可在此插入二开种子数据
    },
    ExtraPermissions: []app.PermissionDef{
        {Key: "orders:list", Name: "查看订单", Group: "订单管理"},
    },
    ExtraPermissionExpand: func(key string) [][]string {
        if key == "orders:list" { return [][]string{{"GET", "/api/v1/orders"}} }
        return nil
    },
})
```

### 前端：`createApp(opts)`

```tsx
// 二开项目 frontend/src/main.tsx
import { createApp } from 'go-nx-admin-frontend/core'
import { OrderListPage } from './pages/orders'

const App = createApp({
    extraResources: [{ name: 'orders', list: '/orders' }],
    extraRoutes: [<Route key="orders" path="/orders" element={<OrderListPage />} />],
    extraRouteLabels: { '/orders': '订单管理' },  // 面包屑标签
})
```

## Submodule 二开模式结论

`go-nx-admin` 可以作为 Git Submodule 放在二开项目根目录，业务代码与 submodule 同级独立维护，最终仍可构建为单文件 embed 二进制。

推荐二开项目结构：

```text
my-project/
├── go-nx-admin/              # git submodule，底座代码，只升级不二改
├── cmd/main.go               # 管理后台入口，调用 go-nx-admin/app.Run
├── cmd/frontend/dist/        # make build-frontend 后复制来的前端产物，供 go:embed
├── cmd/api/main.go           # 可选：独立 API 服务入口，单独监听端口
├── internal/models/          # 二开业务模型
├── internal/handlers/        # 二开业务 handler
├── frontend/src/pages/       # 二开业务页面
├── frontend/src/main.tsx     # createApp 注入业务路由/页面/面包屑
├── go.mod                    # replace go-nx-admin => ./go-nx-admin/backend
└── Makefile
```

已验证能力：
- 业务 Go 包放在二开项目 `internal/` 下，不影响管理后台构建。
- 业务前端页面放在二开项目 `frontend/src/pages` 下，Vite 构建时会和底座前端一起打包。
- `make build` 会生成一个管理后台单二进制，前端 dist 通过 `cmd/main.go` 的 `go:embed all:frontend/dist` 嵌入。
- 同一二开项目可另起 `cmd/api`，构建独立 API 服务二进制，监听独立端口。
- 二开 API 通过 `ExtraRoutes` 挂到受保护的 `/api/v1` group，默认走 JWT + Casbin。
- 二开菜单/种子数据通过 `AfterMigrate` 写入数据库。

## 目录布局
| 目录 | 说明 |
|------|------|
| `backend/` | Go 后端（Fiber、GORM、Casbin） |
| `backend/cmd/` | 入口 main.go，内含 `go:embed frontend/dist` |
| `backend/app/` | 可配置启动器 `Run(opts Options)`，供二开项目 import |
| `backend/internal/models/` | 数据模型、迁移、种子 |
| `backend/internal/handlers/` | 请求处理 |
| `backend/internal/middleware/` | JWT / Casbin 鉴权 |
| `backend/internal/routes/` | 路由注册 |
| `frontend/` | React 前端（Refine + Shadcn/ui） |
| `frontend/src/core.tsx` | 导出 `createApp(opts)`，供二开项目 import |
| `frontend/src/components/layout/` | Tabs、锁屏、侧栏、用户菜单、全局工具 |
| `frontend/src/components/ui/` | DateTimeInput、Dialog、Button、Input 等 |
| `frontend/src/pages/` | 各功能页面 |
| `frontend/src/contexts/` | tabs、i18n、lock-screen 状态管理 |
| `frontend/src/providers/` | Refine auth/data provider |
| `frontend/src/types/` | TypeScript 类型定义 |

## 快速开始

```bash
make build-frontend   # 前端构建 + 复制 dist 到 embed 目录
./nx-admin serve      # 启动后端（自动迁移、种子、建库、监听）
make dev-frontend     # 前端 Vite 热更新（另一个终端）
make build            # 完整构建单二进制
```

CLI 子命令：
```bash
./nx-admin serve                        # 启动 HTTP 服务（默认 0.0.0.0:19500）
./nx-admin reset-password admin         # 重置 admin 密码为 admin123
./nx-admin reset-password zhangsan pwd  # 重置 zhangsan 密码为 pwd
```

种子账号：`admin` / `admin123`

健康检查：`curl http://172.18.0.1:19500/health`

## 数据模型

| 模型 | 表 | 说明 |
|------|-----|------|
| User | `admin_users` | 用户，含 MFA 字段 |
| Role | `admin_roles` | 角色，`is_system=true` 不可删除 |
| Menu | `admin_menus` | 侧栏菜单树 |
| RoleMenu | `admin_role_menus` | 角色-菜单关联 |
| Api | `admin_apis` | API 端点登记，用于角色接口权限弹窗 |
| AuditLog | `admin_audit_logs` | 操作审计日志 |
| LoginLog | `admin_login_logs` | 登录日志，含 geo 地理信息 |
| OnlineUser | `admin_online_users` | 在线会话，token 唯一索引，多设备登录 |
| UserIPWhitelist | `admin_user_ip_whitelists` | 用户登录 IP 白名单 |
| CasbinRule | `admin_casbin_rule` | Casbin 权限策略 |
| Media | `media` | 媒体文件，无 `admin_` 前缀 |
| SystemConfig | `system_configs` | 系统配置，无 `admin_` 前缀 |

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录，MFA 用户返回 `mfa_required` |
| POST | `/api/v1/auth/logout` | 注销当前会话 |
| POST | `/api/v1/auth/mfa/setup` | 生成 TOTP secret 和二维码 URL |
| POST | `/api/v1/auth/mfa/enable` | 验证 TOTP 后启用 MFA，返回恢复码 |
| POST | `/api/v1/auth/mfa/disable` | 密码 + TOTP 关闭 MFA |
| POST | `/api/v1/auth/mfa/verify` | MFA 验证码/恢复码校验，签发正式 JWT |
| POST | `/api/v1/auth/mfa/recovery-codes` | 重新生成恢复码 |
| POST | `/api/v1/auth/verify-password` | 锁屏密码验证 |
| GET | `/api/v1/auth/me` | 当前用户信息，含 `mfa_enabled` |
| PUT | `/api/v1/auth/profile` | 修改用户名 |
| GET | `/api/v1/auth/menus` | 当前角色授权的菜单树 |
| GET/POST | `/api/v1/users` | 用户列表（含在线状态、会话列表）/ 创建 |
| PUT/DELETE | `/api/v1/users/:id` | 编辑 / 删除（不能删自己） |
| DELETE | `/api/v1/users/:id/mfa` | 管理员清空指定用户 MFA |
| GET/POST | `/api/v1/users/:id/ip-whitelists` | 用户 IP 白名单列表 / 新增 |
| PUT/DELETE | `/api/v1/users/:id/ip-whitelists/:wid` | 修改 / 删除白名单规则 |
| GET/POST | `/api/v1/roles` | 角色列表 / 创建 |
| PUT/DELETE | `/api/v1/roles/:id` | 编辑 / 删除（系统角色不可删） |
| GET/PUT | `/api/v1/roles/:id/menus` | 角色菜单授权 |
| GET/PUT | `/api/v1/roles/:slug/policies` | 角色 Casbin 接口策略 |
| GET/POST | `/api/v1/menus` | 菜单树 / 创建 |
| PUT/DELETE | `/api/v1/menus/:id` | 编辑 / 递归删除子菜单 |
| GET/POST | `/api/v1/apis` | API 端点列表（支持分页） / 新增 |
| PUT/DELETE | `/api/v1/apis/:id` | 编辑 / 删除 |
| GET | `/api/v1/audit-logs` | 操作日志（分页+筛选） |
| GET | `/api/v1/login-logs` | 登录日志（分页+筛选+geo） |
| GET | `/api/v1/online-users` | 在线用户列表 |
| DELETE | `/api/v1/online-users/:id` | 踢下线（不能踢自己） |

## 鉴权架构

### JWT
- 环境变量 `NX_JWT_SECRET`，默认 `nx-admin-jwt-secret-2024`
- 签发 24h 有效期
- 中间件 `JWTAuth` 校验 token 是否在 `admin_online_users` 表中
- MFA 验证路径 `/auth/mfa/*` 不做在线会话检查

### Casbin RBAC
- 模型：`g(r.sub, p.sub) && regexMatch(r.obj, p.obj) && regexMatch(r.act, p.act)`
- Admin 角色种子：`admin, .*, .*`
- 所有 `/api/` 路径经过 Casbin 认证（login、verify-password 除外）
- 异步审计日志（读写分离，避免 ctx 复用 panic）

### 角色权限管理
- **菜单授权**：树形勾选，控制侧栏可见性
- **接口权限**：从 `apis` 表读取，按分组展示复选框，管理员只需勾选无需手写正则

### API 自动发现
- `main.go` 在 `routes.Register` 后调用 `models.AutoDiscoverAPIs`
- 遍历 Fiber 路由栈，自动将 `/api/v1/*` 路由同步到 `apis` 表
- 新增/删除 CRUD 接口后重启自动生效，无需手动维护 seed

## 安全特性

### MFA 两步验证
- TOTP 标准（`github.com/pquerna/otp/totp`）
- 流程：登录 → `mfa_required` → 6 位码或恢复码 → 签发 JWT
- 前端 MFA 设置入口：右上角用户菜单 → MFA 设置
- 管理员可在用户管理页清空任意用户的 MFA

### 登录 IP 白名单
- 用户可配置自己的白名单（账号管理弹窗）
- 管理员可在用户管理页管理任意用户的白名单
- 登录时：密码通过后匹配白名单，无规则默认放行
- 环境变量 `NX_LOGIN_IP_WHITELIST_ENABLED=false` 可全局关闭

### 登录 IP 解析
- 优先读取 `X-Forwarded-For` / `X-Real-IP` / `CF-Connecting-IP`
- 通过 `ip-api.com` 解析 geo 信息，超时 800ms，失败放空

### 在线会话
- 每次登录创建新会话（多设备支持）
- 用户管理可展开查看在线会话，踢除非当前会话
- 注销仅清除当前 token 对应会话

## 前端规范

### CRUD 页面
- 标题行左侧标题，右侧“列表工具区”（新增等操作）
- 筛选条件在白色卡片中，按钮靠右（重置 → 查询）
- 分页：左侧 `共 N 条`，右侧页码
- 日志页必须后端分页筛选，禁止固定最近 N 条

### 日期时间选择器
- 统一使用 `<DateTimeInput>`（`src/components/ui/datetime-input.tsx`）
- 基于 `flatpickr`，`enableTime: true, dateFormat: 'Y-m-d H:i', time_24hr: true`
- `flatpickr/dist/flatpickr.min.css` 在 `main.tsx` 全局引入
- `index.css` 有专用 CSS 回退 Tailwind 对 flatpickr 内部的 border-color 污染

### 暗黑皮肤
- 已彻底移除，无 `dark:` 类、无 ThemeProvider、无切换入口

### 危险操作保护
- 前后端双重保护：删除自身、删除系统角色、踢当前会话等
- 前端按钮禁用置灰，后端返回明确错误

### 在线详情
- 使用弹窗表格承载，不在主表格内展开

## 权限维护（RBAC）

**权限定义位置**：底座内置权限在 `backend/internal/models/permissions.go`；二开权限通过 `app.Options.ExtraPermissions` 和 `ExtraPermissionExpand` 注入，不需要改 submodule。

**理由**：权限本质是 API 路由的分组标签，路由定义在代码里，权限就该跟路由放一起。放数据库会导致"增了路由忘记插 DB"或"DB 删了权限但路由还在"的同步漏洞。

### 权限结构

权限采用 `资源:操作` 命名（如 `users:create`），分组展示（如"用户管理"组）。每个权限 key 在 `ExpandPermission()` 函数中展开为具体 Casbin 策略（路径+方法）。

```go
{Key: "users:list",   Name: "查看用户", Group: "用户管理"},
{Key: "users:create", Name: "新增用户", Group: "用户管理"},
{Key: "roles:menus",  Name: "菜单授权", Group: "角色管理"},
```

### 底座内新增权限的步骤

1. 在 `permissions.go` 的 `PermissionDefs` 数组中加一行
2. 在 `ExpandPermission()` 的 switch 中加 case，返回该权限对应的 Casbin 策略二维数组 `[][]string{{method, path}, ...}`
3. `make build` 重新编译即可

**不需要**改数据库 seed、不需要改前端组件、不需要改中间件。前端的角色权限对话框自动从 `GET /api/v1/roles/:slug/permissions` 拉取列表并渲染。

### 二开项目新增权限

二开项目不要改 `go-nx-admin/backend/internal/models/permissions.go`。在项目自己的 `cmd/main.go` 里注入：

```go
app.Run(app.Options{
    ExtraPermissions: []app.PermissionDef{
        {Key: "products:list", Name: "查看产品", Group: "产品管理"},
        {Key: "products:create", Name: "新增产品", Group: "产品管理"},
    },
    ExtraPermissionExpand: func(key string) [][]string {
        switch key {
        case "products:list":
            return [][]string{{"GET", "/api/v1/products"}}
        case "products:create":
            return [][]string{{"POST", "/api/v1/products"}}
        default:
            return nil
        }
    },
})
```

这些权限会出现在角色管理的权限弹窗中，并参与 Casbin 策略保存。

### 删除路由时

如果删除了某个 API 路由：
1. 确认 `permissions.go` 中是否还有权限引用该路由，有则一并移除
2. `apis` 表由 `AutoDiscoverAPIs()` 自动同步，无需手动维护

### 前端权限对话框

- 在角色管理页点击"接口权限"按钮打开
- 按分组展示，每个权限一行 checkbox
- 组名左侧有全选勾选框（与下级联动）
- 保存时 `PUT /api/v1/roles/:slug/permissions` 发送 `{keys: [...]}`，后端自动展开为 Casbin 策略

### 文件清单

| 文件 | 作用 |
|------|------|
| `backend/internal/models/permissions.go` | 权限定义 + 展开逻辑 |
| `backend/internal/handlers/crud.go` | `ListRolePermissions` + `SaveRolePolicies` |
| `frontend/src/pages/roles/index.tsx` | 权限对话框 UI |
| `backend/internal/routes/routes.go` | 路由注册 |

## 配置文件

项目根目录 `config.yaml` 控制所有运行时行为。首次使用从模板复制：

```bash
cp config.example.yaml config.yaml
```

### 数据库

| 驱动 | DSN 示例 |
|------|----------|
| sqlite | `data/nx.db` |
| mysql | `user:pass@tcp(localhost:3306)/nxadmin?charset=utf8mb4&parseTime=True&loc=Local` |
| postgres | `host=localhost user=postgres password=xxx dbname=nxadmin port=5432 sslmode=disable` |

首次启动自动 `AutoMigrate` 建库建表 + 种子，不需要手动创建。

## 环境变量

环境变量会覆盖 `config.yaml` 中的对应值。

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `NX_JWT_SECRET` | JWT 签名密钥 | `nx-admin-jwt-secret-2024` |
| `NX_CORS_ORIGINS` | 跨域允许来源 | `*` |
| `NX_LOGIN_IP_WHITELIST_ENABLED` | IP 白名单总开关 | `true`（设 `false` 关闭） |

## 监听地址
后端通过 `config.yaml` 的 `server.host` / `server.port` 或环境变量 `NX_SERVER_HOST` / `NX_SERVER_PORT` 配置监听地址，当前默认 `172.18.0.1:19500`。
Vite dev 默认 `172.18.0.1:5173`，可用 `VITE_HOST` 覆盖 host，代理 `/api` → `http://172.18.0.1:19500`。

## 构建产物
- 单二进制 `nx-admin`，`go:embed all:frontend/dist`
- `make build` → 前端 build → 复制到 `backend/cmd/frontend/dist/` → Go 编译
- DB 路径 `data/nx.db`（相对工作目录）
- `make clean` 清理所有构建产物

### Submodule 二开项目构建产物

二开项目的管理后台入口在项目根目录 `cmd/main.go`，不是 submodule 的 `backend/cmd/main.go`。构建必须在二开项目根目录执行：

```bash
make build
```

生成流程：
1. `cd frontend && npm run build` 生成二开前端产物。
2. 复制 `frontend/dist/*` 到 `cmd/frontend/dist/`。
3. `go mod tidy` 补齐二开项目 `go.sum`。
4. `go build -o <项目名> ./cmd` 从二开项目根目录编译。

最终得到一个单文件后台二进制，包含底座后端、二开后端、底座前端、二开前端。

如果要另起独立业务 API 服务，可在二开项目中添加 `cmd/api/main.go`，用 `go build -o biz-api ./cmd/api` 单独构建，独立监听端口。

## 以此为骨架创建新项目（Git Submodule 模式）

```bash
# 1. 创建二开项目目录
mkdir my-project && cd my-project && git init

# 2. 引入底座 submodule（锁定版本）
git submodule add -b v1.2.3 https://github.com/xxx/go-nx-admin.git go-nx-admin

# 3. 生成骨架文件（go.mod / cmd/main.go / frontend / Makefile）
./go-nx-admin/scaffold.sh my-project

# 4. 启动开发
cp config.example.yaml config.yaml
cd frontend && npm install
make build   # 构建前后端
./my-project serve
```

脚本会自动：
1. 生成 `go.mod`（含 `replace go-nx-admin => ./go-nx-admin/backend`）
2. 生成 `cmd/main.go`（import `go-nx-admin/app`，预留 ExtraModels/ExtraRoutes/AfterMigrate/ExtraPermissions 注入点）
3. 生成 `frontend/` 完整骨架（import 底座 `createApp`，预留 extraResources/extraRoutes/extraRouteLabels 注入点）
4. 生成 `Makefile`、`.gitignore`、`config.example.yaml`

### 升级底座

```bash
make upgrade   # 即 git submodule update --remote go-nx-admin
```

然后 `make build` 重新编译即可。

### 独立运行底座

底座仓库自身仍然支持 `make build && ./nx-admin serve` 独立启动，与子模块模式完全兼容。
