# FireMailPlus E2E 问题定位审阅研究文档

生成日期：2026-04-30

对应提交：`f117b861c42db1c107a6e5b82be2fe796ff012ca`

## 1. 范围与结论

本文只做 E2E 发现问题的源码级定位、证据整理和修复建议，不包含代码修复。研究对象来自本次本地生产构建回退测试：

- 后端地址：`http://localhost:8080`
- 前端地址：`http://localhost:3100`
- E2E 产物目录：`/tmp/firemailplus-e2e-artifacts`
- 运行态数据目录：`/tmp/firemailplus-e2e`
- 机器可读后端报告：`/tmp/firemailplus-e2e-artifacts/backend-curl-report.json`
- 前端 HAR：`/tmp/firemailplus-e2e-artifacts/frontend.har`
- 测试数据库：`/tmp/firemailplus-e2e/data/firemail.db`

总体结论：

- 本次 E2E 的前置质量门禁是绿的：`backend go test ./...`、`frontend pnpm type-check`、`make check-api-generated`、后端生产二进制构建、前端生产构建均通过。
- 后端 curl E2E 共 71 项，62 项符合预期，9 项异常；9 项异常集中在 auth refresh 语义、邮箱分组默认语义、同步远端 IMAP 的批量已读、dedup 增强能力降级、dedup schedule 入参校验。
- 前端 jshook/HAR 发现 4 类问题：SSE 心跳/重连噪声、一次邮件已读 500、搜索筛选页无 `account_id` 拉取 folders、搜索结果期望与测试数据不一致。
- Docker 构建失败不是应用回归，而是拉取 `golang:1.24-alpine` 基础镜像时的外部网络/Registry 风险。
- 敏感日志扫描通过：E2E 报告显示 Authorization、Bearer、refresh token、password、JWT-like 泄露计数均为 0。但前端 SSE 客户端当前会把带 token 的 EventSource URL 打到浏览器 console，这不一定进入后端日志，却仍属于前端日志面风险。

## 2. 证据来源

已读取和交叉验证的主要证据：

- `/tmp/firemailplus-e2e-artifacts/E2E_REPORT.md`：E2E 总结、通过项、失败项、运行环境。
- `/tmp/firemailplus-e2e-artifacts/backend-curl-report.json`：71 个 curl 检查的状态、耗时和错误摘要。
- `/tmp/firemailplus-e2e-artifacts/frontend.har`：前端网络请求，包含一次登录 401、一次 `PUT /api/v1/emails/29/read` 500、两次 `GET /api/v1/folders` 400、一次搜索 API 200。
- `/tmp/firemailplus-e2e/data/firemail.db`：运行态 SQLite 数据，核对邮件、文件夹、发送记录和测试前缀是否真实存在。
- `backend/internal/auth/jwt.go`、`backend/internal/auth/service.go`、`backend/internal/handlers/auth.go`：auth refresh 语义。
- `backend/internal/services/email_service.go`、`backend/internal/handlers/email_groups.go`、`backend/internal/services/email_group_test.go`：邮箱分组默认语义、已读状态写回、批量账户已读。
- `backend/internal/handlers/deduplication_handler.go`、`backend/internal/services/deduplication_manager.go`：dedup 统计/报告/schedule。
- `backend/internal/handlers/soft_delete.go`、`openapi/firemail.yaml`、`frontend/src/api/generated/firemail.ts`：soft-delete cleanup 契约。
- `backend/internal/handlers/sse.go`、`backend/internal/sse/*.go`、`frontend/src/lib/sse-client.ts`、`frontend/src/hooks/use-sse.ts`：SSE 连接和心跳。
- `frontend/src/components/mailbox/search-filters.tsx`、`frontend/src/hooks/use-search-emails.ts`、`frontend/src/components/mailbox/search-results-page.tsx`、`frontend/src/lib/api.ts`：搜索页 folders 和搜索执行。
- `Dockerfile`、`frontend/next.config.ts`：本地生产部署和前端 API rewrite。

## 3. 问题总览

| ID | 现象 | 分类 | 严重级别 | 根因置信度 | 建议优先级 |
| --- | --- | --- | --- | --- | --- |
| E2E-01 | `POST /api/v1/auth/refresh` 对新 token 返回 401 | API 语义/契约不一致 | 中 | 高 | P1 |
| E2E-02 | 新建 group 后立即 update 返回“默认分组不可编辑” | 产品语义/API 期望不一致 | 中 | 高 | P1 |
| E2E-03 | `POST /api/v1/accounts/batch/mark-read` 60s 超时 | 真实远端 IMAP 同步阻塞 request path | 高 | 高 | P0 |
| E2E-04 | dedup stats/report 对两个账户返回 500 | 能力降级缺失/未实现路径暴露 | 高 | 高 | P0 |
| E2E-05 | dedup schedule 空 body 返回 500 | 入参校验和状态码错误 | 中 | 高 | P1 |
| E2E-06 | admin soft-delete cleanup 空 body 返回 400 EOF | 契约严格但可用性差 | 低 | 高 | P2 |
| E2E-07 | 前端 repeated SSE heartbeat timeout/EventSource errors | SSE 心跳/前端连接健壮性/日志风险 | 中 | 中 | P1 |
| E2E-08 | HAR 捕获 `PUT /api/v1/emails/29/read` 500 | 邮件已读远端写回失败或代理错误 | 高 | 中 | P0 |
| E2E-09 | 搜索页两次 `GET /api/v1/folders` 无 `account_id` 返回 400 | 前端调用必填参数缺失 | 中 | 高 | P1 |
| E2E-10 | 搜索页无可见前缀搜索结果 | 测试数据/期望偏差，非确认业务缺陷 | 低 | 中 | P2 |
| E2E-11 | Docker build 拉 `golang:1.24-alpine` 失败 | 外部 Registry/网络风险 | 中 | 高 | P2 |
| E2E-12 | 首次 jshook 随机密码 UI 登录失败 | E2E harness/input 可靠性问题 | 低 | 中 | P3 |

