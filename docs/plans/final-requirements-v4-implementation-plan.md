# 最终需求清单 v4 落地计划

## 1. 文档状态

- 状态：执行计划草案 v1
- 上游基线文档：[docs/requirements/final-requirements-v4.md](/home/heligong/Downloads/sub2api/docs/requirements/final-requirements-v4.md)
- 用途：指导后续分阶段开发、节点验收、排期与 PR 切分
- 细粒度节点文档：[docs/plans/final-requirements-v4-detailed-node-plan.md](/home/heligong/Downloads/sub2api/docs/plans/final-requirements-v4-detailed-node-plan.md)
- 数据库字段清单：[docs/implementation/final-requirements-v4-database-fields.md](/home/heligong/Downloads/sub2api/docs/implementation/final-requirements-v4-database-fields.md)
- API 清单：[docs/implementation/final-requirements-v4-api-list.md](/home/heligong/Downloads/sub2api/docs/implementation/final-requirements-v4-api-list.md)
- 验证模板：[docs/implementation/final-requirements-v4-validation-templates.md](/home/heligong/Downloads/sub2api/docs/implementation/final-requirements-v4-validation-templates.md)
- 测试数据初始化清单：[docs/implementation/final-requirements-v4-test-data-bootstrap.md](/home/heligong/Downloads/sub2api/docs/implementation/final-requirements-v4-test-data-bootstrap.md)

## 2. 计划目标

在尽量降低后续上游合并成本的前提下，按可独立验收的节点，逐步落地以下能力：

- `admin / sales / user` 三层角色
- 邀请链与销售归属链
- 仅直接邀请人的首充一次性奖励
- 用户自己的充值记录与开票入口
- 销售查看自己归属范围内的数据
- 管理员修改邀请归属与销售归属
- 百望云开票

## 3. 规划原则

### 3.1 升级友好

优先新增模块与新表，少重写现有热点文件。

重点控制改动面的现有热点文件：

- `backend/internal/service/payment_order.go`
- `backend/internal/service/payment_fulfillment.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/server/middleware/admin_auth.go`
- `frontend/src/router/index.ts`
- `frontend/src/components/layout/AppSidebar.vue`

### 3.2 节点粒度

每个节点都应满足：

- 可以单独开发与合并
- 有明确依赖关系
- 有明确退出标准
- 有最小验证命令

### 3.3 先数据、后链路、再界面

推荐顺序：

1. 数据模型与角色基础
2. 注册与归属链
3. 管理员归属调整
4. 订单快照与用户充值记录
5. 邀请奖励
6. 销售数据域与销售后台
7. 发票资料与用户开票申请
8. 百望云异步开票
9. 加固与回归

## 4. 依赖关系总览

### 串行主链

- 节点 1 -> 节点 2 -> 节点 3
- 节点 1 -> 节点 4 -> 节点 5
- 节点 4 -> 节点 7 -> 节点 8

### 可并行分支

- 节点 6 可在节点 3 和节点 4 完成后开展
- 节点 7 可在节点 4 完成后开展

### 收尾

- 节点 9 依赖所有前置节点完成

## 5. 节点计划

## 节点 1：角色与数据模型基础

### 目标

建立后续所有功能依赖的数据基础与角色基础。

### 范围

- 扩展 `users` 表基础字段
- 新建邀请链接表
- 新建邀请奖励流水表
- 新建发票资料 / 发票申请 / 发票文档表
- 为 `payment_orders` 增加归属与奖励快照字段
- 新增 `sales` 角色常量与基础枚举

### 不做

- 不接注册流程
- 不做前端页面
- 不做实际奖励发放

### 主要改动

- Ent schema
- migration SQL
- domain constants
- DTO / service model 基础结构

### 依赖

- 无

### 验收标准

