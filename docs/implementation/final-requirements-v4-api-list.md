# 最终需求清单 v4 API 清单

## 1. 文档目的

本文件用于定义 v4 所需新增或扩展的 API，供后端、前端、测试和 agent 执行时对齐。

上游文档：

- [docs/requirements/final-requirements-v4.md](/home/heligong/Downloads/sub2api/docs/requirements/final-requirements-v4.md)
- [docs/plans/final-requirements-v4-detailed-node-plan.md](/home/heligong/Downloads/sub2api/docs/plans/final-requirements-v4-detailed-node-plan.md)

## 2. 命名与风格约束

- 复用现有 `/api/v1/...` 路由风格
- 销售能力独立放在 `/sales/*`
- 管理员能力放在 `/admin/*`
- 开票配置不放在 `/admin/payment/config`，统一放在 `/admin/settings`
- 用户订单仍沿用现有 `/payment/orders/*`

## 3. 用户侧 API

## 3.1 邀请链接

### `GET /api/v1/referral/my-link`

- 认证：`user | sales`
- 作用：获取当前用户默认邀请链接
- 响应建议：
  - `code`
  - `url`
  - `status`
  - `owner_sales_id`

### `POST /api/v1/referral/my-link/regenerate`

- 认证：`user | sales`
- 作用：重生成默认邀请链接
- 请求体：可空
- 响应建议：
  - `code`
  - `url`
  - `status`

### `POST /api/v1/referral/my-link/disable`

- 认证：`user | sales`
- 作用：禁用自己的邀请链接

### `POST /api/v1/referral/my-link/revoke`

- 认证：`user | sales`
- 作用：作废自己的邀请链接

## 3.2 我的邀请关系

### `GET /api/v1/referral/my-invitees`

- 认证：`user | sales`
- 作用：查看当前用户直接邀请的用户
- 查询参数：
  - `page`
  - `page_size`
  - `search`

### `GET /api/v1/referral/my-rewards`

- 认证：`user | sales`
- 作用：查看当前用户的邀请奖励记录
- 查询参数：
  - `page`
  - `page_size`
  - `status`

## 3.3 用户充值记录

### `GET /api/v1/payment/orders/my`

- 认证：`user | sales | admin`
- 备注：现有接口扩字段
- 需新增返回字段：
  - `invoice_status`
  - `invoice_request_id`
  - `can_request_invoice`
  - `invite_reward_status`

### `GET /api/v1/payment/orders/:id`

- 认证：`user | sales | admin`
- 备注：现有接口扩字段
- 行为：
  - 用户只能看自己的订单
  - 销售不能走该接口看别人的订单，销售看客户订单走 sales API

## 3.4 发票资料

### `GET /api/v1/invoice-profiles`

- 认证：`user`
- 作用：获取当前用户发票资料列表

### `POST /api/v1/invoice-profiles`

- 认证：`user`
- 作用：创建发票资料

### `PUT /api/v1/invoice-profiles/:id`

- 认证：`user`
- 作用：更新当前用户自己的发票资料

### `DELETE /api/v1/invoice-profiles/:id`

- 认证：`user`
- 作用：删除当前用户自己的发票资料

### `POST /api/v1/invoice-profiles/:id/set-default`

- 认证：`user`
- 作用：设置默认发票资料

## 3.5 用户开票申请

### `POST /api/v1/invoices`

- 认证：`user`
- 作用：基于自己的已支付订单申请开票
- 请求体建议：
  - `order_id`
  - `profile_id`

### `GET /api/v1/invoices/my`

- 认证：`user`
- 作用：查看自己的开票申请记录

### `GET /api/v1/invoices/:id`

- 认证：`user`
- 作用：查看自己的单条开票记录

## 4. 销售侧 API

## 4.1 销售仪表盘

### `GET /api/v1/sales/dashboard`

- 认证：`sales`
- 作用：查看归属范围内客户统计

## 4.2 销售客户

### `GET /api/v1/sales/customers`

- 认证：`sales`
- 作用：查看自己归属范围内的客户
- 查询参数：
  - `page`
  - `page_size`
  - `search`
  - `status`

### `GET /api/v1/sales/customers/:id`

- 认证：`sales`
- 作用：查看单个客户详情
- 限制：目标用户必须属于当前销售

## 4.3 销售客户订单

### `GET /api/v1/sales/customers/:id/orders`

- 认证：`sales`
- 作用：查看客户订单
- 限制：目标用户必须属于当前销售

## 4.4 销售客户开票

### `GET /api/v1/sales/customers/:id/invoices`

