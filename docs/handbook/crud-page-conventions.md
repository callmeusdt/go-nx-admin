# CRUD 页面规范

适用于基座和所有二开项目的数据管理页。目标是新页面不用再做视觉决策:按本文的区域顺序和组件对照搭,效果与用户、角色、菜单等现有页一致。本文也明确二开页面的范围边界:需求没有要求时,不额外增加“按 ID 查看”、详情卡片、行内详情区块或列表下方的详情面板。

组件来源均在 `frontend/src/components/ui/`,清单见 `docs/CODEMAP.md`。缺组件时先在 `ui/` 加封装,不在页面里临时拼样式(见 `AGENTS.md` 前端规范)。

> 说明:基座现有 CRUD 页都用**弹窗**编辑和新建。"独立编辑页/新建页"一节描述的是基座机制能支撑的做法,基座本身还没有现成实例,二开首次使用时请对照本文落地并回补样板。

> **AI/代理执行边界:**执行 CRUD 二开时,除非需求或开发者明确提出,不得自行添加“按 ID 查看”、查看详情按钮、详情卡片、行内展开详情、列表下方详情面板或额外的 ID 查询区块。看到模型有 `id` 字段、存在详情 API 或认为“这样更完整”,都不是增加这些 UI 的理由;不确定时保持列表页最小结构并等待明确需求。

### 页面密度

- 内容区外边距统一使用 `p-3 sm:p-4`;由布局组件统一控制,页面不要重复增加大块外边距。
- 标准 CRUD 区域之间使用 `space-y-3`;筛选栏保持 `p-3` 和 `gap-2`,不要为了压缩空间破坏控件的点击区域。
- 普通卡片使用 `p-3 sm:p-4`;表单字段之间保持 `space-y-4`,弹窗保持公共组件的 `p-6`。
- 不压缩表格行高、分页按钮、操作按钮或其他交互目标;高密度布局应减少容器留白,不能牺牲可操作性。
- 页面只负责区域间距;内容区内边距由 `KeepAliveOutlet` 统一提供,避免页面容器、卡片和组件重复叠加外边距。

## 1. 列表页

自上而下固定六个区域,没有的区域直接省略,不留空位:

| # | 区域 | 组件 | 规则 |
|---|---|---|---|
| 1 | 标题区 | `PageHeader` | 标题左,主操作按钮右;"新增"是主操作,放这里,不放表格上方。简单页面可用兼容的 `actions`;有两组动作时用 `primaryActions`/`secondaryActions` |
| 2 | 搜索区 | `FilterBar` + `FilterField` | 白色卡片;字段固定宽度靠左排,"重置/查询"靠右;输入框内回车触发查询;可以没有 |
| 3 | 批量编辑区 | `DataTable` 的 `selectable` + `batchActions` | 选中行才出现,位于表格卡片顶部;含"已选 N 项"、批量动作、"清除选择" |
| 4 | 数据区 | `DataTable` | 表头最后一列放列设置图标;操作列放 `IconButton`;状态/类型用 `Badge` |
| 5 | 分页工具条 | `DataTable` 的 `page`/`pageSize`/`total`/`onPageChange` | 左侧总数,右侧页码;没传 `total` 则不显示 |
| 6 | 扩展工具按钮 | `DataTable` 的 `toolbar` | 表格专属附加动作(如按当前列为粒度的导出、表格内同步),右对齐,位于表格上方;页面级刷新/导出/上传等辅助动作按 1.1 放 `secondaryActions`;没有就不传 |

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
- `primaryActions` 放新增、保存、提交等主要动作;`secondaryActions` 放刷新、导出、同步等辅助动作。
- 两组动作同时存在时,由 `PageHeader` 自动渲染分隔线;业务页不要自己拼竖线或用空白模拟分组。
- 旧页面可以继续使用 `actions`,它等价于主动作组;新页面需要分组时优先使用 `primaryActions`/`secondaryActions`。
- 辅助动作数量过多、权限状态复杂或窄屏无法保持清晰顺序时,业务页才自行选择下拉菜单;不要默认把辅助动作折叠。
- **范围边界:**列表页默认只展示列表、筛选、批量动作、分页和页面级工具。除非需求明确要求,不得增加“按 ID 查看”、按 ID 查询卡片、详情卡片、列表内展开详情、列表下方详情面板或额外的 ID 输入区块。

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
- 只有需求明确要求按 ID 精确筛选时,才增加 ID 筛选字段;这属于搜索区的一项后端筛选,不是额外的“按 ID 查看”区块。

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
- 只有字段具备稳定、明确且对用户有价值的排序语义时才启用排序;不要因为列是数字、日期或存在数据库字段就默认加排序。
- 启用排序时,列设置 `sortable: true`,表格传入 `sort`/`defaultSort` 和 `onSortChange`;只在真实可排序的列上显示排序图标。操作列、富文本、状态 Badge、拼接展示列和无稳定顺序的计算列默认不排序。
- `sort` 用于受控排序状态,适合服务端分页和后端排序;`defaultSort` 只用于初始化默认字段和方向,适合页面首次加载时带上默认排序。默认方向显式写成 `{ field: 'created_at', order: 'desc' }`,不要依赖后端隐式默认值。
- 排序图标和交互由 `DataTable` 统一提供,业务页面不要在表头自行放箭头或按钮。点击顺序为无排序 → 升序 → 降序 → 清除,并通过 `aria-sort` 暴露当前状态。
- 服务端分页或数据量较大的列表必须由后端执行排序;前端只提交白名单字段和 `asc`/`desc`,不能把请求中的排序字段直接拼接 SQL。排序字段必须映射到固定数据库列或安全表达式。
- 排序与筛选、分页联动时回到第 1 页;切换筛选条件或排序条件后,旧请求不能覆盖新结果。后端不支持排序时不要显示排序图标。

