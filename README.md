# Starbridge AI

Starbridge AI（星桥 AI）是一个自托管的多模型 API 中转站，基于 [Sub2API](https://github.com/Wei-Shaw/sub2api) 适配。它让你用一套用户 API Key 连接多个上游 AI 服务，并提供账号池、智能调度、模型分组、用量计费、限流和管理后台。

## 已做的适配

- 品牌默认名称改为 **Starbridge AI**，保留后台设置中的自定义站点名称和 Logo。
- 添加 OpenAI 上游快捷配置：OpenAI、DeepSeek、SiliconFlow、OpenRouter、Ollama。
- 支持 OpenAI OAuth：管理员可以登录 ChatGPT / OpenAI Pro 账号，把订阅账号加入账号池，再生成星桥 API Key 使用该账号的 Codex 订阅用量（受账号权限和用量限制）。
- OpenAI 账号默认使用设备码授权，适合手机与远程服务器；绑定结果在当前站点接收，保留手动回调和会话导入。
- 手机弹窗只纵向滚动，长授权链接换行，组合线路以卡片布局显示。
- 支持同一平台绑定多个 OAuth / API Key 账号，并为同一公开模型配置跨平台主备线路；主账号池不可用时按优先级自动切换到下一条兼容线路。
- 预设会同时填入 Base URL 和文本协议：OpenAI 使用 Responses；其他预设默认使用 Chat Completions。
- 支持继续手动填写任何 OpenAI-compatible Base URL、API Key 和模型 ID。
- 提供 PostgreSQL、Redis、可选 Caddy HTTPS 的 Docker Compose 部署模板。
- 首次安装脚本生成 256 位随机管理员、数据库、Redis、JWT 和 TOTP 密钥，并以 `0600` 保存 `.env`。
- 默认关闭批量图片 / Vertex 扩展，降低普通中转站的初始运行面；需要时可以在 Compose 中打开。

## 快速部署

需要 Docker Engine 24+、Docker Compose v2 和一个可解析的域名（仅本机测试可以使用 `localhost`）。

```bash
cd deploy/starbridge
python3 init-env.py --admin-email you@example.com
# 查看 .env 中生成的 ADMIN_PASSWORD，并妥善保存
docker compose up -d --build
```

默认只绑定到 `127.0.0.1:8080`。生产环境建议在 Caddy、Nginx 或 Cloudflare Tunnel 后面提供 HTTPS；使用 Caddy 配置时执行：

```bash
docker compose --profile https up -d --build
```

详细的首次配置、上游接入、模型分组和客户端示例见 [中文部署与接入指南](docs/STARBRIDGE.md)。

已提供 [GitHub Actions 自动部署](docs/STARBRIDGE.md#21-使用-github-actions-自动部署到独立服务器)：推送 `main` 并通过 CI 后，在 GitHub 构建镜像再通过 SSH 部署。默认目录 `/opt/starbridge`、端口 `19090`，部署脚本会检查目录、Compose 资源和端口冲突。

## 上游接入

管理员登录后：**后台 → 账号管理 → 添加账号 → OpenAI → API Key**。点击上游快捷配置，填写对应 API Key，按需填写模型白名单或模型映射，再绑定到一个服务分组。

支持的协议取决于上游：

| 上游类型 | 常见 Base URL | 推荐协议 |
| --- | --- | --- |
| OpenAI 官方 | `https://api.openai.com` | Responses |
| DeepSeek | `https://api.deepseek.com` | Chat Completions |
| SiliconFlow | `https://api.siliconflow.cn/v1` | Chat Completions |
| OpenRouter | `https://openrouter.ai/api/v1` | Chat Completions |
| Ollama | `http://host.docker.internal:11434/v1` | Chat Completions |
| 其他兼容服务 | 由服务商提供 | 以服务商文档为准 |

Ollama 预设指向 Docker 宿主机；Compose 已加入 Linux 的 `host-gateway` 映射，仍需按指南配置可信内网和 HTTP 访问。不要把本地 Ollama 端点暴露到公网。

## API 兼容性

网关保留 `/v1/chat/completions`、`/v1/responses`、模型列表和流式 SSE 等兼容入口，并按账号配置把 Responses 请求桥接到只支持 Chat Completions 的上游。OpenAI 官方文档建议新项目使用 Responses API，同时 Chat Completions 仍可用：

- [OpenAI Responses API migration guide](https://developers.openai.com/api/docs/guides/migrate-to-responses)
- [OpenAI Chat Completions API reference](https://developers.openai.com/api/docs/api-reference/chat)

## 来源与许可证

本仓库是在 Sub2API 上进行的适配版本，保留原项目的 LGPL-3.0 许可证和上游版权声明。`README_UPSTREAM.md` 与 `.github/upstream-workflows/` 保存了上游对应资料，便于同步更新时对照。
