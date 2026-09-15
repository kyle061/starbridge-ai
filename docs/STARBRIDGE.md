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

仓库包含 `.github/workflows/deploy.yml`。推送 `main` 且 **Starbridge CI** 全部通过后，它会根据服务器架构在 GitHub Runner 构建镜像，通过受限 SSH 通道上传镜像，再使用固定 Compose 项目名 `starbridge` 启动自己的 `gateway`、PostgreSQL 和 Redis。也可以在 **Actions → Starbridge Deploy → Run workflow** 手动运行 `main`。

默认独立目录为 `/opt/starbridge`，公网端口为 `19090`，服务地址为 `http://177.0.143.11:19090`（完成部署并放行端口后才能访问）。部署前会检查端口占用、目录归属，以及是否存在其他同名 Compose 资源；冲突时停止部署。PostgreSQL、Redis 不发布宿主机端口，自动部署不启动占用 80/443 的 Caddy。

首次接入时，管理员在服务器安装专用部署账号。先在自己的电脑生成专用 Ed25519 密钥，只将公钥和 `deploy/starbridge` 目录中的脚本上传到服务器，然后在该目录执行：

```bash
bash install-restricted-ssh.sh /path/to/actions.pub 19090
```

安装器创建 `starbridge-deploy` 账号，将部署配置安装为 root 所有，并添加只允许执行固定部署脚本的 sudo 规则。部署账号不能登录普通 Shell、执行任意 Docker 命令、修改 Compose 文件或访问其他项目。SSH 私钥只保存在本机和 GitHub Secrets。容器镜像导入前会校验标签，避免覆盖其他项目的镜像。

自动更新只替换星桥镜像。需要修改 Compose、资源上限或受限部署脚本时，由管理员检查后重新运行安装器。默认内存上限为 gateway 512 MiB、PostgreSQL 256 MiB、Redis 128 MiB，并分别限制 CPU，避免镜像编译和应用过载争抢已有服务的资源。

在 GitHub 仓库 **Settings → Secrets and variables → Actions → New repository secret** 中添加：

| Secret | 内容 |
| --- | --- |
| `DEPLOY_HOST` | 服务器 IP，例如 `177.0.143.11` |
| `DEPLOY_PORT` | SSH 端口；留空时使用 `22` |
| `DEPLOY_USER` | `starbridge-deploy` |
| `DEPLOY_APP_PORT` | 可选，默认 `19090`；必须是未占用的 1024–65535 端口 |
| `DEPLOY_ADMIN_EMAIL` | 首次初始化时的管理员邮箱 |
| `DEPLOY_SSH_KEY` | 上述用户的专用 SSH 私钥（完整 PEM 文本） |
| `DEPLOY_KNOWN_HOSTS` | 经过核对的服务器 SSH host key，使用 `known_hosts` 格式；非 22 端口的主机名需包含 `[IP]:端口` |

服务器需要 Docker Engine 24+、Docker Compose v2.20+、Python 3.6+、`sudo` 和 `curl`；安装器由管理员执行一次，之后 Actions 使用专用账号。服务器无需保存 GitHub 凭据，也无需安装 Node.js 或 Go。不要给部署账号加入 `docker` 或 `wheel` 组，也不要配置通用 sudo 权限。

可在可信终端使用 `ssh-keyscan -p 22 177.0.143.11` 获取主机公钥，并与服务器控制台显示的 SSH 主机指纹核对后存入 `DEPLOY_KNOWN_HOSTS`。缺少必填 Secret 时，工作流会在运行摘要标明跳过部署。

首次部署在 `/opt/starbridge/deploy/starbridge/.env`（或指定目录的对应路径）生成管理员和数据库密码。通过 SSH 在服务器本地查看其中的 `ADMIN_PASSWORD` 登录；后续部署保留已有密码、JWT/TOTP 密钥和数据卷，只更新绑定端口、绑定地址与镜像版本。健康检查通过才会报告成功。若容器未就绪，使用 `docker compose -p starbridge --env-file /opt/starbridge/deploy/starbridge/.env -f /opt/starbridge/deploy/starbridge/compose.yaml logs --tail=100 gateway` 检查日志。

