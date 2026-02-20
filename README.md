# Chat MVP (Frontend + Backend)

这是按计划落地的第一版“可运行前后端对话系统”：

- 前端：浏览器页面 `api/gateway-go/static/index.html`
- 网关：Go `cmd/gateway`
- 命令服务：Go `cmd/chat-command`
- 查询服务：Go `cmd/chat-query`
- LLM Worker：Python `services/llm-worker-py/worker.py`

> 说明：本地无外网依赖环境下，LLM Worker 使用**百炼兼容的本地 mock 流式输出**（先跑通链路）。后续可在 `worker.py` 内替换为真实百炼 SDK 调用。

## 启动

```bash
docker compose up --build -d
```

打开：`http://localhost:8080`

## 验证

```bash
./scripts_e2e.sh
```

## 停止

```bash
docker compose down
```
