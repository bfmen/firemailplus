# FireMailPlus 后端问题与 TODO 深度审阅

审阅日期：2026-04-29

审阅范围：`backend/cmd`、`backend/internal`、`backend/database/migrations`、后端 Go module 与当前注册路由。本文只做问题挖掘和整理，不实施修复。

验证基线：在 `backend/` 下执行 `go test ./...`，当前全部通过。结论是“现有测试覆盖的路径可编译并通过”，不代表下面列出的未实现、未注册、运行时和安全风险不存在。

## 总览

当前后端不是“完全没写完”，主链路包括登录、账户、基础邮件列表/详情/发送、读未读、移动、文件夹、分组、附件、SSE 都有实现并能通过测试。但源码里同时存在三类需要优先处理的问题：

1. 已写服务或 handler 未接入真实路由，形成“代码存在但产品不可用”的死链路。
2. 运行时路径里存在明确的 `not implemented`、占位返回、伪成功或只写日志的 TODO。
3. 部分安全、迁移和一致性边界会在生产数据、多人账户或长时间运行后暴露，而现有测试没有覆盖。

## P0/P1 高优先级问题

### 1. 扩展发送、草稿、模板、重发状态接口写了但没有注册到主路由

证据：

- `EmailSendHandler.RegisterRoutes` 定义了 `/emails/send/bulk`、`/emails/send/:send_id/status`、`/emails/send/:send_id/resend`、`/emails/draft`、`/emails/templates` 等接口，见 `backend/internal/handlers/email_send_handler.go:40`。
- `Handler.New` 创建了 `emailComposer`、`emailSender`、`scheduledEmailService`，但没有把 `EmailSendHandler` 实例保存到 `Handler`，见 `backend/internal/handlers/handler.go:95`。
- `setupRoutes` 当前只注册基础 `emails.POST("/send", h.SendEmail)`，没有调用 `NewEmailSendHandler(...).RegisterRoutes(api)`，见 `backend/cmd/firemail/main.go:179`。

影响：

- 批量发送、发送状态查询、重发、草稿和模板 REST API 对外不可达。
- 前端或调用方如果按这些 handler 设计调用，会得到 404，而不是业务错误。
- 这会掩盖 `StandardEmailSender` 内部的未实现问题，因为测试和主路由都没有真正触达这些扩展接口。

建议：

- 明确产品取舍：如果扩展发送/草稿/模板是目标功能，应在 `Handler` 中正式注入并注册；如果不是目标功能，应删除或隔离死代码，避免误导。
- 注册前必须先修下面的发送状态、重发表、迁移 schema 问题，否则注册后会把隐藏缺陷暴露给用户。

### 2. `StandardEmailSender` 状态持久化和重发路径未实现

证据：

- `GetSendStatus` 只先查内存 map，miss 后调用 `loadSendStatusFromDB`，见 `backend/internal/services/email_sender.go:174`。
- `loadSendStatusFromDB` 固定返回 `send status not found`，见 `backend/internal/services/email_sender.go:406`。
- `ResendEmail` 依赖 `loadSentEmailWithAccountFromDB`，见 `backend/internal/services/email_sender.go:187`。
- `loadSentEmailWithAccountFromDB` 固定返回 `load sent email not implemented`，见 `backend/internal/services/email_sender.go:413`。

影响：

- 服务重启后所有发送状态丢失。
- 重发接口即使注册，也必然失败。
- 异步发送如果后台失败，调用方拿到的初始 `pending` 结果不能可靠追踪最终状态。

建议：

- 先设计发送状态的唯一真相源。可以将 `send_queue` / `sent_emails` 作为持久化状态，或删除未承诺的状态查询/重发 API。
- 异步发送需要持久化 `pending -> sending -> sent/failed` 状态，而不是只在内存 map 中更新。

### 3. `sent_emails`、`email_drafts`、`email_quotas` 模型与迁移不一致

证据：

