# 最终需求清单 v4 验证模板

## 1. 文档目的

本文件为后续 agent 执行提供“按节点验收”的最小验证模板。

目标：

- 默认走真实 API 验证
- 不依赖真实浏览器
- 支付和开票可使用 mock/stub/sandbox
- 每个节点都有可复用的验证脚手架

## 2. 全局验证规则

### 2.1 默认真实 API

除支付和开票节点外，所有节点都按以下思路验证：

1. 启动本地后端服务
2. 如需前端接口契约，可启动前端或直接调用后端 API
3. 获取不同角色 token
4. 调用真实 API
5. 记录：
   - 请求
   - 响应
   - 二次查询结果

### 2.2 支付与开票例外

- 支付：
  - 允许用测试路径 / stub provider / 模拟 webhook
- 开票：
  - 允许用百望云 mock/stub/sandbox

但两者都必须验证系统内部状态流转。

## 3. 环境准备模板

### 3.1 启动服务

```bash
cd /home/heligong/Downloads/sub2api/backend
go test ./...
```

实际联调时建议增加：

```bash
cd /home/heligong/Downloads/sub2api/backend
go run ./cmd/server
```

### 3.2 基础变量模板

```bash
export BASE_URL="http://127.0.0.1:8080/api/v1"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="admin123"
export SALES_EMAIL="sales.a@example.com"
export SALES_PASSWORD="sales123456"
export USER_A_EMAIL="user.a@example.com"
export USER_A_PASSWORD="user123456"
export USER_B_EMAIL="user.b@example.com"
export USER_B_PASSWORD="user123456"
```

### 3.3 登录取 token 模板

```bash
curl -sS -X POST "$BASE_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"
```

建议后续将 token 存成环境变量：

```bash
export ADMIN_TOKEN="..."
export SALES_TOKEN="..."
export USER_A_TOKEN="..."
export USER_B_TOKEN="..."
```

## 4. 节点验证模板

## A1-A8 基础模型类节点

### 目标

验证迁移、编译、基础接口不破坏。

### 验证步骤

1. 运行后端测试
2. 启动服务
3. 调管理员基础查询接口
4. 调用户基础查询接口

### 验证命令模板

```bash
curl -sS "$BASE_URL/admin/settings" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

curl -sS "$BASE_URL/user/profile" \
  -H "Authorization: Bearer $USER_A_TOKEN"
```

### 验收点

- 接口不 500
- 新字段扩展不影响旧响应

## B1-B4 邀请链接管理

### 验证步骤

1. 登录普通用户
2. 获取邀请链接
3. 重生成邀请链接
4. 禁用或作废邀请链接
5. 再获取邀请链接确认状态变化

### 命令模板

```bash
curl -sS "$BASE_URL/referral/my-link" \
  -H "Authorization: Bearer $USER_A_TOKEN"

curl -sS -X POST "$BASE_URL/referral/my-link/regenerate" \
  -H "Authorization: Bearer $USER_A_TOKEN"
```

### 验收点

- 普通用户可生成链接
- 销售可生成链接
- 链接禁用后状态正确

## B5-B7 注册归属继承

### 验证步骤

1. 获取邀请链接
2. 带邀请参数调用注册接口
3. 用管理员接口查询新用户
4. 校验：
   - `invited_by_user_id`
   - `owner_sales_id`

### 验收点

- 新用户直属邀请人正确
- 新用户继承销售归属正确

## B8 直属邀请列表

### 验证步骤

1. 构造 A -> B -> C
2. 用 A 查直属邀请人
3. 用 B 查直属邀请人

### 验收点

- A 只看到 B
- B 只看到 C
- A 不直接看到 C

## C1-C9 管理员归属调整

### 验证步骤

1. 构造邀请树
2. 管理员查看邀请树
3. 管理员修改直属邀请人
4. 回查邀请树
5. 回查整棵子树 `owner_sales_id`
6. 管理员执行销售归属迁移
7. 回查整棵树

### 命令模板

```bash
curl -sS "$BASE_URL/admin/referrals/tree/123" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

curl -sS -X POST "$BASE_URL/admin/users/123/change-inviter" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"new_invited_by_user_id":456}'
```

