# 最终需求清单 v4 数据库字段清单

## 1. 文档目的

本文件用于把 v4 需求映射成数据库层面的可执行字段清单，作为：

- schema 设计基线
- migration 编写基线
- repository / service 层实现基线

上游需求文档：

- [docs/requirements/final-requirements-v4.md](/home/heligong/Downloads/sub2api/docs/requirements/final-requirements-v4.md)
- [docs/plans/final-requirements-v4-detailed-node-plan.md](/home/heligong/Downloads/sub2api/docs/plans/final-requirements-v4-detailed-node-plan.md)

## 2. 设计原则

- 优先新增表和新增字段，少重写现有老表语义
- 能做快照的地方做快照，避免归属变更污染历史账目
- 能独立建模的业务域不要挤进 `users` 或 `payment_orders` 的大 JSON
- 所有高频过滤字段都需要索引

## 3. 现有表扩展

## 3.1 `users`

### 新增字段

- `role`
  - 类型：`varchar(20)`
  - 说明：现有字段保留，扩展允许值 `admin | sales | user`
  - 默认：`user`

- `invited_by_user_id`
  - 类型：`bigint null`
  - 说明：直属邀请人，仅记录一级邀请关系
  - 外键：`users.id`
  - 删除策略：建议 `SET NULL`

- `owner_sales_id`
  - 类型：`bigint null`
  - 说明：当前用户最终归属销售
  - 外键：`users.id`
  - 约束：引用用户必须是 `role='sales'`
  - 删除策略：建议限制删除销售，先迁移归属

- `first_paid_order_id`
  - 类型：`bigint null`
  - 说明：记录该用户命中的首充订单
  - 外键：`payment_orders.id`

- `first_paid_at`
  - 类型：`timestamptz null`
  - 说明：记录首充完成时间

### 建议索引

- `idx_users_invited_by_user_id`
- `idx_users_owner_sales_id`
- `idx_users_role_owner_sales_id`

### 备注

- `role` 建议保留在 `users` 主表，不拆到 profile 表
- `sales` 用户可复用现有登录体系

---

## 3.2 `payment_orders`

### 新增字段

- `owner_sales_id_snapshot`
  - 类型：`bigint null`
  - 说明：下单时快照的销售归属

- `invited_by_user_id_snapshot`
  - 类型：`bigint null`
  - 说明：下单时快照的直属邀请人

- `invite_reward_status`
  - 类型：`varchar(30)`
  - 默认：`not_applicable`
  - 建议值：
    - `not_applicable`
    - `pending_evaluation`
    - `granted`
    - `reversed`
    - `skipped`

- `invite_reward_ledger_id`
  - 类型：`bigint null`
  - 说明：关联奖励流水

- `invoice_status`
  - 类型：`varchar(30)`
  - 默认：`not_requested`
  - 建议值：
    - `not_requested`
    - `requested`
    - `processing`
    - `issued`
    - `failed`
    - `voided`

- `invoice_request_id`
  - 类型：`bigint null`
  - 说明：最近一次开票申请 ID

### 建议索引

- `idx_payment_orders_owner_sales_id_snapshot`
- `idx_payment_orders_invited_by_user_id_snapshot`
- `idx_payment_orders_invite_reward_status`
- `idx_payment_orders_invoice_status`
- `idx_payment_orders_user_id_paid_at`

### 备注

- 归属快照必须在订单创建时写入
- 不建议在查询时动态 join `users.owner_sales_id` 替代快照

## 4. 新增表

## 4.1 `invite_links`

### 用途

用于销售和普通用户生成长期有效的邀请链接。

### 字段

- `id`
  - 类型：`bigserial primary key`

- `code`
  - 类型：`varchar(128) not null unique`
  - 说明：用于生成邀请 URL 的 token

- `created_by_user_id`
  - 类型：`bigint not null`
  - 说明：谁创建了这条邀请链接

- `creator_role`
  - 类型：`varchar(20) not null`
  - 说明：创建时的角色快照，值为 `sales | user | admin`

- `owner_sales_id`
  - 类型：`bigint null`
  - 说明：创建时绑定的销售归属快照

- `status`
  - 类型：`varchar(20) not null`
  - 默认：`active`
  - 建议值：
    - `active`
    - `disabled`
    - `revoked`

- `notes`
  - 类型：`text null`

- `created_at`
  - 类型：`timestamptz not null`

- `updated_at`
  - 类型：`timestamptz not null`

### 索引

- `idx_invite_links_created_by_user_id`
- `idx_invite_links_owner_sales_id`
- `idx_invite_links_status`

### 备注

- 邀请链接默认不设置过期时间
- 若后续需要临时活动链接，再单独加 `expires_at`

---

## 4.2 `invite_reward_ledger`

### 用途

记录“仅直接邀请人”的首充一次性奖励。

### 字段

- `id`
  - 类型：`bigserial primary key`

- `inviter_user_id`
  - 类型：`bigint not null`

- `invitee_user_id`
  - 类型：`bigint not null`

- `trigger_order_id`
  - 类型：`bigint not null`

- `reward_type`
  - 类型：`varchar(20) not null`
  - 建议值：`balance`

- `reward_amount`
  - 类型：`decimal(20,8) not null`

- `status`
  - 类型：`varchar(30) not null`
  - 建议值：
    - `pending`
    - `granted`
    - `reversed`
    - `cancelled`

- `reason`
  - 类型：`text null`

- `created_at`
  - 类型：`timestamptz not null`

- `confirmed_at`
  - 类型：`timestamptz null`

- `reversed_at`
  - 类型：`timestamptz null`

### 索引

