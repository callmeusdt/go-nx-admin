# go-nx-admin 项目规则

Fiber + GORM + Casbin 后端，React + Refine + Shadcn/ui 前端；支持独立运行及 Git Submodule 二开、单文件 embed 二进制。

## 入口与扩展边界

- 二开后端只能 import 公开的 `go-nx-admin/app`；需跨 module 使用的包不得放 `backend/internal/`。
- `backend/app/` 的 `app.Run(app.Options)` 是启动与扩展入口：`EmbedFS` 嵌入前端，`ExtraModels` 注册迁移模型（在底座表之后迁移，可外键引用 `admin_users` 等），`ExtraRoutes` 注入受 JWT + Casbin 保护的 `/api/v1` 路由，`AfterMigrate` 在底座 seed 后注入二开种子。`app.Run` 遇启动错误即 `log.Fatal`；`app.RunE(opts)` 返回 error、不终止进程，二者共用同一个 HTTP 栈：`RunE` 做配置、建目录、迁移、种子、策略后，委托 `NewRuntime`（`PublicUploads:true`）并在 SIGINT/SIGTERM 时关停。新增 HTTP 中间件、错误处理、`/health` 等只改 `runtime.go` 一处，不要再往 `app.go` 加分叉。服务端 5xx 的原因只写日志，响应只回通用状态文案：handler 里数据库/库错误一律经 `handlers.dbError`/`internalError`（唯一键冲突回 409 `already exists`，外键回 409，其余 500 `internal error`），不得再写 `c.Status(..).JSON(fiber.Map{"message": err.Error()})`；只有 `checkPasswordPolicy`、批量 id 校验这类自己构造的业务文案才可回显。`app.OpenDatabase` 开启 `TranslateError`；调用方自带 `*gorm.DB` 时 `dbError` 也会按驱动错误码兜底识别。
- 需要显式生命周期的二开使用`app.NewRuntime(app.RuntimeOptions)`：传入公开`app.Config`与调用方拥有的GORM连接，`Listen/Close`不接管信号，构造时不迁移/seed/发现API/创建日志目录。`app.MigrateSchema`单独执行且不创建默认用户。`DisableMedia`禁内置媒体路由，`PublicUploads`必须显式开启才静态公开上传目录。
- 显式runtime沿用底座全局配置，仅支持每进程一个实例且不得与Run混用；Close超时后保留实例锁至进程退出，不能在旧请求仍运行时复用配置。DB连接始终由调用方关闭。
- 二开权限通过 `ExtraPermissions` / `ExtraPermissionExpand` 注入，业务代码与 submodule 同级独立维护，不直接二改底座。
- 前端通过 `frontend/src/core.tsx` 的 `createApp(opts)` 注入 `extraResources`、`extraRoutes`、`extraRouteLabels`，业务页面留在二开项目自己的 `frontend/src/pages/`。
- `createApp({i18n:{languages,messages,defaultLocale}})`可选扩展语言列表和文案，不配置仍为原中英。`messages[locale]['login.title']`覆盖登录/侧栏品牌；`routes.<完整路径>`覆盖侧栏/页签标题。动态菜单仍以服务端授权为准，翻译不新增菜单权限；业务页使用同一useI18n上下文。
- 仪表盘：`createApp({dashboard:{widgets,hide,page}, homePath})` 增删/覆盖卡片或整页替换；widget 的 `permission` 只控制显示，数据接口须自行鉴权；后端数字经 `app.Options.DashboardStats` 合并（同名键覆盖内置）。
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
| 仪表盘定制 | `frontend/src/dashboard/`、`createApp({dashboard,homePath})`、`app.Options.DashboardStats` |
| 配置及默认值 | `config.example.yaml`、`backend/internal/config/config.go` |

表名和端点以模型、路由为准，不在规则中维护第二份全量目录；目录级代码地图见 `docs/CODEMAP.md`，入口、路由、模型、配置或核心流程变化时同步更新。

## 权限维护

