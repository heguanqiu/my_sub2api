# 最终需求清单 v4 测试数据初始化清单

## 1. 文档目的

本文件用于为后续 agent 提供统一、可复用的测试数据基线，保证：

- 节点执行不中断
- 每个节点都有稳定的基础账号和样本数据
- 真实 API 验证可以复用同一套数据准备逻辑
- 支付与开票例外链路也有固定测试桩数据

关联文档：

- [docs/requirements/final-requirements-v4.md](/home/heligong/Downloads/sub2api/docs/requirements/final-requirements-v4.md)
- [docs/plans/final-requirements-v4-detailed-node-plan.md](/home/heligong/Downloads/sub2api/docs/plans/final-requirements-v4-detailed-node-plan.md)
- [docs/implementation/final-requirements-v4-validation-templates.md](/home/heligong/Downloads/sub2api/docs/implementation/final-requirements-v4-validation-templates.md)

## 2. 总体原则

### 2.1 优先使用同一批固定数据

不要每个节点都重新随机构造测试用户，统一使用固定命名的数据集：

- 管理员
- 销售
- 普通用户
- 邀请链
- 奖励触发样本
- 支付订单样本
- 开票资料样本

### 2.2 真实 API 优先

除支付和开票外，初始化过程优先使用真实 API 创建数据。

允许直接数据库操作的场景：

- 功能尚未实现，无法通过 API 造数
- 需要快速把现有普通用户升级成 `sales`
- 需要构造复杂历史状态用于回归

### 2.3 数据要可重建

推荐使用以下任一模式：

- 模式 A：每次从干净数据库开始
- 模式 B：使用固定邮箱 / 固定 code，重复运行前先清理同名数据

推荐对 agent 使用模式 A，最稳定。

## 3. 环境基线

## 3.1 基础环境变量

```bash
export BASE_URL="http://127.0.0.1:8080/api/v1"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="admin123456"

export SALES_A_EMAIL="sales.a@example.com"
export SALES_B_EMAIL="sales.b@example.com"
export SALES_PASSWORD="sales123456"

export USER_A_EMAIL="user.a@example.com"
export USER_B_EMAIL="user.b@example.com"
export USER_C_EMAIL="user.c@example.com"
export USER_D_EMAIL="user.d@example.com"
export USER_E_EMAIL="user.e@example.com"
export USER_F_EMAIL="user.f@example.com"
export USER_PASSWORD="user123456"
```

## 3.2 推荐数据隔离标识

为避免和其他测试混淆，推荐统一前缀：

- 邮箱前缀：`v4.`
- 邀请链接 code 前缀：`v4_`
- 发票资料 title 前缀：`V4-`

例如：

- `v4.sales.a@example.com`
- `v4.user.a@example.com`
- `v4_invite_sales_a`

## 4. 核心测试数据集

## 4.1 账号集合

### 管理员

- `ADMIN_1`
  - email: `admin@example.com`
  - role: `admin`
  - 用途：全量管理、归属调整、系统设置

### 销售

- `SALES_A`
  - email: `sales.a@example.com`
  - role: `sales`
  - 用途：查看自己名下客户

- `SALES_B`
  - email: `sales.b@example.com`
  - role: `sales`
  - 用途：权限隔离对照组

### 普通用户

- `USER_A`
  - 直属销售归属：`SALES_A`
  - 用途：由销售邀请注册的第一层用户

- `USER_B`
  - 直属邀请人：`USER_A`
  - 销售归属：`SALES_A`
  - 用途：直接邀请奖励验证链

- `USER_C`
  - 直属邀请人：`USER_B`
  - 销售归属：`SALES_A`
  - 用途：多级链路但不应给 `USER_A` 奖励

- `USER_D`
  - 直属销售归属：`SALES_B`
  - 用途：销售隔离对照组

- `USER_E`
  - 无销售归属
  - 用途：验证“邀请人无销售归属时，新用户也无销售归属”

- `USER_F`
  - 直属邀请人：`USER_E`
  - 无销售归属
  - 用途：无销售归属链路验证

## 4.2 邀请链样本

### 链 1：销售 A 主链

```text
SALES_A -> USER_A -> USER_B -> USER_C
```

用途：

- 邀请链查询
- 销售归属继承
- 直接邀请奖励验证
- 子树迁移验证

### 链 2：销售 B 对照链

```text
SALES_B -> USER_D
```

用途：

- 销售数据域隔离
- 销售越权验证

### 链 3：无销售归属链

```text
USER_E -> USER_F
```

用途：

- 无销售归属继承验证

## 4.3 邀请链接样本

### 销售邀请链接

- `INVITE_LINK_SALES_A_ACTIVE`
  - 创建者：`SALES_A`
  - 状态：`active`

- `INVITE_LINK_SALES_A_DISABLED`
  - 创建者：`SALES_A`
  - 状态：`disabled`

- `INVITE_LINK_SALES_B_ACTIVE`
  - 创建者：`SALES_B`
  - 状态：`active`

### 用户邀请链接

- `INVITE_LINK_USER_A_ACTIVE`
  - 创建者：`USER_A`
  - 状态：`active`

- `INVITE_LINK_USER_B_ACTIVE`
  - 创建者：`USER_B`
  - 状态：`active`

## 4.4 订单样本

### 订单分组

- `ORDER_USER_B_FIRST_PAID`
  - 用户：`USER_B`
  - 状态：已支付
  - 作用：触发 `USER_A` 首充奖励

- `ORDER_USER_B_SECOND_PAID`
  - 用户：`USER_B`
  - 状态：已支付
  - 作用：验证不重复发奖励

- `ORDER_USER_B_FIRST_REFUNDED`
  - 用户：`USER_B`
  - 作用：验证奖励冲回