#### 1.4.1 排序接入模板

```tsx
const [sort, setSort] = useState<SortState | null>(null)

const load = async (nextSort = sort) => {
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
  if (nextSort) {
    params.set('sort', nextSort.field)
    params.set('order', nextSort.order)
  }
  const response = await api.get(`/orders?${params}`)
  setRows(response.data)
  setTotal(response.total)
}

<DataTable
  columns={[
    { key: 'name', title: t('orders.name'), sortable: true },
    { key: 'created_at', title: t('orders.created_at'), sortable: true },
    { key: 'actions', title: t('common.actions'), required: true, align: 'right' },
  ]}
  sort={sort}
  onSortChange={next => {
    setSort(next)
    setPage(1)
    void load(next)
  }}
  rows={rows}
  rowKey={row => row.id}
  page={page}
  pageSize={pageSize}
  total={total}
  onPageChange={nextPage => { setPage(nextPage); void load(sort) }}
/>
```

- `sort.field` 必须是前端列 key 与后端白名单之间的映射结果;不要直接信任任意请求字段。
- 如果 API 使用 `sort_by`/`direction` 等其他参数,在页面 provider 或请求适配层统一转换,不要让每个列表自行发明一套参数。
- 对一次性小列表,可以在前端排序,但仍应只对明确允许排序的列显示 `sortable`;不要对权限、操作和展示拼接列排序。
- 给每张表一个稳定的 `storageKey`,列设置选择会存到本地。
- 粘性列:宽表用 `stickyStart={n}` 让前 n 列、`stickyEnd={n}` 让后 n 列粘在表格两侧,横向滚动时它们不动,其余列从下面滚过。详见 1.4.1。
- 行内状态一律 `Badge`:启用 `success`、禁用 `danger`、分类 `info`、系统内置 `warning`、其他 `neutral`;在线状态用 `shape="pill"`。
- 行内操作用 `IconButton`,传 `label`(悬浮提示和无障碍名称)和 `tone`:编辑 `primary`、删除 `danger`、授权类 `success`/`primary`、其他默认。
- 列级水平对齐使用 `DataTableColumn.align` (`left`/`center`/`right`),同时作用于表头和单元格;操作列按需要传 `align: 'center'` 或 `align: 'right'`,不要在 `render` 中重复包对齐容器或使用 `!important` 覆盖默认样式。
- 不可执行的操作保持显示并 `disabled`,提示文字说明原因(如"不能删除当前登录账户"),不要隐藏。
- 空数据传 `emptyText`,默认文案 `common.no_data`。
- 树形数据(如菜单)用同一个 `DataTable`,缩进放在名称列的 `render` 里,不另写 `<table>`。
- 弹窗里的小列表(在线会话、IP 白名单)用 `SimpleTable*`,不用 `DataTable`。