- 角色接口权限定义来自代码，不从 `apis` 表读取：内置定义和展开在 `backend/internal/models/permissions.go`，`AllPermissionDefs()` 合并内置与扩展定义。
- key 使用 `资源:操作`，由 `ExpandPermission()` 或二开 `ExtraPermissionExpand` 展开为路径和方法的 Casbin 策略。新增权限须同时维护定义和展开；删除路由时同步移除引用它的权限。
- `backend/internal/handlers/crud.go` 提供角色权限读取/保存；前端 `frontend/src/pages/roles/index.tsx` 通过 `GET /api/v1/roles/:slug/permissions` 拉取分组定义，通过 PUT 同路径提交 `{keys: [...]}`，后端展开策略。
- 底座新增定义不需要另改数据库 seed、前端组件或中间件；二开只使用公开注入点，不改底座 `permissions.go`。
- `AutoDiscoverAPIs()` 在路由注册后同步 API 登记表（跳过 HEAD/OPTIONS；仅删除 `source=auto` 且路由已不存在的行，UI 手建的记录不动）；登记/自动发现不替代权限定义与展开，也不代表新增路由已获得角色授权。
- 菜单授权只控制侧栏可见性，不能替代接口鉴权。权限弹窗按组展示 checkbox，组全选与下级联动。

## 认证与安全边界

- 显式runtime可选`Config.Session.Cookie=true`，需单一HTTPS `Session.Origin`（同源校验以它为准，不再与CORS配置比较）；前端`createApp({sessionMode:'cookie',disabledResources:['media']})`配套。旧Run/serve保留Bearer并拒绝cookie配置，避免隐式默认账号/公开uploads进入严格模式。
- Cookie模式使用Secure/HttpOnly会话与CSRF，完整会话8h，待MFA会话5min且只允许精确verify；未注册MFA只开放注册必需端点。业务变更要求5分钟内密码+TOTP重认证，锁屏unlock不延长该授权。仅显式信任loopback代理时读X-BIM-Client-IP，代理必须覆盖该头。
- 严格模式回归：`go test -race ./...`和`go vet ./...`；`NX_TEST_DATABASE_DSN`启用隔离schema Cookie/MFA PostgreSQL测试，`NX_TEST_POSTGRES_DSN`启用只读启动测试。浏览器测试专用`NX_BROWSER_QA_ADDR`仅用于本地隔离test进程，不是运行配置。
- `app.BootstrapAdmin(db,username,password)`支持PostgreSQL/MySQL(MariaDB)/SQLite，要求已显式迁移且管理种子表为空；PG表锁、MySQL命名锁、SQLite进程内互斥+写锁保证并发仅一次；密码由调用方传入（8～32字节），不生成默认密码，重复执行返回`ErrAdminAlreadyInitialized`且不修改已有账号。初始化账号尚未MFA注册，须经严格模式注册后才能执行业务。

