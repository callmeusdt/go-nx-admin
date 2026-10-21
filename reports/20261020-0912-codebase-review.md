# go-nx-admin 全量代码审查(只读,未改代码)

范围:后端 handlers/models/routes、app/middleware/config/cache/storage/cmd/scaffold/Makefile、前端 src 全部。共约 1.65 万行。
方法:三路并行审查,关键结论已由主审抽查复核(标「已复核」)。未标注者为子代理读码结论,修前建议先复现。

## 总体判断

功能面比预期完整(MFA、Cookie 会话、S3、批量接口、i18n 都有),且 `go vet`、`go test -race`、`tsc` 都过。问题集中在三类:
1. 权限与鉴权边界靠字符串/未锚定正则拼出来,存在真实越权。
2. 只验证了 SQLite 和部分 PostgreSQL,MySQL/PG 的方言问题没人跑过。
3. 对二开的支持停留在“能 import app.Run”,缺稳定公开 API,且有两套行为分叉的入口。

## P0 安全 / 数据损坏

| # | 问题 | 位置 | 修法 |
|---|---|---|---|
| 1 | Casbin `regexMatch` 未锚定,权限被前缀放宽。已复核:`/api/v1/auth/me` 匹配 `/auth/menus`;`roles:edit` 的 `PUT /roles/.*` 也匹配 `/roles/x/permissions`,持有「编辑角色」即可给自己提权。`roles:list` 能读任意角色权限,`users:list` 能读 IP 白名单 | `middleware/casbin.go:30`、`models/permissions.go` | 展开时统一生成 `^…$` 并转义;或改 `keyMatch2` |
| 2 | `/mfa/` 路径整段绕过 Casbin 和在线会话校验(`strings.Contains`),Fiber 忽略末尾斜杠,`DELETE /users/:id/mfa/` 可被低权限用户命中,清除任意用户 MFA。已复核白名单写法,斜杠利用为子代理实测 | `casbin.go:136`、`auth.go:35` | 改为 `method+精确路径`集中白名单;Fiber 开 `StrictRouting`/`CaseSensitive` |
| 3 | 白名单用 `HasSuffix("/login")`、`/captcha` 等,二开路由名撞上就被当公开路由;大小写不敏感路由配合 `HasPrefix(path,"/api/")` 可绕过 Casbin | `auth.go:16`、`casbin.go:129-136` | 同上,并提供 `Options.PublicPaths` |
| 4 | `SaveRolePolicies` 接受请求体原始 `policies`:无校验,可写 `.*`,可改 `admin` 角色把自己锁死,非法正则会让 casbin panic 使该角色永久 500;错误被忽略 | `handlers/crud.go:657` | 只接受 `keys`;拒绝系统角色;校验 slug |
| 5 | 上传无类型/大小白名单,种子里的 `upload_allow_types` 没有任何代码读取;`.html/.svg` 从同源 `/uploads` 提供,可存储型 XSS 偷 token。`Run()` 还无条件公开 uploads(与 AGENTS 说的“必须显式开启”相反) | `handlers/media.go:55-81`、`app.go:147` | 白名单 + 嗅探 + `attachment`/独立域 |
| 6 | 默认 JWT 密钥 `nx-admin-jwt-secret-2024` 写进代码和示例配置;`Run()` 不校验;`jwtSecret()` 在配置为空时还静默回落到它。伪造 token 仍需在线会话记录,实际危害待确认,但属公开弱密钥。已复核 | `config.go:148`、`jwt.go:22` | `Validate()` 两个入口共用,默认值拒绝启动 |
| 7 | 种子账号 `admin/qwer1234` 不强制改密;`reset-password` 默认 `admin123` 并明文打印;`scaffold.sh` 结尾提示的账号密码与实际种子不一致 | `models.go:168`、`reset_password.go` | 首次启动随机密码或强制改密 |
| 8 | CORS 默认 `*`,`Run()` 不校验 | `config.go:149`、`app.go:134` | 同 6 |
| 9 | 菜单可成环,`DeleteMenu` 递归无环检测,栈溢出是 fatal,`recover` 接不住,进程直接退出 | `crud.go:288,301` | 更新时拒绝环,递归加深度/visited |