- `SentEmail.TableName()` 返回 `sent_emails`，见 `backend/internal/models/sent_email.go:34`，但 migrations 里没有 `CREATE TABLE sent_emails`。
- `EmailDraft.TableName()` 返回 `email_drafts`，见 `backend/internal/models/sent_email.go:185`，但迁移创建的是 `drafts`，见 `backend/database/migrations/000009_create_drafts_table.up.sql:2`。
- `EmailQuota.TableName()` 返回 `email_quotas`，见 `backend/internal/models/sent_email.go:251`，但迁移中没有创建 `email_quotas`。
- `email_templates` 迁移只有 `name/subject/body/category/tags/is_shared/is_built_in` 等字段，见 `backend/database/migrations/000008_create_email_templates_table.up.sql:2`；模型却使用 `description/text_body/html_body/variables/is_active/is_default/usage_count/last_used_at` 等字段，见 `backend/internal/models/sent_email.go:39`。

影响：

- 一旦扩展发送 handler 或 template service 被注册，模板创建/读取可能因缺列失败。
- `saveSentEmail` 创建 `models.SentEmail` 时会写不存在的 `sent_emails` 表，见 `backend/internal/services/email_sender.go:420`。
- 当前 `go test ./...` 通过，主要是因为这些路径没有被端到端触发或测试使用 `AutoMigrate` 局部建表，不能代表真实迁移后的数据库可用。

建议：

- 先做 schema inventory：以 migrations 为线上真相源，逐表比对所有 GORM model。
- 不建议直接 `AutoMigrate` 掩盖差异；应补版本化迁移并写迁移测试。

### 4. 认证中间件会记录 Authorization header 和 token 前缀

证据：

- `AuthRequiredWithService` 记录 `Authorization header` 前 50 字符，见 `backend/internal/middleware/auth.go:49`。
- 同函数记录提取后的 token 前 50 字符，见 `backend/internal/middleware/auth.go:62`。
- token 校验失败时继续记录 token 前缀，见 `backend/internal/middleware/auth.go:77`。

影响：

- 日志中会出现 Bearer token 的可识别前缀；在 JWT 长度较短或日志汇聚系统权限较宽时，这是凭证泄露风险。
- 该日志在每个受保护请求都会触发，生产环境噪音也很大。

建议：

- 不记录任何 token 或 Authorization 内容。最多记录是否存在、长度、请求路径、用户 ID 或 trace ID。
- 对认证失败日志做限流或降级为 debug。

### 5. 删除邮件远端失败后仍本地软删除，形成本地/服务器状态分叉

证据：

- `DeleteEmail` 远端 IMAP 删除失败只 `log.Printf("Warning: failed ...")`，不中断流程，见 `backend/internal/services/email_service.go:1662`。
- 即使远端失败，后续仍将 `email.IsDeleted = true` 并保存数据库，见 `backend/internal/services/email_service.go:1694`。

影响：

- 本地 UI 显示已删除，但服务器上邮件仍存在。
- 下一次同步可能重新拉回、产生重复或造成用户误判。
- 该语义与读未读路径不同；读未读已改成“先远端成功，再本地保存”的强一致模式。

建议：

- 明确删除语义：如果要求强一致，应远端失败即返回错误；如果接受本地删除，应标记为 pending remote delete 并有重试/冲突恢复机制。

### 6. 去重管理路由未注册，且权限校验为空实现

证据：

- `DeduplicationHandler.RegisterRoutes` 定义了 `/deduplication/accounts/:id/...` 等接口，见 `backend/internal/handlers/deduplication_handler.go:25`。
- 当前主路由没有创建或注册 `DeduplicationHandler`，见 `backend/cmd/firemail/main.go:117` 起的 `setupRoutes`。
- 其 `validateAccountAccess` 固定 `return nil`，见 `backend/internal/handlers/deduplication_handler.go:364`。

影响：

- 现在接口不可达，所以权限漏洞未暴露。
- 如果未来只是简单注册 handler，会立即引入跨账户访问风险：任意登录用户可对任意 account ID 触发去重、获取报告或配置任务。

建议：

- 注册前必须实现 `account_id + user_id` 权限校验。
- 去重报告、计划任务也应按用户隔离。

### 7. 去重计划任务是伪成功

证据：

- `ScheduleDeduplication` 注释写“应该集成到任务调度系统中”，实际只写日志并返回 nil，见 `backend/internal/services/deduplication_manager.go:388`。
- `CancelScheduledDeduplication` 也只写日志，见 `backend/internal/services/deduplication_manager.go:396`。