#### 1.4.1 粘性列

```tsx
<DataTable
  storageKey="orders"
  selectable
  stickyStart={2}  // 前 2 个数据列(如 ID、名称)
  stickyEnd={1}    // 最后 1 列(操作列)
  minWidth={900}
  columns={[...]}
/>
```

- 计数针对**当前可见的数据列**,不含选择框列;`selectable` 时选择框列自动随首侧粘住,不用算进 `stickyStart`。列设置里隐藏某列后,粘性数量按剩余可见列重新计算,所以被粘住的列通常应是不可隐藏的(`required`)或默认必显的列。
- 两个数值都是可选的,默认不固定。传 0、负数或超过列数时按边界处理;首尾两组不会重叠同一列。
- 含操作列的表默认 `stickyEnd={1}`,操作按钮在窄屏和长表里始终可点。身份列(ID、名称、用户名)放在前面,需要时用 `stickyStart`。
- 粘性列越多,可滚动区域越小。粘性列总宽度不要超过表格可视宽度的一半;窄屏(约 600px)下首侧最多粘 2 列。
- 粘性列自带不透明背景和边缘阴影,页面不要给它的 `className` 再设背景色(会盖掉悬停和选中态),也不要自己写 `position: sticky`。
- 只在表格需要横向滚动时才有意义;`minWidth` 不大于容器宽度时,粘性列看不出区别。

### 1.5 分页

- 服务端分页:`page`、`pageSize`、`total`、`onPageChange` 四个一起传。默认每页 20。
- 数据量小、一次返回全部的列表(角色、菜单、API)不传 `total`,不显示分页条。
- 翻页、筛选、重置都要回到第 1 页。

### 1.6 扩展工具按钮

- 只放**表格专属**的附加动作(如按当前列为粒度的导出、表格内同步);页面级的刷新、导出、上传、备份等辅助动作按 1.1 放 `PageHeader.secondaryActions`,不要放在这里。
- 放在 `toolbar`,出现在表格上方右侧。
- 导出用 `ExportButton`(组件已提供,基座页面暂无使用实例);其他按钮用 `Button size="sm" variant="outline"`。
- 它是整张表的动作,不依赖选中行;依赖选中行的动作放批量编辑区。

### 1.6.1 分页专属扩展

- 需要在分页控件旁增加每页大小、视图切换或分页范围等控件时,使用 `DataTable.paginationToolbar`。
- `paginationToolbar` 只放分页语境内的控件;刷新、导出、上传、备份等页面级动作仍放 `PageHeader.secondaryActions`。
- 分页扩展与上一页/页码/下一页控件保持同一行;组件会保持分页控件不换行,页面不要自行复制分页 footer。

### 1.7 页签与刷新

- 页面挂在 `KeepAliveOutlet` 里,切换页签时保留筛选、分页和滚动位置。
- 需要"回到页签时刷新"的列表(日志)调用 `useOnTabActivate(() => void reload())`。
- 实时、轮询类页面在 `createApp({ keepAliveExclude: ['/path'] })` 里登记,离开页签就卸载。

### 1.8 请求状态、重复提交与竞态

- 列表首次加载用 `Spinner`/`Skeleton`,加载失败用 `Alert`;不要在页面里手写错误 `<p>`。
- 查询、重置、翻页、批量操作和保存期间,相关按钮必须 disabled,避免重复提交。
- 筛选、翻页或排序改变后,旧请求返回不能覆盖较新的列表结果;用请求序号、AbortController 或现有数据层机制保证结果只提交给当前查询。
- 保存/删除/批量操作成功后统一刷新受影响列表;失败时保留表单输入、筛选状态和必要的弹窗上下文。
- 服务端 5xx 或数据库错误只显示通用失败文案,不要把 `err.message`、SQL、堆栈或内部错误原文直接显示给用户;只有明确的业务校验错误才显示具体提示。
- 列表空状态只在请求成功且确实没有数据时显示;加载中或失败时不要同时显示“暂无数据”。

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
- 侧栏菜单只登记列表页;新建/编辑页从列表页的按钮跳转(`navigate`),不单独做菜单项。深链接进入已授权的二开路径时,页签系统应保留该路径对应的页面;二开仍需通过 `extraRouteLabels` 或 `RouteLabelsContext` 提供完整路径的标题,并验证关闭、刷新和返回行为。
- 带参数路径不会在侧栏高亮定位菜单(基座既有行为)。
- 不要把每个详情路径都注册成侧栏菜单;菜单授权控制入口可见性,接口仍必须单独鉴权。

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
- 只有需求明确要求时才提供详情页或详情路径;“编辑”不自动意味着要再增加“查看”页面。
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