## P1 真实 bug / 明显反模式

跨库(只测过 SQLite+部分 PG):
- `MyMenus` 非超管角色 Join 后 `Order("id asc")` 列名歧义,SQLite 下报错(子代理实测)→ 非超管用户菜单 500。`crud.go:511`。已复核代码存在该写法。
- 反引号 `` `group` `` 在 PG 失败(`crud.go:364`);`key = ?` 在 MySQL 是保留字,系统配置读写静默失败且错误被吞(`models/config.go:30`、`models.go:286`);`path||'|'||method` 在 MySQL 是 OR,过期 API 清理失效(`models.go:344`)。已复核 `key = ?`。
- LIKE 大小写在 PG 敏感,MySQL/SQLite 不敏感,`%` 不转义。
- SQLite `MaxOpenConns(100)`,无 busy_timeout/WAL,易 `database is locked`。

一致性与并发:
- Casbin 用非同步 `Enforcer`,运行时改策略与请求并发有 map 竞态;`SavePolicy` 全表清空重写,中途失败内存与库不一致;多实例部署权限变更不生效。
- 删除/禁用逻辑分叉:`DeleteUser` 不清会话、不拦系统管理员且吞错误,批量版有;`UpdateUser` 未带 `role_id` 会写成 0;禁用在 Bearer 模式不踢会话;`DeleteRole` 不检查在用、不清策略和 `role_menus`,新建同 slug 角色会继承旧权限。
- `Seed` 以角色数判断,非原子;`ensureSystemMenus` 每次启动覆盖用户对系统菜单的改动并复活被删菜单,找不到「后台管理」就整体 return,跳过后续种子。
- `AutoMigrate` 里 `DropIndex` 错误被忽略;`ExtraModels` 先于底座表迁移,外键引用 `admin_users` 的二开模型全新库会失败。
- 审计日志每请求 `go func()` 无上限,退出不 flush;写入嵌在 Casbin 中间件里。
- IP 白名单更新/删除忽略路径里的用户 id(IDOR);`Enabled default:true` 的零值陷阱导致传 false 被改成 true;无 CIDR。

认证细节:
- MFA 验证无次数限制,6 位码在临时令牌有效期内可暴力枚举;`MFASetup` 已启用时可直接覆盖 secret;恢复码消费不在事务内可并发复用。
- 登录先判禁用再验密码(枚举);用户不存在不计限流;验证码依赖客户端 `fingerprint`;`resolveIPGeo` 同步阻塞登录最多 800ms 并明文 HTTP 外发用户 IP。
- Fiber 默认 `BodyLimit` 4MB,配置的 10MB 上传限制实际无效。

入口:
- `Run()` 与 `NewRuntime` 两套大量重复且行为分叉:`Run` 的错误处理器回显 `err.Error()`(可泄露 SQL,已复核),CORS 头、`/health`、`InitCasbin` vs `LoadCasbin` 都不同。
- `Run()`/`captcha_resources.go` 的 `init()` 里 `log.Fatal`;优雅退出丢弃 Shutdown 错误且不关库、日志、缓存。
- 配置加载吞 YAML 解析错误(写错静默回落默认含默认密钥),`strconv.Atoi` 错误也吞;`NX_LOGIN_IP_WHITELIST_ENABLED` 只认精确 `"true"`,`1`/`True` 会静默关闭安全开关。

前端:
- 401 整页 `window.location.href='/login'`,硬编码 `/login`、`/api/v1`,二开部署在子路径会错。
- `auth-provider.check()` 失败只清部分 key,权限缓存、tabs 留给下一位登录者。
- 非 2xx 一律丢弃服务端返回体,校验错误/403 原因用户看不到。
- 列表请求无竞态保护;多处 `useEffect` 缺依赖;没有 `eslint-plugin-react-hooks` 所以没被发现。
- `tabs-context` 的 `JSON.parse` 无 try/catch,localStorage 损坏白屏。
- token(含 refresh)存 localStorage;`RichEditor` 链接未做协议白名单。