- 数据库迁移可正常执行
- 新表与新字段命名符合需求文档
- `sales` 角色在后端模型中可识别
- 现有基础查询不因新增字段报错

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...
```

---

## 节点 2：邀请链接与注册归属继承

### 目标

让销售和普通用户都能生成长期有效的邀请链接，并让新用户在注册时正确写入邀请归属与销售归属。

### 范围

- 用户/销售生成邀请链接
- 邀请链接查询与禁用/作废
- 注册流程解析邀请链接
- 写入 `invited_by_user_id`
- 继承 `owner_sales_id`
- 覆盖邮箱注册、OAuth 注册、补全注册流程

### 不做

- 不做邀请奖励
- 不做管理员手工归属调整

### 主要改动

- auth/register 相关 handler
- OAuth complete-registration 相关 handler
- referral/invite service
- 用户侧“我的邀请链接”接口

### 依赖

- 节点 1

### 验收标准

- 普通用户可生成邀请链接
- 销售可生成邀请链接
- 链接默认长期有效
- 新用户通过邀请链接注册后，正确写入直属邀请人
- 新用户正确继承邀请人的 `owner_sales_id`
- 不带邀请链接注册时，系统行为符合默认规则

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...

cd /home/heligong/Downloads/sub2api/frontend
pnpm test:run
```

---

## 节点 3：管理员归属调整与子树迁移

### 目标

实现管理员修改邀请归属与销售归属，并确保销售归属对子树正确重算或迁移。

### 范围

- 管理员修改直属邀请人
- 防环校验
- 邀请归属修改后，自动重算目标用户整棵子树的 `owner_sales_id`
- 管理员直接发起销售归属整棵子树迁移
- 迁移预览影响人数
- 审计日志记录

### 不做

- 不做销售后台
- 不做邀请奖励发放

### 主要改动

- admin handler / service
- tree traversal / subtree rebuild logic
- audit log

### 依赖

- 节点 1
- 节点 2

### 验收标准

- 只有管理员可以修改邀请归属
- 只有管理员可以修改销售归属
- 修改邀请归属时不能产生环
- 修改邀请归属后，目标用户整棵子树销售归属重算正确
- 修改销售归属时，整棵子树迁移正确
- 审计日志包含操作者、影响节点数、前后归属

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...
```

---

## 节点 4：订单快照与用户充值记录

### 目标

为邀请奖励和开票提供稳定的订单数据基础，并让用户可以查看自己的充值记录。

### 范围

- 在订单上固化邀请/销售归属快照
- 固化邀请奖励触发状态
- 新增用户自己的充值记录接口
- 用户订单列表页/订单详情中的开票入口预留

### 不做

- 不做奖励发放
- 不接百望云

### 主要改动

- `payment_orders` 写入快照
- 用户订单查询接口
- 用户订单列表前端页面

### 依赖

- 节点 1

### 验收标准

- 新订单生成后包含归属快照字段
- 历史订单查询不受影响
- 用户可以查看自己的充值记录
- 用户只能查看自己的订单
- 订单具备后续开票所需的最小信息

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...

cd /home/heligong/Downloads/sub2api/frontend
pnpm test:run
```

---

## 节点 5：邀请奖励引擎

### 目标

实现“仅直接邀请人、首充一次性奖励”的完整后端闭环。

### 范围

- 识别被邀请用户是否发生首充
- 仅直接邀请人获得奖励
- 同一被邀请用户只触发一次
- 写入邀请奖励流水
- 退款后奖励冲回/作废
- 用户查看自己的邀请奖励记录

### 不做

- 不做多级奖励
- 不做销售返佣

### 主要改动

- payment completion hook
- invite reward service
- user reward query API

### 依赖

- 节点 2
- 节点 4

### 验收标准