## 4. 详细发现

### E2E-01 Auth Refresh 新 Token 返回 401

现象：

- 后端 E2E 记录：`auth refresh`，HTTP 401，耗时 4ms。
- 响应摘要：`Token refresh failed` / `Unauthorized`。
- E2E 调用顺序是刚登录成功后立刻请求 `POST /api/v1/auth/refresh`。

定位：

- `backend/internal/auth/jwt.go` 的 `JWTManager.RefreshToken()` 先校验 token，然后只允许在过期前 30 分钟内刷新：

```go
if time.Until(claims.ExpiresAt.Time) > 30*time.Minute {
    return "", errors.New("token is not eligible for refresh")
}
```

- `backend/internal/auth/service.go` 的 `Service.RefreshToken()` 没有区分“不符合刷新窗口”和“token 无效”，直接把错误返回 handler。
- `backend/internal/handlers/auth.go` 的 `RefreshToken()` 将所有刷新错误映射为 401：

```go
response, err := h.authService.RefreshToken(token)
if err != nil {
    h.respondWithError(c, http.StatusUnauthorized, "Token refresh failed")
    return
}
```

根因判断：

- 这是确认的契约/语义不一致，不是认证基础能力失败。
- 实现是“临近过期才允许 refresh”，而 API 名称、前端/测试直觉和通用 refresh 语义更接近“有效 token 可滚动刷新”。
- 当前 401 会误导调用方认为 token 已失效，实际只是“尚未进入刷新窗口”。

影响：

- 前端如果在启动、路由切换或轮询中主动 refresh 新 token，会被误导为未授权。
- 自动化测试和 API 消费方无法从状态码/错误码判断是否应退出登录、延后刷新或忽略。

建议修复：

- 推荐方案：明确采用滚动刷新语义，允许有效 token 直接 refresh，并配套刷新频率保护。
- 保守方案：保留 30 分钟刷新窗口，但把错误改成稳定业务码和更合适的状态，例如 400/409，消息为 `token_not_eligible_for_refresh`，前端不应登出。
- OpenAPI 需要描述 refresh 策略；若保留窗口，响应 schema 中应有可机器判断的错误码。

验收建议：

- 新增 handler/service 测试：新 token refresh 的目标语义必须固定。
- E2E 中分别覆盖 fresh token、near-expiry token、expired token。

### E2E-02 Group Create 后立即 Update 命中“默认分组不可编辑”

现象：

- 后端 E2E 先 `POST /api/v1/groups` 创建分组，再用返回的 `id` 立即 `PUT /api/v1/groups/{id}`。
- update 返回 HTTP 400，响应摘要：`Failed to update group: 默认分组不可编辑`。

定位：

- `backend/internal/services/email_service.go` 的 `CreateEmailGroup()` 会先创建普通非默认 group。
- 若这是第一个非默认分组，代码立即调用 `SetDefaultEmailGroup()` 把它设为默认，并把 `resultGroup` 改成 default 后的对象再返回：

```go
resultGroup := group
var nonDefaultCount int64
if err := s.db.WithContext(ctx).Model(&models.EmailGroup{}).
    Where("user_id = ? AND is_default = 0", userID).
    Count(&nonDefaultCount).Error; err == nil && nonDefaultCount == 1 {
    defaultGroup, err := s.SetDefaultEmailGroup(ctx, userID, group.ID)
    if err == nil {
        resultGroup = defaultGroup
    }
}
return resultGroup, nil
```

- 同文件 `UpdateEmailGroup()` 明确禁止编辑默认分组：

```go
if group.IsDefault {
    return nil, fmt.Errorf("默认分组不可编辑")
}
```

- `backend/internal/services/email_group_test.go` 中 `TestCreateEmailGroupFirstCustomReturnsFreshDefaultState` 明确断言第一个自定义分组返回 `IsDefault=true`，说明这是现有测试固化过的行为。

根因判断：

- 这是确认的产品/API 期望不一致。
- 后端现行不变量是“第一个自定义分组自动成为默认分组，默认分组不可编辑”。
- E2E 和常规 CRUD 预期是“创建出来的自定义分组可立即编辑”。

影响：

- UI 如果使用标准“创建后重命名/编辑”流程，会在新用户或空分组状态下失败。
- API 消费方无法简单把 create 返回对象当成普通可编辑资源。
- 若默认分组不允许编辑，用户第一次创建的命名可能被永久锁定，产品体验不一致。