建议在服务器防火墙只放行你选择的端口，并使用 HTTPS 反向代理。不要把 `.env`、SSH 私钥或管理员密码提交到仓库。

### 2.2 HTTPS 与证书自动轮换

有正式域名时，在 `.env` 中设置 `DOMAIN=你的域名`，保持 `CADDYFILE=./Caddyfile`，再启动 Caddy：

```bash
docker compose --profile https up -d caddy
```

没有域名时也可以直接为公网 IP 申请受浏览器信任的 Let’s Encrypt 短期证书。IP 证书有效期约 6 天，因此必须自动续期；Caddy 会使用持久化的 `caddy_data` 卷自动申请和轮换。先确认公网的 80/443 端口都指向本机且未被其他服务占用，然后设置：

```dotenv
CADDYFILE=./Caddyfile.ip
PUBLIC_IP=177.0.143.11
ACME_EMAIL=你的通知邮箱
```

再执行：

```bash
docker compose pull caddy
docker compose --profile https up -d caddy
docker compose logs --tail=100 caddy
```

首次启用 IP HTTPS 前，需要用本版本的 `install-restricted-ssh.sh` 重新安装一次部署配置，使服务器获得 `Caddyfile.ip` 和对应的 Compose 配置；该操作需要服务器 root 管理员执行。成功后入口为 `https://177.0.143.11`，应用端口 `19090` 可只保留作健康检查或在防火墙中限制来源。

也可以直接使用无需注册的免费通配 DNS，例如 `starbridge-177-0-143-11.nip.io` 会解析到 `177.0.143.11`。将它写入 `DOMAIN` 后即可按“正式域名”方式签发和自动续期普通 Let’s Encrypt 证书。该地址依赖第三方免费 DNS，适合 MVP；长期商用建议换成自己持有的域名。

## 3. 添加 OpenAI 或兼容上游

### OpenAI / ChatGPT Pro 登录账号

1. 登录后台，点击 **账号管理 → 添加账号**。
2. 选择 **OpenAI → OAuth**，点击下一步。
3. 默认选择 **设备码登录**，点击「获取登录设备码」，复制一次性设备码，再点击「打开 OpenAI 登录页面」。在官方页面登录自己的 Plus / Pro 账号并输入设备码，完成后切回原来的星桥标签页，账号会自动保存。设备码 15 分钟内有效；首次使用请先在 **ChatGPT → 设置 → 安全** 中开启设备码登录。服务器无法直连 OpenAI 时，先给该账号选择可用代理。
4. 保存账号，点击测试；成功后在 **分组管理** 中创建 OpenAI 分组，例如“我的 Pro”。需要固定使用这个账号时，分组只绑定这一个 OAuth 账号。
5. 在 **API 密钥 → 创建密钥** 中选择“我的 Pro”分组，生成面向客户端的 Starbridge API Key。这个 Key 是给客户端调用的 Token，网关会使用已登录账号的 Codex 订阅用量。账号管理中可查看上游返回的用量窗口与重置时间。
6. 标准模式下，星桥用户还需要有站内余额或分组订阅。管理员可以在用户管理中分配站内余额；这个余额是站内记账，与 OpenAI Pro 的上游额度分别管理。

星桥保留上游的 OAuth 刷新能力，普通调用者只使用星桥 API Key。不要把客户端 API Key 发布到前端、日志、代码仓库或公共聊天中。可用模型及用量上限由 OpenAI 账号决定；Pro 不等于全部 API 模型都可用，也不会转换为 OpenAI Platform 的 API 余额。官方说明见 [Codex 身份验证](https://learn.chatgpt.com/docs/auth)。

设备码登录通过当前站点接收授权结果，页面显示的绑定地址使用当前域名、协议和端口；无需访问 `localhost:1455`。传统「手动回调登录」保留为备用方式，其 `localhost` 回调属于 OpenAI 客户端注册配置，不能简单替换成任意站点域名。使用备用方式时，复制完整回调地址回到星桥粘贴即可。设备码不可用时，也可导入自己通过 Codex 官方客户端获得的会话。参考 [OpenAI 设备码登录说明](https://learn.chatgpt.com/docs/auth#preferred-device-code-authentication-beta)。

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
