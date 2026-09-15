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

## 3. 添加 OpenAI 或兼容上游

### OpenAI / ChatGPT Pro 登录账号

1. 登录后台，点击 **账号管理 → 添加账号**。
2. 选择 **OpenAI → OAuth**，点击下一步。
3. 点击生成授权链接，在打开的 OpenAI 页面登录你的 Pro 账号并授权；回到星桥后粘贴回调地址完成绑定。服务器无法直连 OpenAI 时，先给该账号选择可用代理。
4. 保存账号，点击测试；成功后在 **分组管理** 中创建服务分组并绑定这个 OAuth 账号。
5. 在 **密钥管理** 创建面向客户端的 Starbridge API Key。这个 Key 是给客户端调用的 Token，会由网关使用已登录的 OpenAI 账号转发请求，并消耗该账号的订阅额度。

星桥只在服务端保存 OAuth 刷新凭据，不把 OpenAI 刷新 Token 返回给客户端。不要把客户端 API Key 发布到前端、日志、代码仓库或公共聊天中。OpenAI 账号订阅状态和可用模型仍由 OpenAI 账号本身决定。

### API Key 上游账号

1. 登录后台，点击 **账号管理 → 添加账号**。
2. 选择 **OpenAI → API Key**。
3. 在 **上游快捷配置** 里选择服务商，确认 Base URL 和文本协议。
4. 填写上游 API Key。Ollama 这类不需要密钥的服务，可以填写 `ollama-local` 作为占位值。
5. 保存账号，点击测试；成功后在 **分组管理** 中创建服务分组并绑定账号。
6. 在 **密钥管理** 创建面向客户端的 API Key，只把客户端 Key 发给使用者，不要暴露上游 Key。

自定义兼容服务直接填服务商提供的 Base URL。若服务商只实现 `/v1/chat/completions`，选择 Chat Completions；只有确实支持 `/v1/responses` 的服务才选择 Responses。

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
- Ollama 连接失败：确认模型已加载、容器能访问宿主机端口，并仅在可信网络中使用 `ALLOW_PRIVATE_UPSTREAMS=true`。
- 数据库或 Redis 反复重启：检查 `.env` 中密码是否为空、是否改过已创建数据卷的密码；首次安装建议删除测试卷后重新初始化。