- `ORDER_USER_C_FIRST_PAID`
  - 用户：`USER_C`
  - 状态：已支付
  - 作用：只给 `USER_B` 奖励，不给 `USER_A`

- `ORDER_USER_D_FIRST_PAID`
  - 用户：`USER_D`
  - 状态：已支付
  - 作用：销售数据域隔离中的订单对照

## 4.5 发票资料样本

### USER_B 发票资料

- `PROFILE_USER_B_DEFAULT`
  - user: `USER_B`
  - title: `V4 USER_B COMPANY`
  - tax_no: `91310000TESTUSERB01`
  - is_default: `true`

### USER_D 发票资料

- `PROFILE_USER_D_DEFAULT`
  - user: `USER_D`
  - title: `V4 USER_D COMPANY`
  - tax_no: `91310000TESTUSERD01`
  - is_default: `true`

## 4.6 开票申请样本

- `INVOICE_REQ_USER_B_SUCCESS`
  - 对应订单：`ORDER_USER_B_FIRST_PAID`
  - 状态：`issued`

- `INVOICE_REQ_USER_D_FAILED`
  - 对应订单：`ORDER_USER_D_FIRST_PAID`
  - 状态：`failed`
  - 用途：管理员重试

## 5. 初始化顺序

推荐顺序必须固定，否则后续链路会断。

### Step 1：准备管理员

- 完成系统初始化
- 确保管理员可登录

### Step 2：创建基础用户

使用管理员接口创建：

- `SALES_A`
- `SALES_B`
- `USER_A`
- `USER_D`
- `USER_E`

说明：

- 在 `sales` 角色接口尚未完成前，可先通过管理员创建普通用户，再通过数据库或管理员扩展接口把角色调整为 `sales`

### Step 3：设置销售角色

如果此时已有角色编辑接口：

- 把 `SALES_A`
- 把 `SALES_B`

改为 `sales`

如果没有角色编辑接口：

- 允许通过数据库直接更新 `users.role = 'sales'`

### Step 4：建立初始销售归属

通过管理员方式设置：

- `USER_A.owner_sales_id = SALES_A`
- `USER_D.owner_sales_id = SALES_B`
- `USER_E.owner_sales_id = null`

### Step 5：生成邀请链接

为以下用户生成并保存默认邀请链接：

- `SALES_A`
- `SALES_B`
- `USER_A`
- `USER_B`（后续创建后）

### Step 6：注册链路造数

通过真实注册 API 使用邀请链接注册：

- `USER_B` 通过 `USER_A` 邀请链接注册
- `USER_C` 通过 `USER_B` 邀请链接注册
- `USER_F` 通过 `USER_E` 邀请链接注册

### Step 7：支付测试数据

使用测试支付路径、测试 provider 或模拟 webhook 生成：

- `ORDER_USER_B_FIRST_PAID`
- `ORDER_USER_B_SECOND_PAID`
- `ORDER_USER_C_FIRST_PAID`
- `ORDER_USER_D_FIRST_PAID`

### Step 8：发票资料数据

通过真实 API 创建：

- `PROFILE_USER_B_DEFAULT`
- `PROFILE_USER_D_DEFAULT`

### Step 9：开票测试数据

通过真实开票申请 API + mock/stub/sandbox：

- 创建成功样本
- 创建失败样本

## 6. 推荐初始化方式

## 6.1 方式 A：完全 API 初始化

适用于：

- 功能已实现
- 角色编辑已有 API

优点：

- 最接近真实行为
- 可直接作为验收过程

缺点：

- 初始化步骤较长

## 6.2 方式 B：API + 最小数据库辅助

适用于：

- `sales` 角色接口尚未完成
- 归属迁移逻辑尚未完成

允许直接数据库操作的最小范围：

- 把用户角色改为 `sales`
- 回填 `owner_sales_id`
- 回填 `invited_by_user_id`

其余行为尽量仍通过 API 构造。

## 7. 每个节点建议复用的数据

## A 阶段

只需要：

- `ADMIN_1`

## B 阶段

需要：

- `ADMIN_1`
- `SALES_A`
- `SALES_B`
- `USER_A`

## C 阶段

需要：

- `ADMIN_1`
- `SALES_A`
- `SALES_B`
- `USER_A`
- `USER_B`
- `USER_C`
- `USER_D`
- `USER_E`
- `USER_F`

## D 阶段

需要：

- `USER_B`
- `USER_C`
- `USER_D`
- 支付订单样本

## E 阶段

需要：

- `USER_A`
- `USER_B`
- `USER_C`
- 首充和非首充订单样本

## F 阶段

需要：

- `SALES_A`
- `SALES_B`
- `USER_A`
- `USER_B`
- `USER_C`
- `USER_D`

## G 阶段

需要：

- `ADMIN_1`
- `USER_B`
- `USER_D`
- 已支付订单样本
- 发票资料样本
- 百望云 mock/stub/sandbox

## 8. 推荐清理策略

如果不是全新数据库，每次跑前建议清理以下对象：

- `invite_links`
- `invite_reward_ledger`
- `invoice_documents`
- `invoice_requests`
- `invoice_profiles`
- 与测试邮箱相关的 `users`
- 与这些用户关联的 `payment_orders`

注意：

- 清理顺序必须从子表到父表
- 不要清空真实管理员账号

## 9. 推荐输出物

后续 agent 执行初始化时，建议产出：

- `test-data-bootstrap.log`
- `tokens.env`（如安全允许，仅本地）
- `seed-summary.md`

## 10. 进一步建议

如果你要让 agent 更稳定持续工作，下一步最值得补的是：

- 一份 `test-data-bootstrap.sh` 脚本草案
- 一份 `tokens.env.example`

这样它就不是“清单”，而是可以半自动执行的初始化工具。