影响：

- 一旦路由注册，用户会收到“scheduled successfully”，但没有任何后续任务会执行。
- 这是典型的产品伪成功，比直接返回 501 更难排查。

建议：

- 未实现前接口应返回明确 `501 Not Implemented` 或隐藏入口。
- 真实现需要持久化 schedule，并接入 scheduler lifecycle。

## P2 功能完整性与一致性问题

### 8. 邮件模板处理在 composer 中明确未接入

证据：

- `StandardEmailComposer.processTemplate` 返回 `template processing requires TemplateService dependency injection - will be implemented in service integration`，见 `backend/internal/services/email_composer.go:365`。

影响：

- 如果 compose request 携带 template ID，会直接失败。
- 代码里已经存在 `EmailTemplateService`，但 composer 与模板服务的依赖边界没有打通。

建议：

- 要么通过依赖注入接入 `EmailTemplateService`，要么从公开 request 中移除模板入口，避免半成品 API。

### 9. 定时邮件调度的状态机不完整

证据：

- scheduler 只查询 `status = "scheduled" AND scheduled_at <= now`，见 `backend/internal/services/scheduled_email_service.go:93`。
- 失败后 `updateScheduledEmailError` 会把状态置为 `retry` 并设置 `next_attempt`，见 `backend/internal/services/scheduled_email_service.go:171`。
- 但查询条件不包含 `status = "retry" AND next_attempt <= now`，所以失败后的重试不会再被处理。
- 发送调用 `emailSender.SendEmail` 是异步返回；`processScheduledEmail` 只要成功入队就立刻把队列状态改成 `sent`，见 `backend/internal/services/scheduled_email_service.go:143`。

影响：

- 定时邮件可能显示已发送，但真实 SMTP 发送仍在后台且可能失败。
- retry 状态可能永久卡住。

建议：

- 将队列状态分为 `scheduled/processing/sending/sent/failed/retry`，并由最终 SMTP 结果驱动。
- 查询到期任务时同时处理 `scheduled` 和到期 `retry`。

### 10. 批量发送结果切片存在并发写入数据竞争

证据：

- `SendBulkEmails` 为每封邮件开 goroutine，多个 goroutine 并发 `results = append(results, result)`，见 `backend/internal/services/email_sender.go:141`。

影响：

- 并发 append 会导致数据竞争、结果丢失或内存破坏风险。
- 当前普通 `go test ./...` 不带 `-race`，不会发现该问题。

建议：

- 使用带索引的预分配切片或 mutex/channel 收集结果。
- 增加 `go test -race ./...` 或至少对该服务加并发单测。

### 11. 邮件移动和归档路径可能忽略 OAuth2 token refresh callback

证据：

- 读未读路径创建 provider 后调用 `setupProviderTokenCallback`，见 `backend/internal/services/email_service.go:1503`。
- `MoveEmail` 直接 `CreateProvider(account.Provider)` 并连接，未调用 token callback，见 `backend/internal/services/email_service.go:1866`。
- `findOrCreateArchiveFolder` 同样直接 `CreateProvider(account.Provider)` 并连接，见 `backend/internal/services/email_service.go:3223`。

影响：

- OAuth2 账户在移动、归档、创建归档文件夹时若触发 token refresh，更新后的 token 可能不能回写数据库。
- 结果可能表现为部分路径频繁要求重新授权，而读未读/同步路径正常。

建议：

- 统一使用 `CreateProviderForAccount` 并设置 token callback。
- 通过 OAuth2 fake provider 覆盖 Move/Archive refresh 场景。

### 12. `SyncSpecificFolder` 使用 folder name 而不是完整 path

证据：

- `SyncSpecificFolder` 调用 `s.syncService.SyncFolder(ctx, account.ID, folder.Name)`，见 `backend/internal/services/email_service.go:3047`。
- 文件夹创建和移动等路径使用 `folder.Path` 或 `folder.GetFullPath()` 才能表达层级文件夹，见 `backend/internal/services/email_service.go:2040`。

影响：

