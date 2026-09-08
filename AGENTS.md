# go-nx-admin 项目规则

Fiber + GORM + Casbin 后端，React + Refine + Shadcn/ui 前端；支持独立运行及 Git Submodule 二开、单文件 embed 二进制。

## 入口与扩展边界

- 二开后端只能 import 公开的 `go-nx-admin/app`；需跨 module 使用的包不得放 `backend/internal/`。
- `backend/app/` 的 `app.Run(app.Options)` 是启动与扩展入口：`EmbedFS` 嵌入前端，`ExtraModels` 注册迁移模型，`ExtraRoutes` 注入受 JWT + Casbin 保护的 `/api/v1` 路由，`AfterMigrate` 在底座 seed 后注入二开种子。
- 二开权限通过 `ExtraPermissions` / `ExtraPermissionExpand` 注入，业务代码与 submodule 同级独立维护，不直接二改底座。
- 前端通过 `frontend/src/core.tsx` 的 `createApp(opts)` 注入 `extraResources`、`extraRoutes`、`extraRouteLabels`，业务页面留在二开项目自己的 `frontend/src/pages/`。
- 修改扩展接口前先读上述公开入口；底座升级要保持二开可 import 和 embed 构建边界，不用内部包绕过公开 API。

## 定向定位

| 任务 | 先读入口 |
|---|---|
| 启动、构建、生成二开骨架 | `Makefile`、`backend/cmd/`、`backend/app/`、`scaffold.sh` |
| 模型、迁移、种子 | `backend/internal/models/` |
| API 路由与处理 | `backend/internal/routes/routes.go`、`backend/internal/handlers/` |
| JWT、Casbin | `backend/internal/middleware/` |
| 权限定义及保存 | 下文“权限维护” |
| 前端布局、Tabs、锁屏、侧栏 | `frontend/src/components/layout/`、`frontend/src/contexts/` |
| 页面、通用 UI、数据与认证 | `frontend/src/pages/`、`frontend/src/components/ui/`、`frontend/src/providers/` |
| 配置及默认值 | `config.example.yaml`、`backend/internal/config/config.go` |

表名和端点以模型、路由为准，不在规则中维护第二份全量目录。历史演进可参考 `/opt/mgaming/docs/20260612-0700-go-nx-admin-enhancement-plan.md`，旧计划不是当前权限权威规范。

## 权限维护

- 角色接口权限定义来自代码，不从 `apis` 表读取：内置定义和展开在 `backend/internal/models/permissions.go`，`AllPermissionDefs()` 合并内置与扩展定义。
- key 使用 `资源:操作`，由 `ExpandPermission()` 或二开 `ExtraPermissionExpand` 展开为路径和方法的 Casbin 策略。新增权限须同时维护定义和展开；删除路由时同步移除引用它的权限。
- `backend/internal/handlers/crud.go` 提供角色权限读取/保存；前端 `frontend/src/pages/roles/index.tsx` 通过 `GET /api/v1/roles/:slug/permissions` 拉取分组定义，通过 PUT 同路径提交 `{keys: [...]}`，后端展开策略。
- 底座新增定义不需要另改数据库 seed、前端组件或中间件；二开只使用公开注入点，不改底座 `permissions.go`。
- `AutoDiscoverAPIs()` 在路由注册后同步 API 登记表；登记/自动发现不替代权限定义与展开，也不代表新增路由已获得角色授权。
- 菜单授权只控制侧栏可见性，不能替代接口鉴权。权限弹窗按组展示 checkbox，组全选与下级联动。

## 认证与安全边界

- JWT 有效期 24h；`JWTAuth` 校验 token 对应的 `admin_online_users` 会话，MFA 验证路径不做在线会话检查。修改认证前先核对 middleware 和路由例外，不以菜单可见性代替认证。
- Casbin 模型为 `g(r.sub, p.sub) && regexMatch(r.obj, p.obj) && regexMatch(r.act, p.act)`；Admin 种子为 `admin, .*, .*`。原有 login/verify-password 等路由例外按注册实现维护，不擅自扩大放行范围。
- 审计异步写入须保持上下文隔离，避免复用请求 ctx。
- MFA 使用 TOTP：登录返回 `mfa_required` 后以六位码或恢复码换取正式 JWT；用户菜单提供设置，管理员用户管理入口可清空 MFA。
- 用户可管理自己的登录 IP 白名单，管理员可管理其他用户；密码通过后匹配，无规则默认放行。`NX_LOGIN_IP_WHITELIST_ENABLED=false` 可关闭开关，不能把调试关闭作为默认安全状态。
- 现有 IP 解析涉及 `X-Forwarded-For` / `X-Real-IP` / `CF-Connecting-IP`；geo 使用 ip-api，800ms 超时且失败留空。改动前核对实际代理信任边界，不把 header 或 geo 当作已验证身份。
- 每次登录建立独立会话，注销只清当前 token；只能踢非当前会话。
- 删除自身、删除系统角色、踢当前会话等危险动作必须前后端双重保护：按钮禁用，后端明确拒绝，不能只靠 UI。

## 前端规范

- CRUD 标题左、工具右；筛选用白色卡片，重置/查询靠右；分页左侧总数、右侧页码。日志必须后端分页筛选，禁止固定最近 N 条。
- 日期时间统一用 `frontend/src/components/ui/datetime-input.tsx` 的 `DateTimeInput`；flatpickr 全局样式及 Tailwind 污染回退沿用现有入口，不在页面重复实现。
- 不恢复暗黑皮肤：无 `dark:` 类、ThemeProvider 或切换入口。
- 在线详情用弹窗表格，不在主表格内展开。

## 配置与运行

- 运行配置用根目录 `config.yaml`，模板为 `config.example.yaml`；环境变量覆盖对应配置。数据库支持 SQLite/MySQL/Postgres，默认 SQLite 路径 `data/nx.db` 相对工作目录。
- `serve` 会自动迁移、建表及写种子，不能当只读健康检查。种子账号和默认 JWT/CORS 配置以模板、seed 为准，不把默认值当生产凭据。
- 后端 `server.host/port` 或 `NX_SERVER_HOST/PORT` 控制监听，默认 `172.18.0.1:19500`；健康入口 `/health`。
- Vite 默认 `172.18.0.1:5173`，`VITE_HOST` 覆盖 host，`/api` 代理到后端。运行前读 manifest/lock、Makefile，避免照搬二开项目命令。
- 常用入口：`make build-frontend` 构建并复制 embed 资源，`make dev-frontend` 热更新，`make build` 完整构建，`./nx-admin serve` 启动。密码重置命令会写数据，仅在授权内使用。

## Submodule 构建

- 底座自身 `make build` 将前端嵌入 `backend/cmd/frontend/dist/`，生成 `nx-admin`；`make clean` 会清构建产物。
- 二开在自己的项目根目录构建，入口为 `cmd/main.go`，不是 submodule 的 `backend/cmd/main.go`；二开前端产物进入 `cmd/frontend/dist/` 并由 `go:embed all:frontend/dist` 嵌入。
- `scaffold.sh` 生成 go.mod replace、公开 app/createApp 注入点、前端骨架、Makefile 和配置模板；使用前读脚本及生成项目的 Makefile，不在此维护重复创建教程。
- 业务 API 可用独立 `cmd/api/main.go` 构建独立二进制与端口，不与管理后台入口混用。
- 升级底座使用项目既有升级入口并重新构建，核对 submodule 锁定版本及改动；不要把可漂移远端更新当成已验证版本。
