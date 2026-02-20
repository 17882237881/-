# Gemini 官网对标学习项目实施计划（第一优先：对话功能）

## 1. 本轮目标（按最新要求收敛）

- **先只完成“对话功能”**：多轮会话 + 流式输出 + 基础持久化 + 基础观测。
- **模型 API 固定为阿里云百炼**（DashScope / 百炼应用 API），第一阶段不做多模型抽象。
- **微服务之间“零直连”**：除与基建（Kafka / Redis / MySQL / Nginx）交互外，服务之间不互调、不共享内部存储。
- 保留 **Go + Python + gRPC**，并新增强制要求：**每个服务内部采用 DDD 模式开发**。

---

## 2. 总体架构原则（严格隔离 + 服务内 DDD）

### 2.1 跨服务原则
1. **服务自治**：每个服务只负责自己的限界上下文（Bounded Context）。
2. **禁止服务互调**：A 服务不能直接调用 B 服务（HTTP/gRPC 都不允许）。
3. **只经基建交换信息**：跨服务协作只能走 Kafka 事件或 Redis Stream。
4. **单写原则**：每个业务实体只有一个服务负责写入 MySQL。
5. **对外唯一入口**：Nginx + API Gateway。

### 2.2 服务内 DDD 原则（所有服务统一遵守）
1. **分层固定**：`interface(适配层) -> application(应用层) -> domain(领域层) -> infrastructure(基础设施层)`。
2. **领域纯净**：领域层不依赖具体框架/SDK（不直接依赖 gin/fastapi/kafka/mysql）。
3. **聚合根驱动写模型**：通过聚合根维护业务不变式（例如会话状态、消息顺序、幂等）。
4. **仓储接口放领域或应用层，仓储实现放基础设施层**。
5. **领域事件显式化**：领域事件映射为 Kafka 事件，避免隐式 side-effect。
6. **应用服务只编排，不承载业务规则**：业务规则归属领域服务/实体/值对象。

---

## 3. 第一阶段最小服务集（MVP）与 DDD 划分

### 3.1 api-gateway（Go）
- 责任：鉴权、限流、协议转换（HTTP/SSE -> gRPC）、trace 注入。
- DDD 视角：偏“防腐层/适配层”，**不承载核心领域模型**。
- 与外部交互：
  - 同步：gRPC 调用 `chat-command-service` 与 `chat-query-service`。
  - 异步：无（不直接消费业务事件）。

### 3.2 chat-command-service（Go）
- 限界上下文：`ConversationCommandContext`。
- 领域职责：接收用户 prompt、校验会话可写状态、生成 request_id、记录提交命令。
- 聚合建议：
  - `ConversationSession`（聚合根）
  - `UserPrompt`（实体）
  - `RequestId` / `TraceId`（值对象）
- 产出领域事件：`PromptSubmitted` -> Kafka `chat.prompt.submitted`。

### 3.3 llm-worker-service（Python）
- 限界上下文：`ModelInferenceContext`。
- 领域职责：消费提交事件、调用百炼、管理推理任务状态、输出 token 流。
- 聚合建议：
  - `InferenceTask`（聚合根）
  - `TokenChunk`（值对象）
  - `ProviderResponse`（值对象）
- 产出领域事件：`AnswerCompleted` / `AnswerFailed` -> Kafka `chat.answer.completed`。

### 3.4 chat-query-service（Go）
- 限界上下文：`ConversationQueryContext`。
- 领域职责：流式读取 token、组装可消费答案、提供历史查询、处理完成事件落库。
- 聚合建议：
  - `AnswerStream`（聚合根）
  - `MessageView`（读模型）
- 读写分离：
  - 写：消费完成事件后持久化最终答案。
  - 读：提供历史消息分页与 SSE 流透传。

> 说明：服务之间仍保持“零直连”；每个服务内部用 DDD 保证可维护性和可演进性。

---

## 4. 对话链路（无服务互调版本）

1. 前端 `POST /api/chat/send` 到 gateway。
2. gateway(gRPC) 调用 chat-command-service 创建 `request_id` 并记录提交。
3. chat-command-service 发布 Kafka 事件 `chat.prompt.submitted`。
4. llm-worker-service 消费事件并调用阿里云百炼流式接口。
5. llm-worker-service 将 token 增量写入 Redis Stream `stream:chat:{request_id}`。
6. 前端 `GET /api/chat/stream?request_id=...` 建立 SSE。
7. gateway 调用 chat-query-service，chat-query-service 从 Redis Stream 持续读取 token 并回推。
8. llm-worker-service 结束后发送 `chat.answer.completed` 到 Kafka。
9. chat-query-service 消费完成事件并落库答案，更新会话读模型。

---

## 5. 语言与技术栈（按最佳实践）

### 5.1 Go 服务最佳实践栈（gateway / command / query）
- **运行时与框架**：Go 1.22+、`grpc-go`、`connect-go`（可选）/ `gin`（仅网关 HTTP 层）。
- **DDD/CQRS 支撑**：
  - 命令与查询 handler 分离。
  - `wire` 或 `fx` 做依赖注入。
  - `go-playground/validator` 做输入校验。