建议修复：

- 推荐先做产品决策：默认分组到底是系统占位符，还是用户可编辑的常规分组。
- 若默认分组是系统占位符：创建第一个自定义分组不应自动改成默认；默认应由独立 set-default 操作完成。
- 若默认分组是用户可见常规分组：应允许编辑默认分组的名称，但继续禁止删除或破坏唯一默认不变量。
- 无论选择哪种，都需要同步 OpenAPI 说明、前端 group action 约束和 E2E 断言。

验收建议：

- 新增 create-first-custom 后 update 的 API 测试。
- 新增默认分组 edit/delete/set-default 的权限和不变量测试。

### E2E-03 Batch Mark Accounts Read 60 秒超时

现象：

- 后端 E2E 记录：`batch mark accounts read`，状态 0，耗时 60003ms。
- 错误摘要：`The operation was aborted due to timeout`。
- 运行环境是真实 Outlook 账户，`ENABLE_REAL_EMAIL_SYNC=true`、`MOCK_EMAIL_PROVIDERS=false`。

定位：

- `backend/internal/handlers/email_accounts.go` 的 `BatchMarkAccountsAsRead()` 在 HTTP request path 中同步调用：

```go
if err := h.emailService.MarkAccountsAsRead(c.Request.Context(), userID, req.AccountIDs); err != nil {
    h.respondWithError(c, http.StatusBadRequest, "Failed to mark accounts as read: "+err.Error())
    return
}
```

- `backend/internal/services/email_service.go` 的 `MarkAccountsAsRead()` 串行处理每个账户：

```go
for _, accountID := range accountIDs {
    if err := s.MarkAccountAsRead(ctx, userID, accountID); err != nil {
        return err
    }
}
```

- `MarkAccountAsRead()` 会查询账户所有未读邮件，调用 `markEmailsAsReadOnServer()` 后才批量更新本地 DB。
- `markEmailsAsReadOnServer()` 会按文件夹聚合 UID，创建 provider、连接 IMAP、逐文件夹 `SelectFolder()`，再 `MarkAsRead()`：

```go
for folderPath, uids := range folderUIDs {
    if _, err := imapClient.SelectFolder(ctx, folderPath); err != nil {
        return fmt.Errorf("failed to select folder %s: %w", folderPath, err)
    }
    if err := imapClient.MarkAsRead(ctx, uids); err != nil {
        return fmt.Errorf("failed to mark folder %s emails as read on server: %w", folderPath, err)
    }
}
```

根因判断：

- 这是确认的 request-path 设计问题。
- 该接口表面是批量快捷操作，实际在 HTTP 请求生命周期内执行真实远端 IMAP 写操作，而且对账户串行、对文件夹串行。
- 使用 `c.Request.Context()` 会把浏览器或测试 harness 的超时直接传递到 provider 层；60s 超时后既可能远端已部分成功，也可能本地尚未更新，形成不确定状态。

影响：

- 真实邮箱未读量较大、网络慢、Outlook IMAP 响应慢时稳定超时。
- 可能出现部分账户/部分文件夹远端已读、本地未读，或本地未更新但用户以为操作失败。
- SSE 事件只在本地批量更新后发布，超时时前端无法得到明确进度。

建议修复：

- 推荐把批量账户已读改为异步 job：立即返回 202 和 job id，后台按账户/文件夹执行，前端通过 SSE 或轮询展示进度。
- 对单账户 `MarkAccountAsRead()` 也应支持 bounded chunk、超时、可恢复状态和部分失败报告。
- 若必须同步，至少增加服务器端固定超时、最大 UID 批次、每账户并发限制、部分成功响应结构。
- 需要明确强一致策略：远端写成功后本地更新；远端失败时不伪造本地已读，或者返回 pending 状态。

验收建议：

- 构造 fake provider，模拟多文件夹慢 IMAP，验证接口不阻塞超过约定时间。
- 增加真实 E2E：批量已读返回 job id；job 完成后本地 unread、folder unread、SSE payload、远端状态一致。

### E2E-04 Dedup Stats/Report 返回 500

现象：

- 两个 Outlook 账户的 `GET /api/v1/deduplication/accounts/{id}/stats` 均返回 500。
- 两个 Outlook 账户的 `GET /api/v1/deduplication/accounts/{id}/report` 均返回 500。
- 错误摘要均包含：`enhanced deduplicator not available`。

定位：

- `backend/internal/handlers/deduplication_handler.go` 的 `GetDeduplicationStats()` 实际调用 `GetDeduplicationReport()`，再返回 `report.Stats`：

```go
report, err := h.deduplicationManager.GetDeduplicationReport(c.Request.Context(), uint(accountID))
if err != nil {
    c.JSON(http.StatusInternalServerError, ErrorResponse{
        Error:   "Failed to get deduplication stats",
        Message: err.Error(),
    })
    return
}
```

- `backend/internal/services/deduplication_manager.go` 的 `GetDeduplicationReport()` 只接受 `CreateDeduplicator("standard")` 返回值实现 `EnhancedDeduplicator`：