- 嵌套文件夹或服务端显示名与路径不同的文件夹可能同步错误目标。

建议：

- 同步服务入口应明确接收 folder ID 或完整 path，而不是裸 name。

### 13. 文件夹移动父级明确未实现

证据：

- `UpdateFolder` 对 `ParentID` 更新直接返回 `moving folders between parents is not yet supported`，见 `backend/internal/services/email_service.go:2191`。

影响：

- API 表面允许传 `parent_id`，但实际不能移动文件夹。
- 前端如果提供拖拽/层级调整，会遇到运行时失败。

建议：

- 未实现前在 handler/request 层拒绝该字段并返回明确能力说明。
- 真实现需要递归更新子文件夹 path，并处理 IMAP rename/move 失败回滚。

### 14. 文件夹缺失只写日志，数据库没有无效状态

证据：

- `handleMissingFolder` 对自定义文件夹调用 `markFolderAsInvalid`，见 `backend/internal/services/sync_service.go:1052`。
- `markFolderAsInvalid` 只记录日志，并注释 TODO “可以考虑添加 is_valid 字段”，见 `backend/internal/services/sync_service.go:1065`。

影响：

- UI 和后续同步无法知道文件夹已在服务器消失。
- 每轮同步都可能重复尝试同一个无效文件夹。

建议：

- 增加 folder 状态字段，如 `is_valid`、`last_error`、`server_missing_at`。
- 同步、列表、UI 共同消费该状态。

### 15. 附件预览是占位实现

证据：

- 图片预览只读取前 1KB 原始字节，见 `backend/internal/services/attachment_service.go:434`。
- PDF 预览固定返回 `PDF document preview not implemented`，见 `backend/internal/services/attachment_service.go:452`。

影响：

- 图片预览并不是可显示缩略图。
- PDF 预览对用户没有实际价值。

建议：

- 先定义 API 契约：预览返回文本摘要、缩略图 bytes、还是预签名下载 URL。
- 未实现类型应返回能力错误，而不是伪预览内容。

### 16. 附件存储配置声明支持压缩/校验，但实现没有落库

证据：

- `Store` 注释说明 checksum 字段不在模型，见 `backend/internal/services/attachment_storage.go:143`。
- `GetStorageInfo` 总是 `IsCompressed: false`，见 `backend/internal/services/attachment_storage.go:251`。

影响：

- 配置项 `CompressText` 和 `ChecksumType` 容易让调用方误以为已完成端到端能力。
- 文件完整性校验结果只在查询时临时计算，不能用于持久化审计。

建议：

- 要么删掉未使用配置，要么补模型字段和迁移。

### 17. Gmail/Outlook 专用去重信息提取为空实现

证据：

- Outlook `extractConversationID` 固定返回空字符串，见 `backend/internal/services/outlook_deduplicator.go:191`。
- Gmail `extractGmailLabels` 固定返回空数组，见 `backend/internal/services/gmail_deduplicator.go:248`。

影响：

- Gmail 标签系统和 Outlook conversation/thread 信息不能参与去重合并。
- 跨文件夹/跨标签场景可能保留或合并错误。

建议：

- 明确 provider-specific metadata 的来源：IMAP headers、Gmail IMAP extension、Graph/Gmail API。
- 在没拿到可靠字段前，不要在报告中声称支持 Gmail label 或 Outlook conversation 去重。

### 18. provider capability 检测存在伪成功

证据：

- `testSMTPConnection` 注释写需要实现，实际 `return true, nil`，见 `backend/internal/providers/capabilities.go:314`。
- IDLE 检测也只是基于 provider info 返回，见 `backend/internal/providers/capabilities.go:344`。

影响：

- 连接测试或能力报告会把 SMTP 能力误报为可用。
- 用户创建账户时可能通过能力检测，但真实发送失败。

建议：

- SMTP 测试应进行真实 TCP/TLS/认证握手，或明确返回 “unknown/not tested”。

### 19. 新浪 provider TODO 仍未实现

证据：

- provider factory 注释 `TODO: 实现新浪邮箱提供商`，见 `backend/internal/providers/factory.go:34`。

影响：

- 如果配置或前端出现 `sina`，后端会返回 `unknown provider: sina`。

