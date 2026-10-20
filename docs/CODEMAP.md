# CODEMAP

go-nx-admin 代码地图。改代码前先读这里,再打开具体文件核对;入口、路由、模型、配置、核心流程变化时同步更新本文件。跨项目规则与安全边界见根目录 `AGENTS.md`,本文件只回答"东西在哪、怎么连"。

## 运行与验证

- 后端:`cd backend && go build ./... && go vet ./... && go test -race ./...`
- 前端:`cd frontend && npx tsc --noEmit -p . && npx vite build`
- 完整构建:`make build`(前端嵌入 `backend/cmd/frontend/dist/`,产物 `nx-admin`);启动 `./nx-admin serve`(会迁移并写种子)。
- 可选数据库回归:PostgreSQL `NX_TEST_DATABASE_DSN` / `NX_TEST_POSTGRES_DSN`,MySQL/MariaDB `NX_TEST_MYSQL_DSN`(go-sql-driver 格式,如 `root@unix(/run/mysqld/mysqld.sock)/?parseTime=true`);SQLite 始终运行。

## 后端

### 公开入口 `backend/app/`
| 文件 | 内容 |
|---|---|
| `app.go` | `Options`(`EmbedFS`/`ExtraModels`/`ExtraRoutes`/`AfterMigrate`/`ExtraPermissions`/`ExtraPermissionExpand`/**`DashboardStats`**)与 `Run`(`log.Fatal`)/`RunE`(返回 error,委托 `NewRuntime`,SIGINT/SIGTERM 关停并释放缓存、连接池、日志文件);`app.Cache`、`app.Storage` 及 `ErrCacheMiss`/`ErrStorageNotFound` 别名 |
| `runtime.go` | `NewRuntime(RuntimeOptions)`:显式生命周期,也是 `Run`/`RunE` 的 HTTP 栈(`errorHandler`、`/health`、日志、静态页只在此一处);`rollbackGlobals` 回滚全局(权限扩展、`handlers.ExtraDashboardStats`、cache、storage),新增进程级全局必须纳入这里 |
| `secret.go` | `EnsureJWTSecret(path)`:给 `NewRuntime` 调用方的“生成一次并复用”密钥辅助 |
| `sqlutil.go` | 公开 `ContainsFold`/`EscapeLike`(委托 `internal/dbutil`):大小写不敏感、转义 `%`/`_` 的包含筛选,三库一致 |
| `database.go` | `OpenDatabase(cfg)`:`Run` 与 `reset-password` 共用的连库与连接池;SQLite 自动补 `_busy_timeout=5000`+WAL(`_pragma` 自管则不动),池上限 8(**不能设 1**:Casbin 适配器在事务内再用外层句柄会死锁) |
| `bootstrap.go` | `BootstrapAdmin`(PostgreSQL/MySQL/SQLite) |
| `uploads.go` | `/uploads` 挂载与安全响应头;`RequiredDirs` 在 `app.go`,启动先建 data/logs/uploads/SQLite 与密钥文件父目录 |
| `actor.go` | `CurrentActor(c)`：仅在底座认证和授权后的业务路由读取当前主体 `Actor{ID,Username,Role,RoleID}`；缺失、零 ID 或错误类型返回 `false`，不从请求输入取身份 |
| `static.go` | 前端静态服务;每次请求重渲染 index.html 注入 `site.title`/favicon,`/_site/favicon` 提供本地 favicon;`assets/favicon.svg` 为内置牛图标 |
| `*_test.go` | `bearer_test`/`browser_test`(Cookie+MFA)/`batch_test`/`password_test`(改密、强制改密、客户端 IP、仪表盘合并、`/auth/permissions`)/`media_test`/`static_test`/`runtime_*` |

### `backend/internal/`
| 包 | 职责 |
|---|---|
| `config/config.go`、`config/validate.go` | 配置结构与默认值、环境变量覆盖(`NX_*`,格式错误返回 error);`Config.Validate()` 由 `Run` 与 `NewRuntime` 共用(JWT 密钥≥32 字节且非已发布默认);`Warnings()` 报告被忽略的废弃项(`cors.origins`);`config/secret.go` 的 `ResolveJWTSecret` 在配置/`NX_JWT_SECRET` 均未设置时自动生成并持久化到 `data/jwt.secret`(0600,不回写 config.yaml;**运维需备份该文件**,丢失则重新生成、所有登录失效);含 `server.trusted_proxies`/`client_ip_header`、`jwt.access_minutes/refresh_minutes/mfa_minutes`、`site`、`cache`、`upload.driver/s3` |
| `cache/` | `Cache` 接口(Get/Set/GetDel/Del/Incr/TTL)、`ErrMiss`;`memory.go` 默认,`redis.go` 可选;`default.go` 为进程默认实例(`cache.Init`)。登录限流、验证码等共用 |
| `storage/` | `Storage` 接口(Put/Open/Delete/URL);`local.go`、`s3.go`(minio-go,兼容 S3/MinIO/OSS/COS/R2);`registry.go` 按记录的 `Storage` 列路由,切换驱动后旧文件仍可读 |
| `routes/routes.go` | 全部 `/api/v1` 路由;`api` 组统一挂 `JWTAuth` + `CasbinMiddleware`;`RegisterWithOptions` 支持 `disableMedia` |
| `middleware/` | `jwt.go`(令牌,MFA 临时令牌 `PurposeMFA` 仅可访问 `/mfa/verify`)、`auth.go`(会话校验、`must_change_password` 路由白名单,否则 403 `PASSWORD_CHANGE_REQUIRED`)、`casbin.go`(授权;`anchoredMatch` 整串锚定匹配、`UpgradeWildcardPolicies` 迁移旧 `.*` 策略)、`routes.go`(免令牌/免权限路由的精确 `METHOD /path` 集合)、`browser.go`(Cookie 会话/CSRF/重认证)、`ratelimit.go`、`clientip.go` |
| `middleware/audit.go` | 审计行的构造辅助:`auditStatus` 按错误处理器会给出的状态记录(handler 返回 error 时响应状态此刻仍是 200),`auditErrorText` 截到 255 字节并保持合法 UTF-8、去控制字符——MySQL/PG 对超长列直接拒写整行,会丢审计 |
| `middleware/clientip.go` | **客户端 IP 唯一来源** `ClientIP`:对端不在 `trusted_proxies` 时忽略一切头;信任时按 `client_ip_header` 取值,XFF 从右向左跳过受信代理;Cookie 模式沿用 `X-BIM-Client-IP` |
| `models/models.go` | 用户/角色/菜单/API/日志/在线用户模型与种子;`User.MustChangePassword`;首次种子管理员 `admin` 随机初始密码(日志只打一次)并强制改密 |
| `models/permissions.go` | 权限定义 `PermissionDefs` + `ExpandPermission`;`AllPermissionDefs`、`GetGrantedPermissionKeys` |
| `models/session.go`、`media.go`、`config.go` | 浏览器会话、媒体(含 `Storage` 列)、系统配置 |

### `backend/internal/handlers/`
| 文件 | 内容 |
|---|---|
| `auth.go` | 登录(先验密码再判状态,统一 401 并计限流)、`Me`、MFA(按用户限次、恢复码事务消费)、登出、IP 白名单检查(单 IP/CIDR) |
| `refresh.go` | `POST /auth/refresh`:不透明 refresh token 原地轮换;10s 宽限(多标签竞态),宽限外重放则吊销会话 |
| `crud.go` 的 `UpdateUser` | 用户名创建后不可变:更新接口忽略请求里的 `username`,前端编辑框为 `readOnly` |
| `password.go` | `POST /auth/password`:旧密码(+MFA 开启时 TOTP)、`checkPasswordPolicy`(8–32 字节且不同于旧密码)、登录限流、**不踢会话**、成功后清 `must_change_password` |
| `batch.go` | 批量删除/改状态:逐 id 结果(`deleted`/`updated`/`failed[]`),自身与系统角色跳过并给原因,不是全成或全败 |
| `crud.go` | 用户/角色/菜单/API/日志/在线用户 CRUD;`ListRolePermissions`、`MyMenus`、**`MyPermissions`**(仅返回调用者自己角色的权限键;`all:true` 表示超级角色) |
| `dashboard.go` | `GET /dashboard/stats`;`ExtraDashboardStats`(由 app 层注入)的返回键合并覆盖内置键,扩展 panic 被隔离不影响内置数字 |
| `media.go` | 上传与媒体管理(走 storage 抽象) |
| `browser_auth.go`、`browser_enrollment.go` | Cookie 模式登录与 MFA 注册 |
| `errors.go` | `dbError`/`internalError`:handler 的统一出错出口,原因写日志、响应只回通用文案;唯一键/外键冲突回 409 |
| `captcha*.go`、`config.go` | 滑块验证码(资源首次使用时懒加载,失败只让该请求 500,不再在包 `init()` 里终止进程)、系统配置 |

### 关键流程
- **认证**:login → (mfa_required → `/mfa/verify`) → access 30min + refresh 7d → 到期前 `/auth/refresh`。
- **强制改密**:`must_change_password=true` 的账号只能访问白名单路由;前端 `PasswordGate` 弹不可关闭的改密对话框;用户创建/编辑表单可设置该标记(编辑时 nil 保持原值,显式 false 清除)。
- **权限**:定义在 `permissions.go`,策略行存为 `(role, path, method)`;`AutoDiscoverAPIs` 只登记 API,不授予权限。升级后需重新保存角色权限才能获得新增路径。
- **仪表盘数据**:内置 4 个数字 + 二开 `DashboardStats` 合并。

## 前端 `frontend/src/`

| 路径 | 内容 |
|---|---|
| `core.tsx` | `createApp(opts)` 唯一二开入口。选项:`extraResources`/`extraRoutes`/`extraRouteLabels`/`sessionMode`/`disabledResources`/`i18n`/`keepAliveExclude`/**`dashboard`**/**`homePath`**;`buildRoutes` 装配所有页面 |
| `dashboard/` | **仪表盘定制**,见下节 |
| `components/ui/` | 公共视觉组件(基座统一入口):`button`(`asChild`)、`input`、`textarea`、`select`(Radix,自绘下拉)、`checkbox`、`radio-group`、`switch`、`datetime-input`;`form-field`、`filter-bar`;`dialog`、`confirm-dialog`、`popover`、`dropdown-menu`、`tooltip`、`toast`(`ToastProvider` 在 `createApp` 内,`useToast`);`alert`、`badge`、`spinner`、`empty-state`、`skeleton`、`page-header`、`icon-button`、`tabs`;`data-table`(列设置在表头末列、批量条在卡片内)、`table`(`SimpleTable*`) |
| `components/layout/` | `index.tsx` 外壳(顶栏单行:折叠按钮 + 页签 + 右侧工具区,无面包屑);`route-labels.ts`(`RouteLabelsContext`);`sidebar.tsx`(直达已授权叶子路由时登记页签以保留 pane,随当前路由展开祖先并高亮,未注册路径如 `/users/5` 不定位菜单;`/auth/menus` 返回前显示骨架,不先画基座菜单);`tabs.tsx`(位于顶栏内,`flex-1 min-w-0` 横向滚动,不挤右侧工具区;横向溢出不撑高;首页标签无关闭按钮、不可固定,「关闭其他/全部」「关闭左/右」都保留它);`keep-alive.tsx`(每个标签一个 pane、`useTabActive`/`useOnTabActivate`、刷新仅重挂本 pane、`TAB_REFRESH_EVENT`);`user-menu.tsx`(含「安全设置」);`lock-screen.tsx`、`global-tools.tsx` |
| `components/ui/switch.tsx` | 公共开关 `Switch`(`checked`/`onCheckedChange`/`label`/`description`/`disabled`,`role=switch`)。只有开/关两态的字段一律用它,不用下拉框或裸复选框;多选列表(如角色授权)仍用复选框 |
| `components/ui/data-table.tsx` | 通用表格:服务端分页、可选排序、行选择(跨页保留,跳过禁用行,表头支持半选)、批量操作条(在表格卡片顶部,选中才出现)、列设置(图标在表头末列,`storageKey` 持久化)、`toolbar` 扩展按钮区(表格上方右侧)、`stickyStart`/`stickyEnd` 让首/尾 N 个可见数据列粘住表格两侧(sticky,测量表头宽度算偏移,选择列随首侧粘住,边缘出阴影) |
| `components/security/` | `password-form.tsx`(改密表单)、`force-password-change.tsx`(`PasswordGate`) |
| `hooks/use-batch.tsx` | 批量操作确认框与逐 id 失败汇总;用户/角色/API/媒体页复用 |
| `pages/` | `users`/`roles`/`menus`/`apis`/`audit-logs`/`login-logs`/`media`/`system-config`、`login`、`mfa-verify`、`enrollment` |
| **CRUD 页面规范** | 列表页六区(标题/搜索/批量/数据/分页/扩展工具)、弹窗与独立页的新建编辑、删除确认、文案与检查清单:`docs/20261020-0212-crud-page-conventions.md`;现有页面对照表在该文末尾 |
| `contexts/` | `i18n-context`(五语言,`messages/*.ts` 按模块拆分)、`tabs-context`(`withHome` 保证首页标签始终存在、排首位、不可关闭,会修复本地存储里过期的标签)、`lock-screen-context` |
| `providers/`、`lib/` | `auth-provider`、`data-provider`;`api.ts`(含 401 后 refresh 重试)、`session.ts`(令牌/Cookie 会话,登录/登出时清权限缓存) |

### 仪表盘定制 `frontend/src/dashboard/`
| 文件 | 内容 |
|---|---|
| `registry.ts` | `DashboardWidget`/`DashboardOptions` 类型、`resolveWidgets`(内置→二开按 key 覆盖→`hide` 剔除→按 `order` 稳定排序)、`DEFAULT_HOME_PATH` |
| `builtin.tsx` | 4 张内置卡片:`builtin.total-users`(10)、`builtin.online-users`(20)、`builtin.total-roles`(30)、`builtin.today-logins`(40);共用页面级 `StatsContext`,只发一次请求 |
| `widgets.tsx` | 二开可复用积木:`StatCard`、`DashboardCard`、`useDashboardData(path)`(标签重新激活自动刷新)、`WidgetState` |
| `page.tsx` | `DashboardPage`:网格渲染 widgets,过滤无权限的 widget;`options.page` 存在时整页替换 |
| `permissions.ts` | `useMyPermissions`(请求 `/auth/permissions`,进程内缓存),`resetPermissions` |
| `context.tsx` | `DashboardOptionsContext`、`HomePathContext`、`useHomePath()`(登录落地、MFA 后跳转、固定首页标签、根路径重定向) |

二开定制路径:
1. 加卡片/隐藏内置卡片 → `createApp({dashboard:{widgets,hide}})`,key 与内置相同即覆盖。
2. 整页重做 → `createApp({dashboard:{page}})`。
3. 新数字 → 后端 `app.Options.DashboardStats` 合并键,前端用 `useDashboardData('/dashboard/stats')` 读取。
4. 落地页 → `createApp({homePath})`。
- widget 的 `permission` 只决定显示;数据接口仍须自己鉴权。
- `scaffold.sh` 生成的 `cmd/main.go` 与 `frontend/src/main.tsx` 已带注释示例。

## 配置 `config.example.yaml`
`server`(含代理信任)/`database`/`jwt`/`site`/`cors`/`session`/`login`/`upload`(local|s3)/`captcha`/`cache`(memory|redis)。`site.title`/`site.favicon` 修改后无需重建前端。

## 脚手架 `scaffold.sh`
生成二开项目的 `go.mod` replace、`cmd/main.go`、Makefile、前端骨架(`createApp` 注入点注释)。改动 `app.Options` 或 `CreateAppOptions` 时同步其注释示例。

## 已知未覆盖(有意搁置)
密码过期/复杂度策略、登录设备管理、部门/岗位、数据范围权限、字典、多角色用户、导入导出、通知、定时任务、监控、SSO/OAuth2/LDAP、邮件/短信、WebSocket/SSE、Dockerfile/CI、版本化迁移、结构化日志/指标、多租户、字段级审计。