```go
deduplicator := m.deduplicatorFactory.CreateDeduplicator("standard")
if enhanced, ok := deduplicator.(EnhancedDeduplicator); ok {
    stats, err := enhanced.GetDeduplicationStats(ctx, accountID)
    ...
    return &DeduplicationReport{...}, nil
}

return nil, fmt.Errorf("enhanced deduplicator not available")
```

- 同文件另有 `tryCreateEnhancedStandardDeduplicator()` 和 `tryCreateEnhancedGmailDeduplicator()`，但 `GetDeduplicationReport()` 没有走这套增强创建路径，也没有标准 deduplicator fallback。
- `DeduplicateAccount()` 有标准降级路径 `handleStandardDeduplication()`，但 stats/report 没有对应降级。

根因判断：

- 这是确认的能力降级缺失。
- stats/report 是公开路由，但实现等价于“只有 enhanced deduplicator 可用时才可用”；enhanced 不可用时直接 500。
- 500 表示服务端异常，但这里更像 feature unavailable、not implemented 或 fallback 未接通。

影响：

- 前端或 API 用户无法展示 dedup 基础统计，即使基础去重器存在。
- E2E 中两个真实 Outlook 账户稳定触发 500，说明当前默认配置下该公开 API 不可用。
- OpenAPI 已公开 dedup stats/report，实际能力不满足契约。

建议修复：

- 推荐让 stats/report 支持标准 fallback：至少返回 account 邮件总数、可计算重复数为 unknown 或 0、last_updated、能力标记。
- 如果 enhanced 必须启用，则返回 501/503 和稳定错误码，例如 `dedup_enhanced_unavailable`，不要返回 500。
- 统一 `DeduplicateAccount()`、`GetDeduplicationReport()`、`GetDeduplicationStats()` 的 deduplicator 创建策略，避免同一模块一处可降级、一处不可降级。

验收建议：

- 默认配置下 stats/report 不返回 500。
- enhanced 关闭时返回可解释 fallback 或 503 typed error。
- enhanced 开启且 DB 可用时返回真实 stats/report。

### E2E-05 Dedup Schedule 空 Body 返回 500

现象：

- 两个 Outlook 账户的 `POST /api/v1/deduplication/accounts/{id}/schedule` 均返回 500。
- E2E 请求 body 是 `{}`。
- 响应摘要：`invalid schedule time format`。
- cancel schedule 返回 200。

定位：

- `backend/internal/handlers/deduplication_handler.go` 的 `ScheduleDeduplicationRequest` 没有 binding 必填校验：

```go
type ScheduleDeduplicationRequest struct {
    Enabled   bool                           `json:"enabled"`
    Frequency string                         `json:"frequency"`
    Time      string                         `json:"time"`
    Options   *services.DeduplicationOptions `json:"options"`
}
```

- handler 将零值 `Frequency=""`、`Time=""` 直接传给 service：

```go
schedule := &services.DeduplicationSchedule{
    Enabled:   req.Enabled,
    Frequency: req.Frequency,
    Time:      req.Time,
    Options:   req.Options,
}
err = h.deduplicationManager.ScheduleDeduplication(...)
```

- service 端 `calculateNextDeduplicationRun()` 用 `fmt.Sscanf(hhmm, "%02d:%02d", &hour, &minute)` 解析空字符串，返回 `invalid schedule time format`。
- handler 将该错误统一映射为 500：

```go
c.JSON(http.StatusInternalServerError, ErrorResponse{
    Error:   "Failed to schedule deduplication",
    Message: err.Error(),
})
```

根因判断：

- 这是确认的输入校验和错误映射问题。
- 空 body 是客户端错误，应返回 400 和字段级错误，而不是 500。
- 若产品希望空 body 表示使用默认 schedule，则当前实现缺少默认值填充。

影响：

- UI 难以根据错误引导用户填写 frequency/time。
- 自动化测试中只要发送 `{}` 就被算作服务端异常。

建议修复：

- 明确 schedule API 语义：
  - 若要求显式配置：`frequency`、`time` 必填，`frequency` 限定 daily/weekly/monthly，`time` 限定 HH:MM，校验失败返回 400。
  - 若允许默认：空 body 填充默认值，例如 daily 02:00，返回 200/201。
- service 错误应分类，handler 不应把 validation error 映射为 500。
- OpenAPI 当前 `ScheduleDeduplicationJSONBody` 还是 `map[string]interface{}`，应补全 request schema。

验收建议：

- `{}` 返回稳定 400 或默认成功，取决于产品决策。
- `{"frequency":"daily","time":"25:99"}` 返回 400。
- `{"enabled":true,"frequency":"daily","time":"02:00"}` 返回 200，且 `next_run` 合法。

### E2E-06 Admin Soft-Delete Cleanup 空 Body 返回 400 EOF

现象：

- E2E 对 `POST /api/v1/admin/soft-deletes/cleanup` 发送空 body。
- 该项在 E2E 中按“预期 4xx”处理，但实际响应是 400 `Invalid request body: EOF`。

定位：

- `backend/internal/handlers/soft_delete.go` 要求 JSON body 中有 `retention_days`：

