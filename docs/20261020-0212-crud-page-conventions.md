# CRUD 页面规范

适用于基座和所有二开项目的数据管理页。目标是新页面不用再做视觉决策:按本文的区域顺序和组件对照搭,效果与用户、角色、菜单等现有页一致。

组件来源均在 `frontend/src/components/ui/`,清单见 `docs/CODEMAP.md`。缺组件时先在 `ui/` 加封装,不在页面里临时拼样式(见 `AGENTS.md` 前端规范)。

> 说明:基座现有 CRUD 页都用**弹窗**编辑和新建。"独立编辑页/新建页"一节描述的是基座机制能支撑的做法,基座本身还没有现成实例,二开首次使用时请对照本文落地并回补样板。

## 1. 列表页

自上而下固定六个区域,没有的区域直接省略,不留空位:

| # | 区域 | 组件 | 规则 |
|---|---|---|---|
| 1 | 标题区 | `PageHeader` | 标题左,主操作按钮右;"新增"是主操作,放这里,不放表格上方 |
| 2 | 搜索区 | `FilterBar` + `FilterField` | 白色卡片;字段固定宽度靠左排,"重置/查询"靠右;输入框内回车触发查询;可以没有 |
| 3 | 批量编辑区 | `DataTable` 的 `selectable` + `batchActions` | 选中行才出现,位于表格卡片顶部;含"已选 N 项"、批量动作、"清除选择" |
| 4 | 数据区 | `DataTable` | 表头最后一列放列设置图标;操作列放 `IconButton`;状态/类型用 `Badge` |
| 5 | 分页工具条 | `DataTable` 的 `page`/`pageSize`/`total`/`onPageChange` | 左侧总数,右侧页码;没传 `total` 则不显示 |
| 6 | 扩展工具按钮 | `DataTable` 的 `toolbar` | 导出、同步等页面级附加动作,右对齐,位于表格上方;没有就不传 |

加载失败的提示用 `Alert`,放在标题区和搜索区之间。

### 1.1 标题区

```tsx
<PageHeader
  title={t('users.title')}
  actions={<Button size="sm" onClick={openCreate}><Plus className="w-4 h-4 mr-1" />{t('users.create')}</Button>}
/>
{loadFailed && <Alert>{t('ui.request_failed')}</Alert>}
```

- 标题字号由 `PageHeader` 统一,页面不自己写 `h2`。
- 没有新增能力的页面(如日志)不传 `actions`。

### 1.2 搜索区

```tsx
<FilterBar onQuery={() => setApplied(filters)} onReset={() => { setFilters(empty); setApplied(empty) }}>
  <FilterField><Input placeholder={t('users.username')} aria-label={t('users.username')} value={filters.username} onChange={...} /></FilterField>
  <FilterField><Select aria-label={t('users.role')} value={filters.role_id} onValueChange={...} options={[{ value: '', label: t('users.all_roles') }, ...]} /></FilterField>
</FilterBar>
```

- 编辑中的筛选值(`filters`)和已生效的筛选值(`applied`)分开存,点"查询"才生效;"重置"同时清空两者并刷新。
- 下拉的"全部"选项用 `value: ''`。
- 日期时间用 `DateTimeInput`,不用原生日期控件。
- 每个控件要有 `aria-label` 或 `placeholder`,两者至少一个。
- 筛选必须由后端完成,接口接收查询参数;日志类列表必须后端分页筛选,禁止取固定最近 N 条再前端过滤。
- 搜索项少时不要拉满整行:`FilterField` 默认 192px,超出自动换行。

### 1.3 批量编辑区

```tsx
<DataTable
  selectable
  selectedKeys={selected}
  onSelectionChange={keys => setSelected(keys as number[])}
  isRowDisabled={u => u.id === currentUserId}
  batchActions={({ selectedKeys }) => (<>...</>)}
/>
{batch.dialog}
```

- 批量动作统一走 `hooks/use-batch.tsx` 的 `useBatch`:它负责确认框、逐 id 执行和失败汇总,页面只给 `labelKey`、`confirmKey`、`destructive` 和动作函数。
- 不可操作的行(当前登录账号、系统内置角色)用 `isRowDisabled`,复选框会禁用,全选也会跳过它们。
- 批量删除、批量禁用这类破坏性动作传 `destructive: true`,确认按钮变红。
- 批量完成后清掉已成功的选中项并刷新列表;汇总文案用 `BatchSummary` 显示在表格上方。