## P2 规范与可维护性

- `crud.go` 731 行混 7 类职责,`ParseUint` 重复十余次;响应格式混乱(`{message}`/`{code}`/裸 `SendStatus`、中英混杂、验证码冷却返回 200)。
- 菜单权限名与权限定义不一致(`audit-logs:list` vs `audit:view` 等);`InitCasbin` 硬编码不存在的 `editor` 角色且路径错误;每次启动 `SavePolicy` 重写整表。
- `ListUsers`/`ListOnlineUsers` 无分页;`ListMedia` 的 `page_size` 可为负或超大;`OnlineUser.Token`、MFA secret 明文入库;`CreateMenu`/`CreateApi` 直接 BodyParser 进模型。
- `AutoDiscoverAPIs` 每次启动删除不在路由栈里的记录,UI 手建的 API 重启即丢,并混入 HEAD/OPTIONS 行,逐条查询 N+1。
- `jwt.go` 里 `"NI"+"UBI_JWT_SECRET"` 是为绕开旧项目名检查而拆开的字符串,配置有两套来源。
- Cookie 模式分支散布于 jwt/auth/casbin/clientip/handlers,新增认证路由要改 3-4 处。
- 前端:16 处 `any`;`reauthenticate.tsx` 被压成一行;`user-menu.tsx` 313 行混改密/MFA/IP 白名单且与 `users/index.tsx` 重复;页面不用 Refine 的 hooks,每页重复 loading/dialog/error 状态;按中文分组名 `===` 做分支(`roles`/`apis`/`system-config`),换语言即失效;无路由级 code split(首包 275K + 211K + 256K);无 eslint/prettier/vitest,`@playwright/*` 在 devDependencies 但无任何 spec。
- 依赖:`mattn/go-sqlite3`(cgo) 与 modernc 并存、`CGO_ENABLED=0` 无法构建;测试依赖进了直接依赖。
- Makefile:`build-backend` 里每次 `go mod tidy` 改写 go.mod;`.PHONY` 不全;`npm install` 非 `npm ci`;`scaffold.sh` 的 Makefile 在 submodule 内 `go mod tidy` 会弄脏子模块。
- 默认监听 `172.18.0.1`(作者本机 Docker 网桥),别的机器启动即 bind 失败;`serve.go` 帮助文本写 0.0.0.0,与实际不符。vite 代理同样写死该地址。
- 无 Dockerfile、无 CI;核心 Cookie/MFA/PG/MySQL 测试依赖外部 DSN,CI 默认跳过。
- 日志:标准库 `log`,无级别、无 request-id;审计失败仅 `log.Printf`。

## P3

死代码 `MFAGenerateMfaToken`、`PermissionGroups`、`PermissionsByGroup`;`AllPermissionDefs` 的 `append` 依赖切片容量;`dashboard` 用日期字符串比较;表名硬编码未走 `TableName()`;审计日志 `end` 不含当天;单用户会话数无上限;`.air.toml` 的 root 与 cmd 路径语义待确认;`console.error` 作唯一前端日志;`@types/flatpickr` 疑冗余;`lucide-react` 偏旧;`keep-alive` 渲染期修改 ref;`tabs` 键不按用户隔离。

## 二开不友好专项