```go
var req struct {
    RetentionDays int `json:"retention_days" binding:"required,min=1"`
}

if !h.bindJSON(c, &req) {
    return
}
```

- `openapi/firemail.yaml` 中 `SoftDeleteCleanupRequest` 也要求 `retention_days`：

```yaml
SoftDeleteCleanupRequest:
  type: object
  additionalProperties: false
  required: [retention_days]
  properties:
    retention_days:
      type: integer
      minimum: 1
```

根因判断：

- 严格来说这不是后端 bug，因为 OpenAPI 和 handler 均要求 body。
- 但作为 admin cleanup 操作，空 body 返回 EOF 可用性较差，也不利于 CLI/curl 使用。

影响：

- 管理员从 curl 调用时容易踩坑。
- 前端生成 SDK `cleanupSoftDeletes(softDeleteCleanupRequest)` 要求入参，UI 侧没问题；人工 API 调用体验较差。

建议修复：

- 保守方案：保留必填 body，但错误消息改成字段级错误，例如 `retention_days is required`。
- 可用性方案：允许空 body，使用配置默认保留天数，并在响应中返回实际使用的 retention days。
- OpenAPI 必须同步；若允许默认，`requestBody.required` 应改为 false 或 schema 增加默认值说明。

验收建议：

- 空 body 行为固定为“明确 400”或“默认成功”，不要再返回 EOF。
- 有效 body `{ "retention_days": 30 }` 成功。

### E2E-07 SSE Heartbeat Timeout/EventSource 错误

现象：

- 前端 jshook console 出现 repeated SSE heartbeat timeout/EventSource errors。
- HAR 无法完整捕获 EventSource 流，只能从前端 console 行为和源码判断。
- 后端 E2E 的 `sse stats` 和 `sse test event` 均通过，说明 SSE 管理接口可用。

定位：

- `frontend/src/lib/sse-client.ts` 在连接时构造 URL：

```ts
const url = `${this.config.baseUrl}/sse/events?client_id=${this.config.clientId}&token=${encodeURIComponent(this.config.token)}`;
this.eventSource = new EventSource(url);
console.log('🔗 [SSEClient] 正在连接到:', url);
```

- 该 console 会输出完整 token query，属于前端日志面敏感信息风险。
- 同文件 heartbeat timeout 为 60s；只要 60s 内未处理到 `heartbeat` 类型事件就触发错误并 schedule reconnect：

```ts
this.heartbeatTimer = setTimeout(() => {
    console.warn('⚠️ [SSEClient] 心跳超时，可能连接已断开');
    this.handleError(new Error('心跳超时'));
    this.scheduleReconnect();
}, this.config.heartbeatTimeout);
```

- 后端默认 heartbeat interval 为 30s，`backend/internal/sse/service.go` 会周期性向 user connections 发布 `heartbeat`：

```go
HeartbeatInterval: 30 * time.Second
...
if s.config.EnableHeartbeat {
    s.startHeartbeatRoutine()
}
```

- `backend/internal/sse/events.go` 输出格式包含 `event: heartbeat` 和 JSON `data`，前端也注册了 typed event listener，理论上可处理。
- `frontend/src/hooks/use-sse.ts` 的 `useSSE()` 依赖回调较多，`MailboxSSEBridge` 在 mailbox/search/mobile 组件中都可能挂载；若页面切换时多个 bridge 重复创建/销毁，可能导致连接抖动、超过 max connection、旧连接被关闭后前端仍报错。

根因判断：

- 当前只能定为中等置信度问题，因 E2E 没有保存完整 console 文本和 SSE 流响应。
- 已确认风险点有两个：
  - 前端 console 输出含 token 的 EventSource URL。
  - SSE 客户端遇到 heartbeat timeout 后只 schedule reconnect，但没有显式关闭旧 EventSource；重复页面 bridge 可能放大噪声。
- 还需要用专门 SSE smoke 捕获 2 分钟流内容，验证后端 heartbeat 是否真实到达浏览器。

影响：

- 用户在 mailbox/search/mobile 导航中看到实时状态不稳定。
- heartbeat timeout 会触发重复错误日志，影响调试和监控。
- console 中 token query 会增加浏览器日志、远程调试、用户截图泄露风险。

建议修复：

- 立即移除或脱敏 `console.log` 中的 SSE URL token，只输出 path、client_id 和 token presence。
- heartbeat timeout 触发重连前先 `eventSource.close()` 并清空引用，避免旧连接残留。
- 确认全局只挂一个 `MailboxSSEBridge`，不要在 layout、search page、mobile layout 中重复挂载。
- 增加浏览器级 SSE smoke：连接 120s，断言至少收到 2 个 heartbeat，且 console 不包含 token。
- 如果生产经 Next rewrite 或代理转发 SSE，需要验证 chunk flushing、缓存头和连接保持。

验收建议：

- jshook/Playwright console 采集 2 分钟内无 repeated heartbeat timeout。
- SSE HAR 或自定义 EventSource probe 收到 heartbeat。
- console/log 扫描不出现 JWT/token query。

### E2E-08 `PUT /api/v1/emails/29/read` 500

现象：

