# 星桥 AI 部署与接入指南

## 1. 生成首次安装配置

在仓库根目录运行：

```bash
cd deploy/starbridge
python3 init-env.py --admin-email 你的管理员邮箱
```

脚本只创建一次 `.env`，不会覆盖已有文件。管理员初始密码在 `.env` 的 `ADMIN_PASSWORD` 中；登录成功后请在后台修改密码并配置 MFA。`.env` 只保存本机，不要提交到 Git 或发给他人。

## 2. 启动服务

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f gateway
```

看到 `gateway` 为 `healthy` 后，访问 `http://127.0.0.1:8080`。公网部署时，将 `BIND_HOST` 改为可信反向代理所在地址，并让反向代理负责 TLS、域名和访问控制。需要 Caddy 自动申请证书时，填写 `DOMAIN` 后执行 `docker compose --profile https up -d --build`。

## 2.1 使用 GitHub Actions 自动部署到独立服务器

仓库包含 `.github/workflows/deploy.yml`。它只同步代码到你指定的目录，并使用固定的 Compose 项目名 `starbridge` 启动自己的 `gateway`、PostgreSQL 和 Redis，不会执行其他项目的 `docker compose down`。

在 GitHub 仓库 **Settings → Secrets and variables → Actions → New repository secret** 中添加：

| Secret | 内容 |
| --- | --- |
| `DEPLOY_HOST` | 服务器 IP，例如 `177.0.143.11` |
| `DEPLOY_PORT` | SSH 端口；留空时使用 `22` |
| `DEPLOY_USER` | SSH 用户名，建议使用只负责部署的用户 |
| `DEPLOY_PATH` | 独立目录，建议 `/opt/starbridge` |
| `DEPLOY_APP_PORT` | 对外端口，选择一个未被占用的端口，例如 `18080` |
| `DEPLOY_ADMIN_EMAIL` | 首次初始化时的管理员邮箱 |
| `DEPLOY_SSH_KEY` | 上述用户的专用 SSH 私钥（完整 PEM 文本） |
| `DEPLOY_KNOWN_HOSTS` | 可选，服务器的固定 `ssh-keyscan` 输出；不填时工作流会临时执行 `ssh-keyscan` |

把 `DEPLOY_SSH_KEY` 对应的公钥放入服务器用户的 `~/.ssh/authorized_keys`。服务器需要安装 Docker Engine 24+、Docker Compose v2、Python 3 和 `curl`。首次推送到 `main` 或手动运行 **Starbridge Deploy** 后，工作流会在 `DEPLOY_PATH` 中创建 `.env`，并把 `BIND_HOST` 设置为 `0.0.0.0`、`APP_PORT` 设置为 `DEPLOY_APP_PORT`。

建议在服务器防火墙只放行你选择的端口，并使用 HTTPS 反向代理。不要把 `.env`、SSH 私钥或管理员密码提交到仓库。

## 3. 添加 OpenAI 或兼容上游

### OpenAI / ChatGPT Pro 登录账号

1. 登录后台，点击 **账号管理 → 添加账号**。
2. 选择 **OpenAI → OAuth**，点击下一步。
3. 点击生成授权链接，在打开的 OpenAI 页面登录你的 Pro 账号并授权；回到星桥后粘贴回调地址完成绑定。服务器无法直连 OpenAI 时，先给该账号选择可用代理。
4. 保存账号，点击测试；成功后在 **分组管理** 中创建 OpenAI 分组，例如“我的 Pro”。需要固定使用这个账号时，分组只绑定这一个 OAuth 账号。
5. 在 **API 密钥 → 创建密钥** 中选择“我的 Pro”分组，生成面向客户端的 Starbridge API Key。这个 Key 是给客户端调用的 Token，网关会使用已登录账号的 Codex 订阅用量。账号管理中可查看上游返回的用量窗口与重置时间。
6. 标准模式下，星桥用户还需要有站内余额或分组订阅。管理员可以在用户管理中分配站内余额；这个余额是站内记账，与 OpenAI Pro 的上游额度分别管理。