### 1.4 数据区

- 列定义用 `DataTableColumn`:`title` 是表头,`label` 是列设置里显示的纯文本,`required: true` 的列(通常是"操作")不可被隐藏。
- 给每张表一个稳定的 `storageKey`,列设置选择会存到本地。
- 固定列:宽表用 `fixedLeft={n}` 固定前 n 列、`fixedRight={n}` 固定后 n 列,横向滚动时它们停在表格两侧,其余列从下面滚过。详见 1.4.1。
- 行内状态一律 `Badge`:启用 `success`、禁用 `danger`、分类 `info`、系统内置 `warning`、其他 `neutral`;在线状态用 `shape="pill"`。
- 行内操作用 `IconButton`,传 `label`(悬浮提示和无障碍名称)和 `tone`:编辑 `primary`、删除 `danger`、授权类 `success`/`primary`、其他默认。
- 不可执行的操作保持显示并 `disabled`,提示文字说明原因(如"不能删除当前登录账户"),不要隐藏。
- 空数据传 `emptyText`,默认文案 `common.no_data`。
- 树形数据(如菜单)用同一个 `DataTable`,缩进放在名称列的 `render` 里,不另写 `<table>`。
- 弹窗里的小列表(在线会话、IP 白名单)用 `SimpleTable*`,不用 `DataTable`。

#### 1.4.1 固定列

```tsx
<DataTable
  storageKey="orders"
  selectable
  fixedLeft={2}   // 前 2 个数据列(如 ID、名称)
  fixedRight={1}  // 最后 1 列(操作列)
  minWidth={900}
  columns={[...]}
/>
```

- 计数针对**当前可见的数据列**,不含选择框列;`selectable` 时选择框列自动随左侧固定,不用算进 `fixedLeft`。列设置里隐藏某列后,固定数量按剩余可见列重新计算,所以被固定的列通常应是不可隐藏的(`required`)或默认必显的列。
- 两个数值都是可选的,默认不固定。传 0、负数或超过列数时按边界处理;左右固定不会重叠同一列。
- 含操作列的表默认 `fixedRight={1}`,操作按钮在窄屏和长表里始终可点。身份列(ID、名称、用户名)放在前面,需要时用 `fixedLeft`。
- 固定列越多,可滚动区域越小。固定列总宽度不要超过表格可视宽度的一半;窄屏(约 600px)下左侧最多固定 2 列。
- 固定列自带不透明背景和边缘阴影,页面不要给它的 `className` 再设背景色(会盖掉悬停和选中态),也不要自己写 `position: sticky`。
- 只在表格需要横向滚动时才有意义;`minWidth` 不大于容器宽度时,固定列看不出区别。

### 1.5 分页

- 服务端分页:`page`、`pageSize`、`total`、`onPageChange` 四个一起传。默认每页 20。
- 数据量小、一次返回全部的列表(角色、菜单、API)不传 `total`,不显示分页条。
- 翻页、筛选、重置都要回到第 1 页。

### 1.6 扩展工具按钮

- 放在 `toolbar`,出现在表格上方右侧。
- 导出用 `ExportButton`(组件已提供,基座页面暂无使用实例);其他按钮用 `Button size="sm" variant="outline"`。
- 它是整张表的动作,不依赖选中行;依赖选中行的动作放批量编辑区。

### 1.7 页签与刷新

- 页面挂在 `KeepAliveOutlet` 里,切换页签时保留筛选、分页和滚动位置。
- 需要"回到页签时刷新"的列表(日志)调用 `useOnTabActivate(() => void reload())`。
- 实时、轮询类页面在 `createApp({ keepAliveExclude: ['/path'] })` 里登记,离开页签就卸载。

## 2. 弹窗型新建/编辑

基座现有做法,小型表单(约 8 个字段以内)默认用它。同一个 `Dialog` 同时承载新建和编辑,用 `mode` 区分。

```tsx
const [dialog, setDialog] = useState<{ open: boolean; mode: 'create' | 'edit'; item?: Item }>({ open: false, mode: 'create' })
```

### 2.1 结构

```tsx
<Dialog open={dialog.open} onOpenChange={v => !v && setDialog(p => ({ ...p, open: false }))}>
  <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
    <DialogHeader>
      <DialogTitle>{dialog.mode === 'create' ? t('x.create') : t('x.edit')}</DialogTitle>
    </DialogHeader>
    <div className="space-y-4 py-2">
      <FormField label={...}>{id => <Input id={id} ... />}</FormField>
      ...
      {error && <Alert>{error}</Alert>}
      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button variant="outline" onClick={close}>{t('common.cancel')}</Button>
        <Button onClick={handleSave} disabled={saving}>{saving ? t('x.saving') : t('common.save')}</Button>
      </div>
    </div>
  </DialogContent>
</Dialog>
```