- 前端 HAR 捕获一次：`PUT http://localhost:3100/api/v1/emails/29/read`，响应 500。
- 该请求走前端 `localhost:3100`，再由 `frontend/next.config.ts` rewrite 到 `http://localhost:8080/api/:path*`。
- `/tmp/firemailplus-e2e/logs/backend.log` 只有启动日志，没有对应请求错误日志；因此无法从后端 runtime log 得到直接错误文本。

定位：

- 后端 handler `backend/internal/handlers/emails.go` 对该接口调用 `emailService.MarkEmailAsRead()`；service 错误按 handler 写法应返回 400：

```go
err := h.emailService.MarkEmailAsRead(c.Request.Context(), userID, emailID)
if err != nil {
    h.respondWithError(c, http.StatusBadRequest, "Failed to mark email as read: "+err.Error())
    return
}
```

- `backend/internal/services/email_service.go` 的 `setEmailReadState()` 是强一致远端写回：先校验 UID 和 folder path，再连接 provider、选择 IMAP 文件夹、执行 `MarkAsRead()`，之后才保存本地 DB。
- 测试 DB 中邮件 29 的状态：

```text
id=29, account_id=2, folder_id=19, uid=151, is_read=0, folder.name=Archive, folder.path=Archive
```

- 因此它不属于 missing UID 或 missing folder path；更可能是 provider 连接、Outlook Archive 文件夹选择、IMAP 标记已读或 Next rewrite 代理层错误。
- OpenAPI 对 `/api/v1/emails/{id}/read` 只列出 200/401/404，未列出远端写回失败时可能出现的 400/409/502/503。

根因判断：

- 这是高影响、中等置信度问题。
- 已确认该接口会在用户点击邮件时触发远端 IMAP 写回；远端失败会阻断本地已读更新。
- 未确认 500 是后端返回、Next proxy 返回，还是连接中断造成的前端代理错误。因为后端 handler 对 service 错误写的是 400，而 HAR 看到的是前端同源 URL 500。

影响：

- 用户点击邮件后可能无法标记已读，unread badge 和远端状态都不变。
- 若 500 来自代理层，前端可能无法展示真实错误。
- 若远端 Archive 文件夹路径或 Outlook IMAP folder mapping 有问题，移动/归档后的邮件读写都会受影响。

建议修复：

- 增加后端请求日志或错误日志，至少记录 route、status、typed error，不记录 token。
- 为 `MarkEmailAsRead()` 增加 provider fake 测试：folder path、Archive、select folder 失败、mark read 失败分别返回稳定错误码。
- 对前端 rewrite 环境复现同一请求，比较直连 8080 和经 3100 rewrite 的状态码与响应体。
- 考虑把远端写回失败映射为 502/503 typed error，而不是 400 或代理 500；前端可提示“远端同步失败，本地未更改”。

验收建议：

- 直连后端和前端 rewrite 调用同一邮件 read，状态码一致。
- 远端失败时返回可机器判断错误码，不是 500。
- 本地 DB 状态和 unread counters 在失败时不被伪更新，成功时一致更新并发 SSE。

### E2E-09 搜索页 `GET /api/v1/folders` 缺少 `account_id`

现象：

- 前端 HAR 捕获两次：`GET http://localhost:3100/api/v1/folders`，均返回 400。
- 后端响应原因：folders 列表接口要求 `account_id`。

定位：

- `backend/internal/handlers/folders.go` 的 `GetFolders()` 明确要求 query `account_id`：

```go
accountID := h.parseUintQuery(c, "account_id", 0)
if accountID == 0 {
    h.respondWithError(c, http.StatusBadRequest, "account_id parameter is required")
    return
}
```

- `frontend/src/lib/api.ts` 的 `getFolders(accountId?: number)` 允许不传 accountId，并会生成无 query 的 `/api/v1/folders`：

```ts
async getFolders(accountId?: number): Promise<ApiResponse<Folder[]>> {
    return this.request(this.generatedEndpoint(getListFoldersUrl({ account_id: accountId })));
}
```

- `frontend/src/components/mailbox/search-filters.tsx` 在 folders 为空时先尝试无 account id 拉取：

```ts
const response = await apiClient.getFolders();
if (response.success && response.data) {
    setFolders(response.data);
    return;
}
```

- 失败后才基于 accounts 逐账户拉取 folders；如果 accounts 尚未加载，则直接 return，导致搜索筛选项初始化不完整。

根因判断：

- 这是确认的前端调用契约错误。
- OpenAPI/generated client 里 `account_id` 仍是 optional，但后端实际强制必填，契约也有漂移风险。

影响：

- 搜索页首次加载稳定产生 400 噪声。
- folders 筛选项可能空白，尤其是 accounts 还在加载时。
- HAR 中 400 会污染 E2E 健康判断。

建议修复：

- 删除 `search-filters.tsx` 中无 account id 的 `apiClient.getFolders()` 尝试。
- 先确保 accounts 加载完成，再 `Promise.all(accounts.map(account => apiClient.getFolders(account.id)))`。
- 将 `apiClient.getFolders(accountId?: number)` 改成必填，或者新增显式 `getAllFoldersForAccounts(accounts)` facade，避免其他页面继续误用。
- OpenAPI 中 `ListFoldersParams.account_id` 应标为 required，重新生成 SDK。