建议：

- 如果近期不支持，应从产品选项和配置文档中移除；如果支持，应补 provider、配置、测试。

## P3 技术债与可维护性问题

### 20. 回复主题构造存在重复赋值

证据：

- `ReplyEmail` 中当 `replySubject == ""` 时先无条件赋值 `"Re: " + originalEmail.Subject`，随后再判断是否已有 `re:`，见 `backend/internal/services/email_service.go:2726`。

影响：

- 行为最终大概率正确，但代码可读性差，容易引入重复 `Re:` 的维护错误。

建议：

- 简化为单一条件分支。

### 21. HTML 清理策略会转义全部 HTML，而不是 sanitize

证据：

- `sanitizeHTML` 注释称应使用专门 HTML 清理库，实际直接 `html.EscapeString`，见 `backend/internal/services/email_composer.go:358`。

影响：

- 如果用户希望发送富文本 HTML，会被全部转义成文本。
- 如果后续某处绕过该函数，又缺少真正 sanitizer。

建议：

- 明确 compose 支持纯文本、受限 HTML，还是原样 HTML；根据契约选择 sanitizer。

### 22. MIME header 和文件名编码不符合 RFC 2047/2231

证据：

- `encodeHeader` 注释写“实际应该使用 RFC 2047 编码”，实际原样返回，见 `backend/internal/services/email_composer.go:572`。
- `encodeFilename` 也原样返回，见 `backend/internal/services/email_composer.go:577`。

影响：

- 中文主题、中文附件名或特殊字符在部分客户端显示异常。

建议：

- 用标准库 `mime.WordEncoder` 处理 header；附件文件名按 RFC 2231/5987 或兼容策略处理。

### 23. 邮件发送事件 payload 信息不足

证据：

- `EmailServiceImpl.SendEmail` 发布 `NewEmailSendEvent` 时 sendID 和 emailID 都传空字符串，见 `backend/internal/services/email_service.go:1439`。

影响：

- SSE 客户端无法把发送完成事件关联到具体 compose/send 操作。

建议：

- 基础发送路径如果不走 `StandardEmailSender`，也应生成可追踪 send ID 或返回同步发送结果。

### 24. 搜索会修改 request 对象

证据：

- `SearchEmails` 解析 token 语法后直接写回 `req.From/To/Subject/Body/Query`，见 `backend/internal/services/email_service.go:2530`。

影响：

- 当前单次请求问题不大，但如果同一个 request 对象被复用或记录，会出现隐式副作用。

建议：

- 使用局部变量构造 effective filters，不修改入参。

### 25. 缓存失效策略是全量清空

证据：

- cache key 是纯 MD5，无法按 user 前缀删除，见 `backend/internal/services/email_service.go:2645`。
- `invalidateEmailListCache` 遍历并删除所有 email list cache key，见 `backend/internal/services/email_service.go:2697`。

影响：

- 任意用户一次读未读/删除/移动会清掉所有用户的邮件列表缓存。
- 低并发无感，高并发下会造成缓存抖动。

建议：

- cache key 保留可枚举前缀，例如 `emails:user:{id}:{hash}`，支持按用户失效。

### 26. 默认管理员弱密码仍可在未配置环境变量时创建

证据：

- `createDefaultAdmin` 在 `ADMIN_PASSWORD` 为空时使用 `admin123`，见 `backend/internal/database/database.go:208`。

影响：

- 生产首次部署如果漏配环境变量，会创建弱口令管理员。

建议：

- production 环境缺少 `ADMIN_PASSWORD` 时应启动失败；development 可以保留默认值但明确日志提醒。

### 27. 迁移 dirty recovery 会强制清除 dirty 标记

证据：

- migration service 对 dirty version 8 特判 force 到 7，其他 dirty version 直接 `Force(ctx, dirtyVersion)`，见 `backend/internal/database/migration/service.go:169`。

影响：

- `Force` 可能在未真正修复 schema 的情况下把迁移标记为 clean，后续迁移继续执行，留下半迁移数据库。

建议：

- 生产环境不应自动 force dirty migration。至少需要备份、人工确认、校验表结构。