- 首充成功后仅直接邀请人获得奖励
- 同一被邀请用户不会重复奖励
- 非首充订单不触发奖励
- 退款后奖励状态正确更新
- 用户可以查看自己的奖励记录

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...
```

---

## 节点 6：销售数据域与销售后台

### 目标

建立 `sales` 角色的独立数据视图和页面，而不是把现有管理员后台做裁剪。

### 范围

- `sales` 登录后的路由与菜单
- 销售仪表盘
- 销售客户列表
- 客户订单列表
- 客户开票列表
- 后端销售数据域过滤

### 不做

- 不做销售返佣
- 不做管理员能力复用

### 主要改动

- 前端角色判断从 `isAdmin` 扩展到角色白名单
- 新增 `/sales/*` 页面与接口
- 后端新增 sales query endpoints

### 依赖

- 节点 3
- 节点 4

### 验收标准

- 销售登录后看到销售菜单而非管理员菜单
- 销售只能看到自己 `owner_sales_id` 范围内的客户
- 销售不能通过参数越权访问其他客户订单或发票
- 管理员现有菜单不被破坏

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...

cd /home/heligong/Downloads/sub2api/frontend
pnpm test:run
```

---

## 节点 7：用户发票资料与开票申请

### 目标

让用户可基于自己的已支付订单发起开票申请，并查看申请状态。

### 范围

- 发票资料 CRUD
- 用户选择订单发起开票
- 开票申请状态查询
- 管理员查看全量开票申请

### 不做

- 不直接调用百望云
- 不做自动开票

### 主要改动

- invoice profile service
- invoice request service
- user invoice pages
- admin invoice list page

### 依赖

- 节点 4

### 验收标准

- 用户可维护自己的发票资料
- 用户只能对自己的已支付订单申请开票
- 同一订单的重复开票规则明确且可控
- 管理员能看到全量开票申请

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...

cd /home/heligong/Downloads/sub2api/frontend
pnpm test:run
```

---

## 节点 8：百望云异步开票集成

### 目标

完成百望云开票接入，并保证失败不影响支付主链路。

### 范围

- 百望云客户端封装
- 异步开票任务
- 成功回写发票号与文档记录
- 失败原因落库
- 管理员重试开票

### 不做

- 不做首版自动红冲/作废

### 主要改动

- invoice provider adapter
- worker / retry path
- admin retry endpoint

### 依赖

- 节点 7

### 验收标准

- 开票请求可异步投递到百望云
- 百望云失败不会影响订单支付成功状态
- 开票成功后能回写发票信息
- 开票失败后管理员可以重试
- 销售可查看自己客户的开票结果

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...
```

---

## 节点 9：回归加固与发布前验收

### 目标

在业务链条成型后，补齐跨模块回归、权限穿透测试和发布前核对。

### 范围

- 端到端用例补齐
- 权限穿透检查
- 邀请链/销售归属链一致性校验
- 奖励与退款回滚校验
- 开票失败重试校验
- 文档同步更新

### 依赖

- 节点 1 至 节点 8

### 验收标准

- 用户注册 -> 邀请链 -> 销售归属 -> 首充奖励 -> 开票 全链路跑通
- 销售无越权
- 管理员归属调整后树结构结果正确
- 退款后奖励状态正确
- 开票失败与重试路径正确
- 需求文档、实施计划文档、操作文档保持一致

### 最小验证

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...

cd /home/heligong/Downloads/sub2api/frontend
pnpm test:run
```

## 6. 建议 PR 切分

建议按以下节奏切分：

1. PR-1：节点 1
2. PR-2：节点 2
3. PR-3：节点 3
4. PR-4：节点 4
5. PR-5：节点 5
6. PR-6：节点 6
7. PR-7：节点 7
8. PR-8：节点 8
9. PR-9：节点 9

如果资源有限，可合并为 6 个 PR：

1. 基础模型与角色
2. 邀请链与归属继承
3. 管理员归属调整
4. 用户订单记录与邀请奖励
5. 销售后台与数据域
6. 开票与百望云

## 7. 风险提示

### 高风险点

- 邀请归属修改导致树结构异常
- 销售数据域过滤遗漏导致越权
- 首充奖励重复发放
- 退款与奖励回滚不一致
- 百望云失败拖垮支付链路

### 控制策略

- 邀请树操作统一走 service，不允许 handler 直接拼 SQL
- 销售接口全部新增，不复用管理员全量接口做前端裁剪
- 奖励判定依赖订单快照与唯一触发标记
- 开票必须异步，不进入支付 webhook 主链路

## 8. 后续文档建议

本计划确认后，建议继续补三份文档：

1. 数据库字段清单
2. 后端 API 清单
3. 前端页面清单

后续所有节点排期与任务分配，建议以本文件为准继续细化。