验收建议：

- Search page HAR 中不再出现 `/api/v1/folders` 无 query。
- accounts 未加载时不会发错误请求；加载完成后按账户拉取并合并 folders。
- 前端 type-check 覆盖所有 `getFolders()` 调用，禁止无参调用。

### E2E-10 搜索页无可见前缀搜索结果

现象：

- jshook 报告：搜索页加载并接受输入，但搜索执行没有产生预期可见结果列表。
- HAR 中实际存在搜索请求：`GET /api/v1/emails/search?page=1&page_size=20&q=FMP-E2E-20260430101455`，状态 200。
- 测试 DB 中 `emails` 表没有任何 subject/from/to/text/html 包含 `FMP-E2E-20260430101455` 的记录，计数为 0。
- `sent_emails` 表中存在一条 subject 为 `FMP-E2E-20260430101455 Bulk Send` 的记录，但搜索接口查询的是 `emails` 表，不查 `sent_emails`。

定位：

- `backend/internal/handlers/emails.go` 的 `SearchEmails()` 查询当前邮件库 `emails`，不是发送历史。
- `backend/internal/services/email_service.go` 的搜索条件对 `emails.subject`、`from_address`、`to_addresses`、`text_body`、`html_body` 做 LIKE。
- E2E backend 脚本先发送了 prefix 邮件，并把 bulk send 记录写入 `sent_emails`；但真实收件账号同步是否把该邮件拉回 `emails` 表没有保证。
- HAR 证明搜索请求已发出并返回 200，所以“没有执行搜索”不成立；更准确是“没有可展示匹配数据”。

根因判断：

- 这是测试数据/期望偏差，不是确认业务缺陷。
- 如果产品要求搜索也覆盖发送历史或 send_queue，则当前搜索范围不足；但从现有 handler 命名看它是邮箱邮件搜索，不是发送记录搜索。
- 前端仍有一个弱点：`SearchResultsContent` 的空态依赖 URL `q` 的 `currentQuery`，而 `SearchBar` 在搜索页传入 `onSearch` 后只调用 search，不更新 URL；如果搜索无结果，可能显示“开始搜索”而不是“未找到匹配的邮件”。

影响：

- E2E 对“发送后立即搜索前缀可见”的断言不稳定，受 SMTP 投递、Outlook 入站同步、搜索范围影响。
- 用户在搜索页手动输入但 URL 未更新时，空态文案可能误导。

建议修复：

- E2E 应先在 `emails` 表或真实收件箱中构造可搜索邮件，再断言搜索结果。
- 若要验证发送历史搜索，应新增独立 API 或扩展搜索范围，并明确 UI 入口。
- 前端搜索页应把当前 query 状态作为空态依据，而不是只依赖 URL `q`。
- `SearchBar` 在搜索页执行 onSearch 时可以同步 `router.replace('/mailbox/search?q=...')`，保持 URL、状态和空态一致。

验收建议：

- 测试数据库中存在 prefix 邮件时，搜索结果列表显示该邮件。
- prefix 不存在时，页面显示“未找到匹配的邮件”，而不是“开始搜索”。
- HAR 中搜索请求状态 200，响应 total 与 DB 查询一致。

### E2E-11 Docker Build 基础镜像拉取失败

现象：

- Docker build 尝试两次失败。
- E2E 报告摘要：解析 `golang:1.24-alpine` from Docker Hub/Cloudflare 时 connection reset。
- 本地生产构建 fallback 成功，后端二进制和前端 build 都通过。

定位：

- `Dockerfile` 第一阶段使用：

```dockerfile
FROM golang:1.24-alpine AS backend-builder
```

- 第二、三阶段使用 `node:20-alpine`。
- 失败发生在基础镜像 metadata/pull 阶段，不是应用编译阶段。

根因判断：

- 这是外部 Registry/网络可靠性风险，不是应用代码回归。
- 但它会影响 CI/CD、首次部署和灾备环境构建。

影响：

- 无缓存环境无法稳定构建镜像。
- 如果 CI 依赖 Docker Hub 公网拉取，构建结果受 Cloudflare/Docker Hub 网络波动影响。

建议修复：

- 配置 registry mirror 或内部缓存。
- CI 中预拉取并缓存 `golang:1.24-alpine`、`node:20-alpine`。
- 考虑 pin digest，减少 tag metadata 解析不确定性。
- 保留本地生产 build fallback 作为临时验证路径，但部署验收仍需真实 Docker build 绿。

验收建议：

- 在无本地缓存环境执行 `docker build` 成功。
- CI 记录基础镜像 digest。

### E2E-12 首次 jshook 随机密码 UI 登录失败

现象：

- 前端 jshook 首次随机密码登录失败。
- 直接后端 login 成功。
- 随后在隔离实例内 rotate admin password 后，真实 UI 登录成功。
- HAR 中存在一次 `POST /api/v1/auth/login` 401。

定位：

- 当前证据不能证明 UI 登录组件有确定缺陷。
- E2E 报告也把该问题描述为“direct first random-password form attempt failed, then admin password was rotated and login succeeded through the real UI”。

根因判断：