- `idx_invite_reward_ledger_inviter_user_id`
- `idx_invite_reward_ledger_invitee_user_id`
- `idx_invite_reward_ledger_trigger_order_id`
- `idx_invite_reward_ledger_status`

### 唯一约束

- `uniq_invite_reward_ledger_invitee_user_id`
  - 目的：同一被邀请用户只允许产生一条首充奖励流水

### 备注

- 这是防止重复奖励的核心约束之一

---

## 4.3 `invoice_profiles`

### 用途

保存用户的开票资料。

### 字段

- `id`
  - 类型：`bigserial primary key`

- `user_id`
  - 类型：`bigint not null`

- `title`
  - 类型：`varchar(255) not null`

- `tax_no`
  - 类型：`varchar(100) null`

- `email`
  - 类型：`varchar(255) null`

- `phone`
  - 类型：`varchar(50) null`

- `address`
  - 类型：`text null`

- `bank_name`
  - 类型：`varchar(255) null`

- `bank_account`
  - 类型：`varchar(255) null`

- `invoice_type`
  - 类型：`varchar(30) not null`
  - 建议值：
    - `personal_electronic`
    - `enterprise_electronic`

- `is_default`
  - 类型：`boolean not null default false`

- `created_at`
  - 类型：`timestamptz not null`

- `updated_at`
  - 类型：`timestamptz not null`

### 索引

- `idx_invoice_profiles_user_id`
- `idx_invoice_profiles_user_id_is_default`

### 备注

- 一个用户建议只允许一个默认资料
- 可通过应用层保证，也可加部分唯一索引

---

## 4.4 `invoice_requests`

### 用途

记录用户对订单发起的开票申请与状态机。

### 字段

- `id`
  - 类型：`bigserial primary key`

- `user_id`
  - 类型：`bigint not null`

- `order_id`
  - 类型：`bigint not null`

- `profile_id`
  - 类型：`bigint not null`

- `status`
  - 类型：`varchar(30) not null`
  - 建议值：
    - `requested`
    - `processing`
    - `issued`
    - `failed`
    - `voided`

- `provider`
  - 类型：`varchar(50) not null`
  - 首版固定：`baiwang`

- `provider_request_id`
  - 类型：`varchar(255) null`

- `provider_invoice_id`
  - 类型：`varchar(255) null`

- `fail_reason`
  - 类型：`text null`

- `retry_count`
  - 类型：`integer not null default 0`

- `requested_at`
  - 类型：`timestamptz not null`

- `issued_at`
  - 类型：`timestamptz null`

- `created_at`
  - 类型：`timestamptz not null`

- `updated_at`
  - 类型：`timestamptz not null`

### 索引

- `idx_invoice_requests_user_id`
- `idx_invoice_requests_order_id`
- `idx_invoice_requests_status`
- `idx_invoice_requests_provider`

### 唯一约束

- `uniq_invoice_requests_order_id_active`
  - 建议通过应用层或条件唯一控制一笔订单不能在未完成状态下重复申请

### 备注

- 首版建议一笔订单只允许一条有效开票申请

---

## 4.5 `invoice_documents`

### 用途

保存发票成功后的票据结果。

### 字段

- `id`
  - 类型：`bigserial primary key`

- `invoice_request_id`
  - 类型：`bigint not null`

- `invoice_no`
  - 类型：`varchar(100) null`

- `invoice_code`
  - 类型：`varchar(100) null`

- `file_url`
  - 类型：`text null`

- `file_type`
  - 类型：`varchar(20) null`
  - 建议值：
    - `pdf`
    - `ofd`
    - `xml`

- `raw_payload_summary`
  - 类型：`jsonb null`

- `created_at`
  - 类型：`timestamptz not null`

### 索引

- `idx_invoice_documents_invoice_request_id`
- `idx_invoice_documents_invoice_no`

## 5. 建议新增的系统设置键

开票配置要求进入现有系统设置，而不是独立配置页。

建议新增设置键：

- `invoice_enabled`
- `invoice_provider`
- `invoice_baiwang_enabled`
- `invoice_baiwang_base_url`
- `invoice_baiwang_app_key`
- `invoice_baiwang_app_secret`
- `invoice_baiwang_taxpayer_id`
- `invoice_baiwang_seller_name`
- `invoice_baiwang_default_goods_name`
- `invoice_auto_retry_enabled`
- `invoice_retry_limit`

## 6. 约束与回填建议

### 6.1 老用户数据

- 历史用户默认：
  - `invited_by_user_id = null`
  - `owner_sales_id = null`

### 6.2 历史订单数据

- 历史订单默认：
  - `owner_sales_id_snapshot = null`
  - `invited_by_user_id_snapshot = null`
  - `invite_reward_status = 'not_applicable'`
  - `invoice_status = 'not_requested'`

### 6.3 销售账号

- `sales` 用户建议继续复用 `users` 表
- 不建议单独拆 `sales_users` 主表

## 7. 字段实现优先级

### P0

- `users.invited_by_user_id`
- `users.owner_sales_id`
- `invite_links`
- `invite_reward_ledger`
- `payment_orders.owner_sales_id_snapshot`
- `payment_orders.invited_by_user_id_snapshot`
- `payment_orders.invite_reward_status`

### P1

- `users.first_paid_order_id`
- `users.first_paid_at`
- `payment_orders.invite_reward_ledger_id`
- `payment_orders.invoice_status`
- `payment_orders.invoice_request_id`

### P2

- `invoice_profiles`
- `invoice_requests`
- `invoice_documents`
- invoice system setting keys

## 8. 验收标准

- 所有新增字段与表均有明确迁移脚本
- 所有快照字段都能解释清楚写入时机
- 所有高频查询字段具备索引
- 奖励去重约束可从数据库层防止重复发放
- 开票配置可由系统设置统一表达