- JWT 有效期 24h；`JWTAuth` 校验 token 对应的 `admin_online_users` 会话，MFA 验证路径不做在线会话检查。修改认证前先核对 middleware 和路由例外，不以菜单可见性代替认证。
- Casbin 模型为 `g(r.sub, p.sub) && anchoredMatch(r.obj, p.obj) && anchoredMatch(r.act, p.act)`：整串锚定匹配（`^(?:pattern)$`），非法正则不匹配而非 panic；Admin 种子为 `admin, .*, .*`。权限展开里的路径参数写 `[^/]+`（单段），不得再写 `.*`；旧库的 `/x/.*` 策略在迁移时自动改写。免权限/免令牌路由在 `middleware/routes.go` 用精确 `METHOD /path` 维护，禁止 `HasSuffix`/`Contains` 式匹配，不擅自扩大放行范围。Fiber 已开 `CaseSensitive`+`StrictRouting`。
- 登录先校验密码再判账号状态：未知用户、错误密码、被禁用账号返回同一 401，且都计入限流（无枚举）。Bearer 模式 MFA 验证按用户计失败次数并锁定；恢复码在事务内消费。删除角色须无用户使用（409），并同步清 `role_menus` 与 Casbin 策略；角色 slug 创建后不可改。菜单父级不得成环，删除菜单清其 `role_menus`。`ensureSystemMenus` 只在缺失时创建，不覆盖运维改动。
- 角色权限只能通过 `PUT /roles/:slug/permissions` 提交 `{keys}` 保存；不接受原始策略，拒绝系统角色、不存在的角色和未知 key。
- 审计异步写入须保持上下文隔离，避免复用请求 ctx。
- 手写 SQL 须三库（SQLite/MySQL/PostgreSQL）通用：保留字列（`key`、`group`）用 `clause.Column`/`clause.Eq` 让 GORM 引用，不手写反引号或裸列名；不用 `||` 拼接做条件，在 Go 里比较；文本包含筛选底座内用 `dbutil.ContainsFold`（大小写不敏感且转义 `%`/`_`）。`app/dialect_test.go` 在三库跑这些路径，`NX_TEST_MYSQL_DSN`（go-sql-driver 形式）与 `NX_TEST_DATABASE_DSN` 启用后两者。二开项目同样适用：文本包含筛选用公开的 `app.ContainsFold(q, column, term)`，勿直接写 `LIKE`；`column` 须为代码里的标识符，不得来自请求。
- MFA 使用 TOTP：登录返回 `mfa_required` 后以六位码或恢复码换取正式 JWT；用户菜单提供设置，管理员用户管理入口可清空 MFA。
- 上传：扩展名走白名单（`upload.allow_types`，默认仅无害格式；html/svg/js 等主动内容永远拒绝），按文件内容嗅探 MIME 且不信任客户端 `Content-Type`，图片/PDF 扩展名与内容必须一致；`/uploads` 响应带 `nosniff`、sandbox CSP，非图片/音视频/PDF 强制 `attachment`。
- 用户可管理自己的登录 IP 白名单，管理员可管理其他用户；密码通过后匹配，无规则默认放行；规则可为单个 IP 或 CIDR，更新/删除按 URL 中的用户校验归属。`NX_LOGIN_IP_WHITELIST_ENABLED=false` 可关闭开关，不能把调试关闭作为默认安全状态。
- 现有 IP 解析涉及 `X-Forwarded-For` / `X-Real-IP` / `CF-Connecting-IP`；geo 使用 ip-api，800ms 超时且失败留空。改动前核对实际代理信任边界，不把 header 或 geo 当作已验证身份。
- 每次登录建立独立会话，注销只清当前 token；只能踢非当前会话。
- 删除自身、删除系统角色、踢当前会话等危险动作必须前后端双重保护：按钮禁用，后端明确拒绝，不能只靠 UI。

## 前端规范

- CRUD 标题左、工具右；筛选用白色卡片，重置/查询靠右；分页左侧总数、右侧页码。日志必须后端分页筛选，禁止固定最近 N 条。列表页六区、弹窗与独立页的新建/编辑、删除确认、检查清单见 `docs/20261020-0212-crud-page-conventions.md`，新增 CRUD 页前先读。
- 日期时间统一用 `frontend/src/components/ui/datetime-input.tsx` 的 `DateTimeInput`；flatpickr 全局样式及 Tailwind 污染回退沿用现有入口，不在页面重复实现。
- 基座要保持视觉统一：页面和弹窗只用 `frontend/src/components/ui/` 的公共组件，不再手写原生 `<select>`、`<input type=checkbox>`、`window.confirm`、行内错误 `<p>`、状态标签 `<span>`。对应关系：`Select`、`Checkbox`、`ConfirmDialog`、`Alert`、`Badge`；表单行 `FormField`，筛选区 `FilterBar`/`FilterField`，页头 `PageHeader`，行内图标按钮 `IconButton`，小表格 `SimpleTable*`，列表 `DataTable`，加载/空/骨架 `Spinner`/`EmptyState`/`Skeleton`，轻提示 `useToast`，悬浮提示 `Tooltip`，单选 `RadioGroup`，页内标签 `Tabs`。缺组件时先在 `ui/` 加封装再用，不在页面里临时拼样式。
- 布尔字段用 `Switch`（开关在左，label+说明在右）；列设置在 `DataTable` 表头末列，批量操作条在表格卡片内，页面不再单独放“列设置”工具栏。宽表用 `DataTable` 的 `stickyStart`/`stickyEnd` 让前/后几列粘住两侧（含操作列的表默认 `stickyEnd={1}`），不自己写 sticky。
- 不恢复暗黑皮肤：无 `dark:` 类、ThemeProvider 或切换入口。
- 在线详情用弹窗表格，不在主表格内展开。

## 配置与运行