- 中等置信度归类为 E2E harness/input 可靠性问题。
- 可能原因包括：随机密码注入不一致、表单填入时机、旧密码/新密码混淆、浏览器状态残留。

影响：

- 会污染端到端报告，但不应作为产品缺陷优先修复。

建议修复：

- jshook 登录步骤记录实际使用的是哪个测试密码标签，而不是原文密码。
- 每次登录前清理 localStorage/sessionStorage/cookies。
- 只在同一密码直连后端成功、UI 同密码失败时再升级为前端 bug。

验收建议：

- UI 登录前后保留脱敏 evidence：输入字段非空、提交按钮状态、返回状态码、错误 toast。

## 5. 横向问题

### 5.1 真实远端操作直接阻塞 HTTP 请求

`batch mark accounts read` 和单封邮件 read 都体现了同一类风险：用户操作表面是 UI 状态切换，底层却需要真实 Outlook IMAP 写回。当前实现为了强一致，先远端写成功再本地保存，这是正确方向，但同步放在 request path 中会导致超时、代理 500、前端体验差。

建议把耗时、多账户、多文件夹操作改为 job；单封邮件操作保留同步但增加超时、错误分类和代理一致性测试。

### 5.2 OpenAPI/Generated SDK 与实际后端约束仍有漂移

典型例子：

- `GET /api/v1/folders` 后端要求 `account_id`，生成 SDK 和 facade 允许无参。
- dedup schedule 生成 Go 类型是 `map[string]interface{}`，而 handler/service 实际需要 `enabled/frequency/time/options`。
- `/api/v1/emails/{id}/read` OpenAPI 未描述远端同步失败。

建议下一轮修复从 OpenAPI 契约先行，补 required、typed error、request schema，再生成后端/前端代码。

### 5.3 能力未完全接通但已公开路由

dedup stats/report 是最明显案例：路由公开、权限校验存在，但默认配置下返回 `enhanced deduplicator not available` 500。公开路由至少需要：

- 可用实现。
- 明确 501/503 能力不可用。
- 前端可隐藏入口或展示 unsupported。

### 5.4 E2E 测试数据需要区分“发送历史”和“收件箱邮件”

本次搜索期望使用同一个 prefix，但 prefix 最终出现在 `sent_emails`，没有出现在 `emails`。如果目标是测邮箱搜索，应造 `emails` 可查数据或等待真实收件箱同步；如果目标是测发送历史，应有独立接口和 UI。

## 6. 建议修复顺序

P0：

- 修复 `PUT /api/v1/emails/{id}/read` 的 500/代理不一致，至少保证错误码和响应体可诊断。
- 将 `POST /api/v1/accounts/batch/mark-read` 改为异步或 bounded 同步，避免 60s request timeout。
- 修复 dedup stats/report 默认 500，提供 fallback 或 typed unavailable。

P1：

- 明确 auth refresh 语义并修正状态码/错误码。
- 明确 group 默认语义，保证 create/update 用户路径符合产品预期。
- 修复 dedup schedule 入参校验和 OpenAPI schema。
- 修复搜索页无 `account_id` folders 请求。
- 修复 SSE token console 泄露风险，并增加 heartbeat smoke。

P2：

- 改善 admin soft-delete cleanup 空 body 错误。
- 调整搜索 E2E 数据构造和空态展示。
- 增加 Docker registry 缓存或 mirror。

P3：

- 强化 jshook 登录 harness 证据和状态清理。

## 7. 下一轮修复验收清单

后端：

- `cd backend && go test ./...`
- 对 auth refresh 增加 fresh/near-expiry/expired token 测试。
- 对 dedup stats/report/schedule 增加默认配置和 enhanced 配置测试。
- 对 mark email/account read 增加 fake provider 成功、SelectFolder 失败、MarkAsRead 失败、timeout 测试。
- 对 folders list 增加 OpenAPI required drift 测试。

前端：

- `cd frontend && pnpm type-check`
- 搜索页 HAR 不再出现 `/api/v1/folders` 无 `account_id`。
- 搜索页 prefix 存在时显示结果，不存在时显示“未找到匹配的邮件”。
- SSE smoke 运行 120s，收到 heartbeat，console 不包含 token/JWT。

契约与生成物：

- OpenAPI lint 通过。
- 后端 generated 同步。
- 前端 SDK generated 同步。
- `git diff --exit-code` 检查生成物无漂移。

E2E：

- 后端 curl 71 项不再出现非预期 5xx/timeout。
- HAR 无非预期 400/500。
- Docker build 在配置 mirror/cache 后通过，或 CI 明确使用可复现基础镜像缓存。

## 8. 未决策问题

- Auth refresh 是否采用任意有效 token 滚动刷新，还是保留 30 分钟刷新窗口。
- 默认邮箱分组是否允许重命名；第一个自定义分组是否仍应自动成为默认。
- 批量已读是否必须强一致同步完成后返回，还是允许异步 job + 进度。
- dedup enhanced 不可用时是返回 fallback stats，还是返回 typed 503。
- 搜索是否应覆盖 `sent_emails` 和 `send_queue`，还是只搜索 `emails`。
- soft-delete cleanup 空 body 是否采用默认 retention days。