- **持久化**：`sqlc`（优先，类型安全）+ `pgx/mysql driver`，或 `gorm`（快速迭代阶段）。
- **消息与缓存**：`segmentio/kafka-go` 或 `sarama`、`go-redis/v9`。
- **可观测**：OpenTelemetry + Prometheus + structured logging（`zap`/`zerolog`）。
- **工程质量**：`golangci-lint`、`testify`、`mockery`、`buf`（proto 管理）。

### 5.2 Python 服务最佳实践栈（llm-worker）
- **运行时**：Python 3.11+。
- **架构**：`FastAPI`（仅管理/健康接口）+ `grpcio`（若需 RPC 管理面）+ 独立 worker 进程。
- **DDD 支撑**：
  - `pydantic v2`（DTO/配置）
  - 应用层 use case + 领域层实体/值对象分离
  - 仓储协议使用 `typing.Protocol`
- **百炼接入**：官方 DashScope SDK（统一封装 `BailianClient` Anti-Corruption Layer）。
- **消息与缓存**：`aiokafka/confluent-kafka`、`redis-py`。
- **可观测**：OTel、`structlog`、Prometheus exporter。
- **工程质量**：`ruff`、`mypy`、`pytest`、`pytest-asyncio`。

---

## 6. 基建职责（第一阶段必须落地）

- **Nginx**：TLS、反向代理、连接保持、基础限流。
- **Kafka**：请求与完成事件总线（至少 2 topic + 1 DLQ）。
- **Redis**：流式 token 中转（Stream）+ 幂等键 + 短期上下文缓存。
- **MySQL**：会话与消息最终一致性存储。

---

## 7. 契约设计（先定协议再编码）

### 7.1 gRPC（仅 gateway -> service）

- `ChatCommandService.SubmitPrompt`
  - 入参：`user_id, session_id, prompt, request_id, trace_id`
  - 出参：`request_id, accepted_at, status`

- `ChatQueryService.StreamAnswer`
  - 入参：`request_id, trace_id`
  - 出参（stream）：`token, seq, is_final, finish_reason`

- `ChatQueryService.GetSessionMessages`
  - 入参：`session_id, page, page_size`
  - 出参：消息列表 + 分页游标

### 7.2 Kafka Topic

- `chat.prompt.submitted`
- `chat.answer.completed`
- `chat.answer.dlq`
- `chat.audit.log`（预留）

### 7.3 Redis Key

- `stream:chat:{request_id}`：token 流（XADD）
- `chat:req:meta:{request_id}`：请求元数据（TTL）
- `chat:idempotent:{request_id}`：幂等去重（TTL）

---

## 8. 阿里云百炼接入规范（MVP 必做）

1. llm-worker 内统一封装 `BailianClient`（防腐层），上层不直接依赖 SDK 细节。
2. 支持流式 chunk 回调，每个 chunk 写 Redis Stream，并写入序号 `seq`。
3. 出错时发布 `chat.answer.completed(status=failed, error_code, error_message)`。
4. API Key 只放环境变量/Secret，不入库、不写日志。
5. 记录指标：首 token 延迟、总耗时、输入/输出 token、失败率、重试次数。

---

## 9. 代码组织模板（每个服务统一 DDD 目录）

```text
services/<service-name>/
  cmd/
    server/
      main.go | main.py
  internal/ | src/
    interface/          # gRPC/HTTP handlers, consumer, presenter
    application/        # usecase, command/query handlers
    domain/             # entity, value object, aggregate, domain service, repository interface
    infrastructure/     # db repo impl, kafka producer/consumer, redis client, sdk client
    config/
    observability/
  tests/
    unit/
    integration/
```

---

## 10. 里程碑（先做对话 + DDD 落地）

### M1（第 1 周）：基础环境 + 协议 + 脚手架
- docker-compose 拉起 Nginx/Kafka/Redis/MySQL。
- 完成 proto、Kafka schema、Redis message schema。
- 生成三服务 DDD 目录骨架与基础 lint/test 流程。

### M2（第 2 周）：命令链路 + 百炼流式
- 完成 `SubmitPrompt -> Kafka`。
- 完成 `Kafka -> Bailian -> Redis Stream`。
- 前端看到首版 SSE 流式输出。

### M3（第 3 周）：查询链路 + 持久化 + 稳定性
- 完成 `StreamAnswer` 与历史查询。
- 完成 `chat.answer.completed` 消费落库。
- 增加幂等、重试、超时、DLQ、最小监控仪表盘。

---

## 11. 下一步执行清单（立即开始）

1. 先冻结 DDD 战术设计：聚合根、领域事件、仓储接口。
2. 输出三份契约：`proto`、`Kafka event schema`、`Redis Stream schema`。
3. 实现 command 服务（提交）与 llm-worker（百炼流式）最小闭环。
4. 实现 query 服务（拉流/历史）与完成事件落库。
5. 为每个服务补齐：健康检查、结构化日志、指标、trace、lint/test。

> 这版计划已按你的要求更新：**每个服务内部采用 DDD 模式开发，并按 Go/Python 最佳实践技术栈实施**。