- 运行配置用根目录 `config.yaml`，模板为 `config.example.yaml`；环境变量覆盖对应配置。数据库支持 SQLite/MySQL/Postgres，默认 SQLite 路径 `data/nx.db` 相对工作目录。
- `serve` 会自动迁移、建表及写种子，不能当只读健康检查。`jwt.secret` 留空时首次启动自动生成 32 字节随机密钥写入 `jwt.secret_file`（默认 `data/jwt.secret`，0600），之后一直复用（它是运行状态，与 `data/nx.db` 同属需备份的数据；**运维必须备份 `data/jwt.secret`**，丢失会重新生成并使全部登录失效；不回写 `config.yaml`，因其常为只读/受审计）；显式值（配置或 `NX_JWT_SECRET`）优先，多实例须共用同一显式值，须≥32字节，已发布的默认值视为未设置。应用**不挂** Fiber 的 CORS 中间件、不输出任何 `Access-Control-*` 头（避免与 nginx 叠加出 `*, *` 这类重复值），CORS 一律由反向代理负责；`cors.origins` 仅为兼容旧配置保留，配置了会在启动日志警告“已忽略”。`Run` 启动先创建 `data`、`logs`、上传目录、自定义 SQLite 路径与密钥文件的父目录（`app.RequiredDirs`，`reset-password` 同样先建目录），全新部署无需预建目录。`config.Validate()` 为 `Run` 与 `NewRuntime` 共用；`NewRuntime` 不写文件，要同样的“生成一次”行为的二开项目调用 `app.EnsureJWTSecret(path)`。`config.Load` 返回 error（YAML/环境变量格式错误不再静默回落默认值，布尔变量用 ParseBool）。首次种子管理员 `admin` 使用随机初始密码，仅在启动日志打印一次且 `must_change_password=true`；`reset-password` 不带密码时同样随机生成并强制改密、踢掉其在线会话。
- 后端 `server.host/port` 或 `NX_SERVER_HOST/PORT` 控制监听，默认 `127.0.0.1:19500`（本机验收可设 `NX_SERVER_HOST=172.18.0.1`）；健康入口 `/health`。
- Vite 默认 `127.0.0.1:5173`，`VITE_HOST` 覆盖 host，`VITE_API_TARGET` 覆盖 `/api` 代理目标（默认 `http://127.0.0.1:19500`）。`backend/cmd/frontend/dist/.gitkeep` 与 `frontend/dist/.gitkeep` 入库，保证新克隆不先构建前端也能 `go build`；`make build-frontend` 用 `npm ci`，`Makefile` 不再执行 `go mod tidy`。运行前读 manifest/lock、Makefile，避免照搬二开项目命令。
- 常用入口：`make build-frontend` 构建并复制 embed 资源，`make dev-frontend` 热更新，`make build` 完整构建，`./nx-admin serve` 启动。密码重置命令会写数据，仅在授权内使用。
- 测试监听约定（本机验收用，开发者在浏览器里看效果）：开始改动前代理主动拉起并告知访问地址；push 完成后再关闭。后端在 `backend/` 下 `NX_SERVER_HOST=172.18.0.1 go run ./cmd serve`（`172.18.0.1:19500`，会建表写种子，初始 admin 密码见启动日志）；前端在 `frontend/` 下 `VITE_HOST=172.18.0.1 npx vite --port 8001 --strictPort`（5173 常被其他项目占用，不要抢）。关闭按端口查进程号后精确 `kill <pid>`，不用 `pkill -f`（会匹配到自身命令行）。只在用户明确要求停时提前关闭；运行日志、数据库和构建产物写到仓库外或已忽略路径，不进 git。

## Submodule 构建

- 底座自身 `make build` 将前端嵌入 `backend/cmd/frontend/dist/`，生成 `nx-admin`；`make clean` 会清构建产物。
- 二开在自己的项目根目录构建，入口为 `cmd/main.go`，不是 submodule 的 `backend/cmd/main.go`；二开前端产物进入 `cmd/frontend/dist/` 并由 `go:embed all:frontend/dist` 嵌入。
- `scaffold.sh` 生成 go.mod replace、公开 app/createApp 注入点、前端骨架、Makefile 和配置模板；使用前读脚本及生成项目的 Makefile，不在此维护重复创建教程。
- 业务 API 可用独立 `cmd/api/main.go` 构建独立二进制与端口，不与管理后台入口混用。
- 升级底座使用项目既有升级入口并重新构建，核对 submodule 锁定版本及改动；不要把可漂移远端更新当成已验证版本。