- 标题:新建用"新增 X",编辑用"编辑 X"。`DialogHeader` 自带底部分隔线,不要再自己加。
- 宽度:默认 `max-w-lg`;表单行多列时 `max-w-xl`;内嵌表格的弹窗 `max-w-3xl`;纯勾选列表 `max-w-md`。
- 按钮区:右对齐,"取消"在左(`outline`),"保存"在右(主按钮)。保存中按钮禁用并显示"保存中…"。
- 错误:接口失败的提示放在按钮区上方,用 `Alert`;表单校验错误同一位置。

### 2.2 表单字段

- 字段行用 `FormField`:label 在上,控件在下,可选 `hint`(灰色小字)和 `required`。
- 两列并排用 `grid grid-cols-1 sm:grid-cols-2 gap-3`,窄屏自动变单列。
- 布尔字段用 `Switch`:开关在左,右侧是 label 加说明,不用 `FormField` 包。
- 多选列表用 `Checkbox`;只有 2~4 个互斥选项且需要同时看到时用 `RadioGroup`,更多用 `Select`。
- 密码输入 `type="password"`;编辑时留空表示不修改,label 写明"(留空不修改)"。
- 编辑时不可改的字段(如用户名)用 `readOnly` 加灰底,同时后端也要忽略该字段,不能只靠前端。

### 2.3 新建与编辑的差异

| 项 | 新建 | 编辑 |
|---|---|---|
| 初始值 | 空表单加合理默认(如默认角色、启用) | 用当前行数据回填 |
| 必填 | 按业务必填,密码等必须填 | 密码等可选填 |
| 仅编辑出现 | — | 状态开关等 |
| 打开前 | 清空 `error` | 清空 `error` |
| 保存成功 | 关闭弹窗并刷新列表 | 关闭弹窗并刷新列表 |

### 2.4 删除确认

删除和清除类动作用 `ConfirmDialog`,不用 `window.confirm`,也不自己拼弹窗:

```tsx
<ConfirmDialog
  open={!!delId}
  onOpenChange={v => !v && setDelId(null)}
  onConfirm={handleDelete}
  description={t('users.delete_hint')}
  destructive
/>
```

- 危险动作要前后端双重保护:删自身、删系统角色等,按钮在前端 `disabled`,后端同样明确拒绝。
- 接口失败时把错误通过 `error` 传给 `ConfirmDialog`,弹窗保持打开。

## 3. 独立编辑页/新建页

表单字段多(约 8 个以上)、含富文本或上传、分步骤、需要多个分区时,用页面而不是弹窗。基座没有现成实例,按下面的约定落地。

### 3.1 路由与入口

- 路径约定:列表 `/orders`,新建 `/orders/new`,编辑 `/orders/:id/edit`。
- 通过 `createApp({ extraRoutes })` 注入 `<Route>`;二开项目在 `frontend/src/pages/` 下放页面。
- 页签只在点击侧栏菜单项时创建(`sidebar.tsx` 的 `addTab`),`extraRouteLabels` 只覆盖已有页签的标题。`/orders/new`、`/orders/:id/edit` 这类不在菜单里的路径,**基座目前不会为它们建页签**,二开落地时要自行验证页签显示与关闭行为,必要时扩展 `tabs-context`。
- 侧栏菜单只登记列表页;新建/编辑页从列表页的按钮跳转(`navigate`),不单独做菜单项。
- 带参数路径不会在侧栏高亮定位菜单(基座既有行为)。

### 3.2 页面结构

自上而下:

1. `PageHeader`:标题("新增 X"/"编辑 X"),右侧放"返回列表"次按钮。
2. 加载中用 `Spinner`;加载失败用 `Alert`,并提供返回。
3. 表单主体:一张白色卡片(`bg-white rounded-lg border border-gray-200 p-4 sm:p-6`),卡片内用 `FormField`、`Switch` 等,分区用小标题 `h3 text-sm font-semibold text-slate-700`,分区之间 `space-y-6`。
4. 底部操作条:右对齐,"取消"(`outline`)和"保存"(主按钮),保存中禁用。
5. 错误 `Alert` 放在操作条上方。