1. **能力被 `internal/` 锁死**:二开拿不到当前用户(`c.Locals("user_id"|"role")` 无公开 getter)、ClientIP、分页/批量、密码策略、审计写入、`models.User/Role/Menu`。只能手写表映射。
2. **无法替换认证/用户模型/日志**:`Options` 里没有 Authenticator、UserStore、Logger、PublicPaths、BodyLimit、ErrorHandler 等;`ExtraRoutes` 参数暴露 `*casbin.Enforcer`,把二开绑在 casbin 上;`AfterMigrate` 无 error 返回且在 `InitCasbin` 之前执行,拿不到 enforcer,没法给自有角色预置策略;`ensureAdminRoleMenus` 在其之前就跑完。
3. **`ExtraPermissionExpand` 是单个函数**,多个模块无法组合,没有注册表。
4. **两套入口语义分叉**:`Run`(宽松,`log.Fatal`)与 `NewRuntime`(严格,单实例,不能混用),选了一个就被锁在一种语义里,升级时两边容易漂移。
5. **全局可变状态**:`config.AppConfig`、`cache.Set`、`storage.Set`、`ExtraPermissionDefs/Expand`、`ExtraDashboardStats`;一进程一实例,单测要手工复位。
6. **没有版本化迁移**:只有 AutoMigrate + 每次启动重跑的 `ensure*`,历史遗留 `migrateBackendManageMenu`;底座升级对二开数据的影响不可知。种子菜单名/权限名写死中文,每次启动被重置覆盖。
7. **Submodule 工作流**:module 名 `go-nx-admin` 无域名,强依赖 replace;`frontend/dist` 目录不在 git 里,全新克隆 `go:embed` 编译失败,必须先建前端;`scaffold.sh` 对 `Makefile`/`cmd/main.go` 二次执行无条件覆盖,与 go.mod/.gitignore 的“已存在则跳过”不一致;生成的二进制没有 `serve`/`reset-password` 子命令,但提示仍写 `./app serve`;没有公开 `app.OpenDB`、`app.ResetPassword`。
8. **前端引入靠 alias + 自定义 Vite 解析插件**:`package.json` 为 private,无 `exports`、无 `peerDependencies`,二开要抄依赖清单,底座新增依赖会漏;`ui/*`、`useI18n`、`api`、`useTabActive` 无统一出口,只能深层 import,目录一重构就断。
9. **`createApp` 覆盖点不足**:不能替换登录页/MFA 页/布局、侧栏品牌、菜单图标映射(仅 12 个 lucide 图标,其余降级成 `Circle`)、主题色(`blue-600` 31 处硬编码,`tailwind.config` 的 `extend` 为空)、API 前缀、401 钩子、权限判断、dataProvider/authProvider;`createApp` 调用即改模块级 session 变量。`localStorage` 键名(`auth_token`、`tabs`、`locale`)全局硬编码,同域多应用会冲突。
10. **i18n 半成品**:`messages` 按 5 种语言平铺在每个文件里,新增语言要改 14 个文件;分组名靠中文字符串比对。

## 建议的修复顺序(待你决定是否授权)

1. **先堵 P0-1/2/3/4**:Casbin 策略锚定 + 精确白名单 + `StrictRouting/CaseSensitive` + `SaveRolePolicies` 只收 keys。这是真实越权,影响面最大,改动集中在 `casbin.go`、`permissions.go`、`crud.go`,并补回归测试。
2. **P0-5~8 启动期加固**:共用 `Config.Validate()`(默认密钥、CORS、监听 host),上传白名单,默认账号首启处理。
3. **跨库方言**:用 MariaDB + 临时 PG 把 `MyMenus`、`key`、`group`、`||` 跑通,并把 MySQL/PG 回归接进 CI(`NX_TEST_*` 默认配上 docker service)。
4. **统一 `Run` 与 `NewRuntime`**:抽 `buildApp`,`Run` 返回 error,去掉库代码里的 `log.Fatal`。
5. **二开公开 API**:`app.CurrentUser(c)`、`app.ClientIP(c)`、`app.Audit`、`PublicPaths`、`ExtraPermissions` 注册表、`AfterMigrate` 带 enforcer 与 error。
6. **前端引入契约**:`exports` + `peerDependencies`、`createApp` 增加 `loginPath/apiBase/basePath/onUnauthorized/theme/layout/loginPage`,清理中文字符串分支。
7. **工程化**:去掉 Makefile 里的 `go mod tidy`、补 `.PHONY`、`frontend/dist/.gitkeep`、Dockerfile、CI、eslint/prettier/vitest、路由级 code split。