星桥保留上游的 OAuth 刷新能力，普通调用者只使用星桥 API Key。不要把客户端 API Key 发布到前端、日志、代码仓库或公共聊天中。可用模型及用量上限由 OpenAI 账号决定；Pro 不等于全部 API 模型都可用，也不会转换为 OpenAI Platform 的 API 余额。官方说明见 [Codex 身份验证](https://learn.chatgpt.com/docs/auth)。

授权回调包含敏感的一次性授权码，只粘贴到自己部署的星桥后台。真实登录、额度读取与模型调用需要部署后使用你的账号验收，本仓库测试使用模拟上游。

### API Key 上游账号

1. 登录后台，点击 **账号管理 → 添加账号**。
2. 选择 **OpenAI → API Key**。
3. 在 **上游快捷配置** 里选择服务商，确认 Base URL 和文本协议。
4. 填写上游 API Key。Ollama 这类不需要密钥的服务，可以填写 `ollama-local` 作为占位值。
5. 保存账号，点击测试；成功后在 **分组管理** 中创建服务分组并绑定账号。
6. 在 **密钥管理** 创建面向客户端的 API Key，只把客户端 Key 发给使用者，不要暴露上游 Key。

自定义兼容服务直接填服务商提供的 Base URL。若服务商只实现 `/v1/chat/completions`，选择 Chat Completions；只有确实支持 `/v1/responses` 的服务才选择 Responses。

新上游域名还需加入 `.env` 的 `UPSTREAM_HOSTS`（逗号分隔），再执行 `docker compose up -d`。多个同平台上游应配置准确的模型白名单，避免请求被调度到不支持该模型的账号。

### 本机 Ollama

Compose 已配置 Linux 的 `host-gateway`。先启动 Ollama、加载模型，并确保容器能连接到宿主机 11434 端口。生产场景推荐把 Ollama 放到带 HTTPS 的可信端点，再把端点域名加入 `UPSTREAM_HOSTS`。

只在你控制的本机测试环境中使用默认 HTTP 预设时，在 `.env` 设置下列选项并重建容器配置：

```dotenv
BIND_HOST=127.0.0.1
SECURITY_URL_ALLOWLIST_ENABLED=false
ALLOW_PRIVATE_UPSTREAMS=true
ALLOW_HTTP_UPSTREAMS=true
```

这是上游原有 HTTP 接入方式，会跳过 URL 白名单；不要将该配置用于不可信账号或公共服务。只设置 `ALLOW_PRIVATE_UPSTREAMS=true` 不会使 HTTPS 白名单接受 HTTP 地址。

## 4. 常见客户端

OpenAI SDK 的 `base_url` 指向站点的 `/v1`：

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-your-starbridge-key",
    base_url="https://your-domain.example/v1",
)
response = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": "你好"}],
)
print(response.choices[0].message.content)
```

客户端发送的模型名必须是分组允许的模型名。需要将一个公开名称映射为实际模型时，在账号的模型映射中填写“请求模型 → 实际模型”。

## 5. 安全与排错

- 生产环境保持 `SECURITY_URL_ALLOWLIST_ENABLED=true`、`ALLOW_PRIVATE_UPSTREAMS=false`、`ALLOW_HTTP_UPSTREAMS=false`。
- 只在受信任的内网中启用私有上游；不要让用户可控的 URL 指向云元数据地址、数据库或宿主机管理端口。
- 上游返回 401：检查账号 API Key、Base URL 是否重复 `/v1`、以及 Key 是否绑定到正确分组。
- 上游返回 404：确认协议与路径；Chat-only 服务不要强制 Responses。
- Ollama 连接失败：按上面的“本机 Ollama”配置检查协议、模型和宿主机可达性。
- 数据库或 Redis 反复重启：检查 `.env` 中密码是否为空、是否改过已创建数据卷的密码；先保留数据并检查服务日志。