## 5. 二开边界与接口约定

### 5.1 二开扩展边界

- 二开业务后端放在二开项目自身的 `internal/` 或公开业务包中,只能通过底座公开的 `go-nx-admin/app` 接口接入;不要 import 底座 `backend/internal/`。
- 业务 API 通过 `app.Options.ExtraRoutes` 注入;公开接口才使用 `ExtraPublicRoutes`,不要为了省事绕过 JWT/Casbin。
- 新增业务权限同时维护 `ExtraPermissions` 与 `ExtraPermissionExpand`;权限 key 使用 `资源:操作`,路径参数使用 `[^/]+`。
- 前端业务资源、路由和标题通过 `createApp({ extraResources, extraRoutes, extraRouteLabels })` 注入;业务页面放在二开项目自己的 `frontend/src/pages/`。
- 菜单可见性不等于接口授权。列表页隐藏某个按钮不能替代后端权限检查,后端每个读写接口都必须独立鉴权。
- 删除或修改路由、权限 key、资源名时,同步检查菜单、权限定义、权限展开、前端资源和页签标题,不要只改一层。

### 5.2 推荐 API 响应结构

二开列表接口优先使用与 `DataTable`/data provider 对齐的结构:

```json
{
  "data": [],
  "total": 100
}
```

- 服务端分页列表返回 `data` 和 `total`;`total` 是当前筛选条件下的总数。
- 一次性返回的角色、菜单等小列表只返回 `data`,页面不传分页参数。
- 详情接口返回单个对象;只有需求明确需要详情页面或详情弹窗时才提供,不要因有 `id` 字段自动增加“按 ID 查看”入口。
- 新建、编辑成功后返回已保存对象或明确的成功状态,不要让页面猜测响应结构。
- 批量操作返回逐 ID 的成功/失败结果,页面据此清理成功选中项并展示失败汇总;不要把部分失败伪装成全成功。
- 错误响应遵循底座通用错误策略:内部原因写日志,响应不回显数据库、驱动或堆栈信息;业务校验错误才返回可操作的业务文案。

### 5.3 二开 CRUD 最小模板

新建二开列表页可按下面的骨架开始,再按需求删减区域;不要在骨架之外自行增加“按 ID 查看”区块:

```tsx
<div className="space-y-4">
  <PageHeader
    title={t('orders.title')}
    primaryActions={
      <Button size="sm" onClick={openCreate}>
        {t('orders.create')}
      </Button>
    }
    secondaryActions={
      <Button size="sm" variant="outline" onClick={reload} disabled={loading}>
        {t('common.refresh')}
      </Button>
    }
  />

  {error && <Alert>{error}</Alert>}

  <FilterBar onQuery={applyFilters} onReset={resetFilters}>
    <FilterField>
      <Input aria-label={t('orders.keyword')} value={filters.keyword} onChange={...} />
    </FilterField>
  </FilterBar>

  <DataTable
    storageKey="orders"
    columns={columns}
    data={items}
    page={page}
    pageSize={pageSize}
    total={total}
    onPageChange={setPage}
    selectable
    selectedKeys={selected}
    onSelectionChange={setSelected}
    batchActions={batchActions}
    stickyEnd={1}
  />
</div>
```

- `columns` 中操作列使用 `IconButton`,状态使用 `Badge`,操作列设置 `required: true`。
- 批量动作使用 `useBatch`;详情、编辑和删除是否存在必须由需求决定,不能因为列表有 `id` 就默认加入查看按钮。
- 新增、编辑和删除分别遵循第 2 节;字段较多时才升级为第 3 节的独立页面。

