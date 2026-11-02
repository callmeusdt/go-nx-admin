# 开发规范

本文是基座和二开项目的通用开发规范。具体页面布局和 CRUD 检查清单见 [`crud-page-conventions.md`](crud-page-conventions.md);公开扩展入口以 `backend/app/`、`frontend/src/core.tsx` 和项目 `AGENTS.md` 为准。

## 国际化（i18n）

i18n 是标准功能，不是页面完成后的可选优化。所有基座和二开前端页面、组件、路由标题、提示、错误和空状态都必须从 i18n 字典取文案。

### 基本规则

- 页面和组件通过同一 `useI18n()` 上下文获取 `t`、`locale` 和语言切换能力。
- 可见文案使用稳定的模块化 key，例如 `orders.title`、`orders.create`、`orders.failed`；不要在 JSX 中直接写中文、英文或其他自然语言字符串。
- `aria-label`、`placeholder`、按钮标题、Tooltip、空状态、加载状态和错误提示同样属于可见/可访问文案，必须国际化。
- 路由和页签标题使用 `routes.<完整路径>` 或 `extraRouteLabels`；不要在布局、侧栏或页面中重复维护同一路径的硬编码标题。
- 服务端返回的错误不能直接作为界面文案；按错误类型映射到前端 i18n key，内部错误只显示通用失败文案。
- 业务数据中的用户自定义名称、描述等内容不翻译；系统生成的标签、状态、操作和类型必须翻译。

### 语言包维护

- 新增文案时，在对应模块文件 `frontend/src/contexts/messages/<module>.ts` 中维护，不把业务文案集中塞进一个大文件。
- 当前内置语言为 `zh-CN`、`en-US`、`zh-TW`、`es`、`vi`。新增 key 必须同时补齐全部内置语言；不能用某一种语言的文本冒充其他语言。
- 语言包 key 保持模块前缀，避免通用 key 与业务 key 冲突；确实通用的文案复用 `common.*`、`table.*`、`logs.*` 等现有模块。
- 语言包对象保持相同 key 集合。提交前检查缺失 key、拼写错误和无引用 key，避免切换语言后直接显示 key 本身。
- 文案包含变量时统一使用现有占位符约定（例如 `{count}`、`{username}`），由页面按既有方式替换；不要把业务值拼进语言 key。
- 语言切换应保留在用户明确选择的 locale；不要在页面内自行维护第二套 locale 状态或绕过 `I18nProvider`。

### 二开接入

二开通过 `createApp` 的 `i18n` 选项扩展语言和文案，不修改底座内置语言包：

```tsx
createApp({
  i18n: {
    languages: [
      { locale: 'zh-CN', label: '简体中文' },
      { locale: 'en-US', label: 'English' },
      { locale: 'ja-JP', label: '日本語' },
    ],
    defaultLocale: 'zh-CN',
    messages: {
      'zh-CN': {
        'orders.title': '订单',
        'orders.create': '新增订单',
      },
      'en-US': {
        'orders.title': 'Orders',
        'orders.create': 'Create order',
      },
      'ja-JP': {
        'orders.title': '注文',
        'orders.create': '注文を作成',
      },
    },
  },
})
```

- 二开扩展语言时，业务页面使用扩展 locale 的完整文案；底座文案仍由底座字典提供。
- `messages[locale]` 可以覆盖同 locale 的底座文案，例如 `login.title`；覆盖只影响显示，不新增菜单或权限。
- `routes.<完整路径>` 用于覆盖侧栏和页签标题；动态菜单和 API 权限仍由服务端授权决定。
- `configureI18n` 会让认证/数据 provider 等非 React 消费者使用同一套扩展字典；不要在 provider 中另建翻译表。
- 二开新增语言后，必须验证登录、侧栏、页签、表单、弹窗、错误和空状态等基座界面是否都有可用文案；缺失时应补齐扩展字典或回退到明确支持的语言。

### 检查清单

- [ ] 没有在 JSX、TSX、组件属性或页面逻辑中新增硬编码用户文案。
- [ ] 新增 key 放在正确的 `contexts/messages/*.ts` 模块中。
- [ ] 所有内置语言都补齐了新增 key；变量占位符一致。
- [ ] `aria-label`、placeholder、Tooltip 和错误/空/加载状态也已翻译。
- [ ] 新增路由已配置 `extraRouteLabels` 或 `routes.<完整路径>`，没有重复硬编码标题。
- [ ] 二开使用 `createApp({ i18n })` 扩展，没有修改底座字典或在 provider 中建立第二套翻译机制。
- [ ] 已验证语言切换、刷新、深链接和页签标题在目标语言下正常显示。