### 3.3 行为约定

- 新建与编辑复用同一个表单组件,以 `id` 是否存在区分;编辑进入时先请求详情再回填。
- 保存成功后返回列表,并让列表刷新:列表页用 `useOnTabActivate` 重新加载,或保存后直接导航回列表后由列表在挂载时加载。
- 有未保存修改时离开要拦截提示,用 `ConfirmDialog`。
- 一个路径对应一个页签,同一条记录的编辑页重复打开会复用页签;不同记录是不同路径,会各自开页签。表单状态随页签保留(keep-alive),关闭页签才丢弃,这点要在文案或交互上让用户有预期。
- 表单字段控件、必填、只读、错误规则与弹窗相同(见 2.2)。

### 3.4 弹窗还是页面

| 条件 | 选择 |
|---|---|
| 字段 ≤ 8 个,无富文本/上传/分区 | 弹窗 |
| 需要和列表对照着看、改完立刻回到列表 | 弹窗 |
| 字段多、分区多、有富文本或大文件上传 | 页面 |
| 需要保留草稿、可以分享链接给他人 | 页面 |
| 只是确认或单个值的修改 | `ConfirmDialog` 或弹窗,不用页面 |

## 4. 文案与可访问性

- 所有可见文案走 `t('模块.键')`,新增键五种语言(zh-CN、en、zh-TW、es、vi)都要补,按模块放在 `contexts/messages/*.ts`。
- 通用文案复用 `common.*`、`logs.*`(重置/查询/分页)、`table.*`,不重复造键。
- 图标按钮必须有 `label`;表单控件必须有可访问名称(`FormField` 的 label 或 `aria-label`)。
- 错误用 `Alert`(错误自动 `role=alert`),成功和提示用 `variant` 区分。
- 操作后的轻反馈(如"已复制")用 `useToast`;表单字段错误仍然用 `Alert`,不要用 toast。

## 5. 新页面检查清单

提交前对照:

- [ ] 页头用 `PageHeader`,没有自己写 `h2`。
- [ ] 搜索区用 `FilterBar`,筛选在后端完成,"查询/重置/回车"都能用。
- [ ] 列表用 `DataTable`,有 `storageKey`,列设置能用,操作列 `required`;有操作列的表设 `fixedRight={1}`,列多的表按需设 `fixedLeft`。
- [ ] 行内状态用 `Badge`,图标按钮用 `IconButton` 带 `label`。
- [ ] 批量操作走 `useBatch`,不可操作的行用 `isRowDisabled`。
- [ ] 新建/编辑表单用 `FormField`,布尔值用 `Switch`,下拉用 `Select`,勾选用 `Checkbox`。
- [ ] 删除用 `ConfirmDialog`,危险动作前后端双重保护。
- [ ] 没有原生 `<select>`、`<input type=checkbox>`、`window.confirm`、手写错误 `<p>`、手写标签 `<span>`。
- [ ] 文案五种语言都补了。
- [ ] 窄屏(约 600px)下搜索区自动换行,表格横向滚动,弹窗不溢出。
- [ ] 新增权限同时维护 `permissions.go` 定义与展开(见 `AGENTS.md` 权限维护)。

## 6. 现有页面对照

| 页面 | 搜索区 | 批量 | 分页 | 编辑方式 | 固定列 |
|---|---|---|---|---|---|
| `users` | 用户名、角色 | 启用/禁用/删除 | 前端不分页(接口返回全部,支持筛选参数) | 弹窗 | 左 2 + 右 1 |
| `roles` | 无 | 删除 | 无 | 弹窗 + 授权弹窗 | 右 1 |
| `menus` | 无 | 无 | 无 | 弹窗(树形 `DataTable`) | 右 1 |
| `apis` | 内联新增表单 | 删除 | 无 | 行内表单 | 右 1 |
| `audit-logs` | 操作人、路径、方法、IP、起止时间 | 无 | 服务端 | 只读 | 无 |
| `login-logs` | 用户名、状态、IP、起止时间 | 无 | 服务端 | 只读 | 无 |
| `media` | 分类 Tabs | 删除 | 前端页码 | 上传 + 确认 | 无 |
| `system-config` | 无 | 无 | 无 | 整页表单 | 无 |

`system-config` 是现有唯一的整页表单:`PageHeader` + 保存按钮在右上、分组卡片、`FormField`。做独立编辑页时可参考它的卡片分区,但保存操作条按 3.2 放在底部。