## 6. 新页面检查清单

提交前对照:

- [ ] 页头用 `PageHeader`,没有自己写 `h2`。
- [ ] 主动作和辅助动作分别使用 `primaryActions`/`secondaryActions`;简单旧页面可继续使用 `actions`;没有自己手写分隔线。
- [ ] 页面级刷新/导出/上传/备份在 `PageHeader.secondaryActions`,`DataTable.toolbar` 只放表格专属附加动作。
- [ ] 分页语境内的自定义控件使用 `DataTable.paginationToolbar`,没有复制或重写分页 footer。
- [ ] 搜索区用 `FilterBar`,筛选在后端完成,"查询/重置/回车"都能用。
- [ ] 列表用 `DataTable`,有 `storageKey`,列设置能用,操作列 `required`;有操作列的表设 `stickyEnd={1}`,列多的表按需设 `stickyStart`。
- [ ] 排序只在业务有稳定排序语义且后端/数据源真实支持时启用;使用 `sortable` + 受控 `sort`/`onSortChange`,没有自绘排序图标。
- [ ] 服务端排序字段经过白名单映射,排序方向只允许 `asc`/`desc`,没有把请求字段直接拼接 SQL。
- [ ] 排序与筛选、分页联动时回到第 1 页,旧请求不会覆盖新排序结果;不支持排序的列没有排序图标。
- [ ] 行内状态用 `Badge`,图标按钮用 `IconButton` 带 `label`。
- [ ] 批量操作走 `useBatch`,不可操作的行用 `isRowDisabled`。
- [ ] 新建/编辑表单用 `FormField`,布尔值用 `Switch`,下拉用 `Select`,勾选用 `Checkbox`。
- [ ] 删除用 `ConfirmDialog`,危险动作前后端双重保护。
- [ ] 没有原生 `<select>`、`<input type=checkbox>`、`window.confirm`、手写错误 `<p>`、手写标签 `<span>`。
- [ ] 文案五种语言都补了。
- [ ] 窄屏(约 600px)下搜索区自动换行,表格横向滚动,弹窗不溢出。
- [ ] 新增权限同时维护 `permissions.go` 定义与展开(见 `AGENTS.md` 权限维护)。
- [ ] 二开后端只通过公开 `app` 扩展点接入,前端通过 `createApp` 扩展点接入,没有 import 底座 `backend/internal/`。
- [ ] 列表接口遵循 `data`/`total` 约定;批量接口能表达逐 ID 的部分失败。
- [ ] 保存、删除、批量和查询期间防重复提交,旧请求不会覆盖新查询结果。
- [ ] 5xx/数据库错误没有直接展示内部错误原文。
- [ ] 没有需求时没有增加“按 ID 查看”、详情卡片、列表内详情展开或额外详情区块。
- [ ] 只有需求明确要求时才增加详情页、查看按钮或 ID 精确筛选;三者不能互相推导。

## 7. 现有页面对照

| 页面 | 搜索区 | 批量 | 分页 | 编辑方式 | 粘性列 |
|---|---|---|---|---|---|
| `users` | 用户名、角色 | 启用/禁用/删除 | 前端不分页(接口返回全部,支持筛选参数) | 弹窗 | 首 2 + 尾 1 |
| `roles` | 无 | 删除 | 无 | 弹窗 + 授权弹窗 | 尾 1 |
| `menus` | 无 | 无 | 无 | 弹窗(树形 `DataTable`) | 尾 1 |
| `apis` | 内联新增表单 | 删除 | 无 | 行内表单 | 尾 1 |
| `audit-logs` | 操作人、路径、方法、IP、起止时间 | 无 | 服务端 | 只读 | 无 |
| `login-logs` | 用户名、状态、IP、起止时间 | 无 | 服务端 | 只读 | 无 |
| `media` | 分类 Tabs | 删除 | 前端页码 | 上传 + 确认 | 无 |
| `system-config` | 无 | 无 | 无 | 整页表单 | 无 |

`system-config` 是现有唯一的整页表单:`PageHeader` + 保存按钮在右上、分组卡片、`FormField`。做独立编辑页时可参考它的卡片分区,但保存操作条按 3.2 放在底部。