### 验收点

- 非法修改被拒绝
- 合法修改后整棵子树归属正确
- 审计存在

## D1-D4 用户充值记录

### 验证步骤

1. 通过测试支付路径或现有支付测试机制生成已支付订单
2. 用户查询自己的订单列表
3. 另一个用户查询自己的订单列表
4. 校验隔离

### 命令模板

```bash
curl -sS "$BASE_URL/payment/orders/my" \
  -H "Authorization: Bearer $USER_A_TOKEN"
```

### 验收点

- 用户只看到自己的订单
- 订单返回开票相关字段

## E1-E7 邀请奖励

### 验证步骤

1. 构造 A 直接邀请 B
2. 让 B 完成首充
3. 查询 A 的奖励记录
4. 让 B 再次充值
5. 再查 A 的奖励记录
6. 如支持退款，执行退款后再查奖励状态

### 验收点

- 首充时发一次
- 再充值不重复发
- 只有直接邀请人 A 命中奖励

## F1-F9 销售数据域

### 验证步骤

1. 准备两个销售 `S1` 和 `S2`
2. 准备分别归属给 `S1` 和 `S2` 的客户
3. 用 `S1` 调客户列表
4. 用 `S1` 访问 `S2` 客户订单
5. 用 `S1` 访问 `S2` 客户开票

### 命令模板

```bash
curl -sS "$BASE_URL/sales/customers" \
  -H "Authorization: Bearer $SALES_TOKEN"

curl -sS "$BASE_URL/sales/customers/999/orders" \
  -H "Authorization: Bearer $SALES_TOKEN"
```

### 验收点

- 销售只看到自己客户
- 越权访问返回 403/404 之类受控结果

## G1-G3 开票配置进入系统设置

### 验证步骤

1. 管理员读取 `/admin/settings`
2. 确认开票字段存在
3. 更新开票配置
4. 再次读取确认持久化

### 命令模板

```bash
curl -sS "$BASE_URL/admin/settings" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

curl -sS -X PUT "$BASE_URL/admin/settings" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"invoice_enabled":true,"invoice_baiwang_enabled":true}'
```

### 验收点

- 开票配置通过系统设置接口统一管理
- 不存在单独的散落配置入口

## G4-G7 用户开票申请与后台管理

### 验证步骤

1. 用户创建发票资料
2. 用户对自己的已支付订单申请开票
3. 用户查询自己的开票记录
4. 管理员查询全量开票记录
5. 销售查询自己客户的开票记录

### 验收点

- 用户不能对他人订单申请开票
- 销售不能看非自己客户发票
- 管理员可看全量

## G8-G9 百望云异步开票

### 验证步骤

1. 使用 mock/stub/sandbox 配置百望云
2. 提交开票申请
3. 触发异步任务
4. 模拟成功响应
5. 查询开票状态
6. 模拟失败响应
7. 管理员重试

### 验收点

- 开票状态能从 `requested -> processing -> issued`
- 失败后能进入 `failed`
- 重试后状态可更新
- 支付订单状态不因开票失败而回退

## H1-H5 全链路回归

### 核心回归链

1. 销售生成邀请链接
2. 用户 A 通过该链接注册
3. 用户 A 生成邀请链接
4. 用户 B 通过 A 的链接注册
5. 管理员查看邀请树
6. 用户 B 首充
7. 用户 A 获得首充奖励
8. 用户 B 查询自己的充值记录
9. 用户 B 申请开票
10. 管理员查看并重试开票
11. 销售查看自己名下客户订单与发票

### 最终验收点

- 邀请链正确
- 销售归属正确
- 奖励正确
- 用户充值记录可开票
- 销售数据域无越权
- 开票配置进入系统设置

## 5. 证据输出模板

每个节点完成后，建议至少产出以下证据：

```text
节点编号：
验证时间：
请求接口：
请求角色：
关键响应：
二次回查结果：
结论：PASS / FAIL
```

## 6. 建议下一步

若继续为 agent 准备执行材料，最有价值的是再补一份：

- “测试账号 / 测试数据初始化脚本清单”

这样 agent 可以更稳定地重复跑真实 API 验证。