- 认证：`sales`
- 作用：查看客户开票记录
- 限制：目标用户必须属于当前销售

## 5. 管理员侧 API

## 5.1 邀请树与归属管理

### `GET /api/v1/admin/referrals/tree/:user_id`

- 认证：`admin`
- 作用：查看某用户为根的邀请树

### `POST /api/v1/admin/users/:id/change-inviter`

- 认证：`admin`
- 作用：修改直属邀请人
- 请求体建议：
  - `new_invited_by_user_id`

### `POST /api/v1/admin/users/:id/recompute-sales-owner`

- 认证：`admin`
- 作用：重算当前节点整棵子树的销售归属

### `POST /api/v1/admin/users/:id/migrate-sales-owner/preview`

- 认证：`admin`
- 作用：预览整棵子树迁移影响
- 请求体建议：
  - `target_sales_user_id`

### `POST /api/v1/admin/users/:id/migrate-sales-owner`

- 认证：`admin`
- 作用：执行整棵子树销售归属迁移
- 请求体建议：
  - `target_sales_user_id`

## 5.2 邀请奖励管理

### `GET /api/v1/admin/invite-rewards`

- 认证：`admin`
- 作用：查看全量奖励流水

### `GET /api/v1/admin/invite-rewards/:id`

- 认证：`admin`
- 作用：查看单条奖励流水

## 5.3 开票管理

### `GET /api/v1/admin/invoices`

- 认证：`admin`
- 作用：查看全量开票申请

### `GET /api/v1/admin/invoices/:id`

- 认证：`admin`
- 作用：查看开票详情

### `POST /api/v1/admin/invoices/:id/retry`

- 认证：`admin`
- 作用：重试开票

## 5.4 系统设置中的开票配置

### `GET /api/v1/admin/settings`

- 认证：`admin`
- 备注：现有接口扩字段
- 需新增开票配置字段：
  - `invoice_enabled`
  - `invoice_provider`
  - `invoice_baiwang_enabled`
  - `invoice_baiwang_base_url`
  - `invoice_baiwang_app_key`
  - `invoice_baiwang_app_secret_configured`
  - `invoice_baiwang_taxpayer_id`
  - `invoice_baiwang_seller_name`
  - `invoice_baiwang_default_goods_name`
  - `invoice_auto_retry_enabled`
  - `invoice_retry_limit`

### `PUT /api/v1/admin/settings`

- 认证：`admin`
- 备注：现有接口扩字段
- 作用：通过系统设置更新开票配置

## 6. 内部任务 / 服务层动作

这些不一定暴露为外部 API，但必须在实现层存在：

- 归属树遍历服务
- 归属树防环校验服务
- 首充识别服务
- 奖励幂等服务
- 退款冲回服务
- 百望云适配层
- 开票异步任务执行器

## 7. 建议返回字段规范

### 用户列表最少字段

- `id`
- `email`
- `role`
- `invited_by_user_id`
- `owner_sales_id`

### 订单列表最少字段

- `id`
- `status`
- `amount`
- `pay_amount`
- `payment_type`
- `created_at`
- `paid_at`
- `invoice_status`
- `can_request_invoice`

### 奖励流水最少字段

- `id`
- `inviter_user_id`
- `invitee_user_id`
- `trigger_order_id`
- `reward_amount`
- `status`
- `created_at`

### 开票记录最少字段

- `id`
- `order_id`
- `status`
- `provider`
- `provider_invoice_id`
- `fail_reason`
- `retry_count`
- `created_at`
- `issued_at`

## 8. 权限要求汇总

- `user`
  - 只能访问自己的邀请、奖励、订单、发票

- `sales`
  - 只能访问 `owner_sales_id = 当前销售` 的用户及其订单、发票
  - 不能访问管理员全量接口

- `admin`
  - 可访问全量接口
  - 可改邀请归属与销售归属

## 9. API 实现优先级

### P0

- 用户邀请链接
- 用户直属邀请列表
- 管理员修改邀请归属
- 管理员销售归属迁移
- 用户充值记录

### P1

- 用户奖励记录
- 销售客户与订单查询
- 发票资料 CRUD
- 用户开票申请

### P2

- 管理员开票管理
- 系统设置中的开票配置
- 销售客户开票查看

## 10. 验收标准

- 命名风格与现有路由分组一致
- 销售 API 与管理员 API 明确隔离
- 开票配置确实挂在 `/admin/settings`，不另建配置入口
- 所有默认非支付/非开票 API 都可以通过真实 API 直接验证