## 未注册或疑似死链路清单

这些代码当前存在，但主入口没有注册或没有完整接线：

- `EmailSendHandler.RegisterRoutes`：扩展发送、批量发送、发送状态、重发、草稿、模板接口。
- `DeduplicationHandler.RegisterRoutes`：账户去重、用户去重、去重报告、计划去重接口。
- `backup.go` 中的备份管理 handler：`CreateBackup/ListBackups/RestoreBackup/DeleteBackup/ValidateBackup/CleanupOldBackups`，当前只启动自动备份服务，未暴露管理路由。
- `soft_delete.go` 中的软删除管理 handler：`GetSoftDeleteStats/CleanupExpiredSoftDeletes/RestoreSoftDeleted/PermanentlyDelete`，当前只启动自动清理服务，未暴露管理路由。
- `Auth.RefreshToken/ChangePassword/UpdateProfile` 等 handler 存在，但当前 auth 路由只注册 login/logout/me。

处置建议：不要简单“一次性全部注册”。应逐项核对权限、schema、测试和产品入口。特别是去重和备份/恢复类接口，一旦暴露就是高权限能力。

## 显性 TODO / not implemented 摘要

| 位置 | 当前行为 | 风险 |
| --- | --- | --- |
| `providers/factory.go` | 新浪 provider 注释 TODO | 配置或 UI 出现 sina 时不可用 |
| `services/email_sender.go` | 发送状态 DB 加载未实现 | 重启后状态丢失，状态查询不可靠 |
| `services/email_sender.go` | 重发加载已发送邮件未实现 | 重发接口必失败 |
| `services/email_composer.go` | 模板处理未注入服务 | 模板发送无法使用 |
| `services/email_service.go` | 文件夹父级移动未支持 | 层级管理半成品 |
| `services/sync_service.go` | 缺失文件夹 invalid 状态未落库 | UI/同步无法感知服务端缺失 |
| `services/deduplication_manager.go` | 计划/取消去重只写日志 | 伪成功 |
| `handlers/deduplication_handler.go` | 权限校验固定允许 | 注册后跨账户风险 |
| `services/attachment_service.go` | 图片/PDF 预览占位 | 预览功能不可用 |
| `providers/capabilities.go` | SMTP 检测固定成功 | 能力报告误导 |
| `services/gmail_deduplicator.go` | Gmail label 提取空实现 | Gmail 去重质量不足 |
| `services/outlook_deduplicator.go` | Outlook conversation ID 空实现 | Outlook 去重质量不足 |

## 测试缺口

当前 `go test ./...` 通过，但至少缺少以下测试：

- 路由注册快照测试：验证所有预期 API 是否真实注册，防止 handler 存在但不可达。
- 真实迁移 schema 测试：从空库跑 migrations 后，对 GORM model 做关键 CRUD，尤其 `email_templates`、`sent_emails`、`send_queue`、`drafts/email_drafts`。
- `StandardEmailSender` 持久化状态、重发、异步失败状态流测试。
- `go test -race ./...` 覆盖批量发送并发写结果。
- 删除邮件远端失败时的强一致/弱一致行为测试。
- OAuth2 token refresh callback 在 Move/Archive/CreateArchiveFolder 路径的测试。
- 嵌套文件夹 `SyncSpecificFolder` 使用 path/name 的测试。
- 去重 handler 权限测试，注册前也可以对 handler 单测。
- 定时邮件 retry 状态机测试。
- 认证日志不泄露 token 的测试或日志审计约束。

## 建议修复顺序

1. 先修安全和误导性问题：去掉认证 token 日志；未实现接口返回 501 或不注册；去重权限校验不能空实现。
2. 再修 schema 真相源：补 `sent_emails/email_drafts/email_quotas/email_templates` 与迁移一致性，建立迁移后 CRUD 测试。
3. 再决定扩展发送模块是否正式启用：若启用，补状态持久化、重发、批量发送 race、定时邮件状态机。
4. 处理主邮件一致性：删除远端失败语义、OAuth2 callback、SyncSpecificFolder path。
5. 最后补增强体验功能：附件预览、Gmail/Outlook 专用去重、provider capability 真检测、RFC 编码。

