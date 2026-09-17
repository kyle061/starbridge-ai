# Starbridge AI

Starbridge AI（星桥 AI）是一个自托管的多模型 API 网关。它提供统一的 API Key、账号池、智能调度、模型分组、用量计费、限流、支付和管理后台，帮助你在自己的域名和基础设施上管理 OpenAI、Anthropic、Google、DeepSeek 以及其他兼容服务。

本仓库是基于 [Sub2API](https://github.com/Wei-Shaw/sub2api) 的 Starbridge AI 适配版本。Starbridge 的用户界面、计费展示和部署文档使用 Starbridge 品牌；上游开源项目的版权、许可证和第三方服务条款仍然有效。

## 主要能力

- 自定义站点名称、Logo、副标题、客服和文档链接。
- OpenAI、Anthropic、Gemini、DeepSeek、Grok、Ollama 及 OpenAI-compatible 上游接入。
- API Key、OAuth、Passkey、TOTP 和可选的第三方 OAuth 登录。
- 账号池、优先级调度、模型分组、故障切换、并发和速率限制。
- 用户余额、订阅、优惠码、邀请返利、支付订单和账单导出。
- 用户侧倍率口径与管理端真实上游成本分离显示。
- GPT6 分组可先使用配置的低价模型生成内部需求文档，再由选定的 GPT6 模型完成请求；预处理模型不会作为客户看到的模型名返回。
- 登录前协议确认、使用政策、支持地区和服务特定条款，可由管理员在后台配置。

## 快速部署

需要 Docker Engine 24+、Docker Compose v2 和一个可解析的域名。本机测试可以使用 `localhost`。

```bash
cd deploy/starbridge
python3 init-env.py --admin-email you@example.com
docker compose up -d --build
```

默认入口绑定到 `127.0.0.1:8080`。生产环境建议使用 Caddy、Nginx 或 Cloudflare Tunnel 提供 HTTPS。完整的首次配置、上游接入、模型分组、支付和 OpenAI OAuth 说明见 [星桥部署与接入指南](docs/STARBRIDGE.md)。

## 上游接入

管理员登录后打开 **后台 → 账号管理 → 添加账号**，选择上游类型并填写服务商提供的 Base URL、API Key、协议和模型白名单。客户端只使用 Starbridge API Key，不要把上游凭据暴露给用户。

常见配置：

| 上游 | Base URL 示例 | 协议 |
| --- | --- | --- |
| OpenAI | `https://api.openai.com` | Responses |
| DeepSeek | `https://api.deepseek.com` | Chat Completions |
| SiliconFlow | `https://api.siliconflow.cn/v1` | Chat Completions |
| OpenRouter | `https://openrouter.ai/api/v1` | Chat Completions |
| Ollama | `http://host.docker.internal:11434/v1` | Chat Completions |

新上游域名还需要加入部署环境的 `UPSTREAM_HOSTS`。Ollama 等 HTTP 内网端点只应在可信网络中使用，不要直接暴露到公网。

## 支付与计费

支付配置位于 **后台 → 系统设置 → 支付**。管理员可以配置 Stripe、易支付、支付宝、微信支付及可用的自定义支付方式，设置充值上下限、订单超时、待支付订单数、手续费和商品名称。

支付回调必须使用 HTTPS，并且必须配置服务商要求的签名密钥。订单创建支持幂等键，用户可以按状态、订单类型和支付方式筛选订单并导出 CSV；管理端保留真实订单和成本口径，用户侧只显示经过站点倍率处理后的数据。

## 合规与许可证

部署者和运营者需要自行评估并遵守所在地、目标用户所在地、服务器所在地及上游服务商适用的法律法规、隐私义务、支付规则和服务条款。Starbridge 不代表 OpenAI、Anthropic、Google、DeepSeek 或其他上游服务商，也不替代它们的官方协议。

本仓库保留 Sub2API 上游项目的 LGPL-3.0 许可证和版权声明。详见 [README_UPSTREAM.md](README_UPSTREAM.md) 及仓库中的许可证文件。

## 文档

- [星桥部署与接入指南](docs/STARBRIDGE.md)
- [支付配置指南](docs/PAYMENT_CN.md)
- [支付研究记录](docs/AILINK_PAYMENT_RESEARCH_CN.md)
- [管理员合规承诺](docs/legal/admin-compliance.zh.md)
