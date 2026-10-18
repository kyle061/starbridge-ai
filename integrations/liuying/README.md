# 留影：微信小程序视频解析与原片保存

此目录是现有 `starbridge-ai` 仓库内的独立功能目录，不包含新的 Git 仓库。原生小程序、网页工作台和后端使用同一套真实解析与下载接口。

## 当前接入

- 服务入口：`https://starbridaeai.top/liuying/`
- HTTPS 及解析服务部署目标：`177.0.143.11`，目录 `/opt/starbridge/integrations/liuying`。
- 沿用现有 Caddy 和证书，新增 `/liuying/` 路由；容器 `liuying-video`，宿主机端口仅监听 `127.0.0.1:18787`。
- 当前 AppID：`wx06427263aaf1143c`，微信测试号，只能开发联调，不能审核发布。
- 服务器 AppSecret、会话密钥、工作台访问码仅存在服务器 `.env`，不随代码提交。

部署后的实测结果见 `VERIFICATION.md`。接入解析器不等于所有视频都可下载。

## 已实现

分享文案提取链接、平台识别、异步解析、视频预览、原画质下载、相册兼容模式、下载进度与取消、重试、历史记录、删除、自动过期清理、微信登录和相册授权处理。

小程序“保存到相册”调用 `wx.downloadFile` 后调用 `wx.saveVideoToPhotosAlbum`；“保存原文件”通过微信文件分享面板导出下载的原文件。网页下载保存至浏览器指定目录。

原画质模式不重编码，分离的音视频会合并封装。相册兼容模式选择平台已有的 H.264/AAC MP4 源，也不重编码。没有兼容源时返回明确错误。平台可获取的最高画质不等于相机母版。

### 平台与无水印边界

已接入 yt-dlp 的 YouTube、抖音、TikTok、哔哩哔哩、Instagram、X/Twitter、Vimeo、Facebook 解析器。短链、服务器所在地区、平台变动、Cookie 要求都会影响可用性。不会把解析失败替换成示例视频。

不添加水印。若平台返回的源文件本来没有水印，保存结果也没有水印；已有画面水印会保留。项目不包含水印擦除、登录绕过、付费绕过或 DRM 解密。仅用于自己或已获授权且平台允许获取的视频。

抖音常要求有效 Cookie；未配置时如实返回凭证错误。管理员可将有权使用的 Netscape Cookie 文件只读挂载进容器，并设置 `COOKIES_FILE`。不要收集用户 Cookie 或把 Cookie 写入客户端。

## 本机运行

后端运行时没有 npm 依赖。需要 Node.js 22+、Python 3.10+（推荐 3.12）、FFmpeg/FFprobe。

```bash
python3 -m venv .venv
.venv/bin/pip install -r worker/requirements.txt
cp .env.example .env
```

在 `.env` 中设置 `PYTHON_BIN` 为 `.venv/bin/python` 的绝对路径，再运行：

```bash
npm start
```

本地入口 `http://127.0.0.1:8787/`。开发演示登录仅允许本机访问，生产环境禁用。内置 CC0 示例明确标为示例，可在解析引擎未安装时验证播放和下载。

如需重建图标，执行 `npm ci` 后运行 `node scripts/build-assets.mjs`；运行服务和导入小程序无需安装这些开发依赖。素材授权见 `web/assets/ATTRIBUTION.md`。

## 微信开发者工具

1. 导入本目录，工具会识别根目录的 `project.config.json` 和 `miniprogramRoot`。不需要创建新的代码项目。
2. 使用当前测试号 AppID 登录联调。`miniprogram/config.js` 已指向 HTTPS 服务，并启用真实 `wx.login`。
3. 微信后台的 request 和 downloadFile 合法域名均配置为 `https://starbridaeai.top`，不带 `/liuying` 路径。
4. 如需纯本地演示，将 `config.js` 改成 `apiBaseUrl: 'http://127.0.0.1:8787', demoMode: true`，仅在开发者工具中临时关闭域名校验。正式上传脚本会阻止演示配置。
5. 真机需验证粘贴授权、拒绝及重新开启相册权限、下载取消、后台切换、低存储空间及原文件导出。

## 正式发布

当前测试号不能发布。需要管理员通过[微信公众平台](https://mp.weixin.qq.com/)注册正式“小程序”账号，完成该主体所需的认证、备案及类目材料。主体信息、验证码和协议由账号本人确认。

1. 将正式 AppID 写入 `project.config.json`；服务器私有 `.env` 替换正式 `WX_APP_ID` 和 `WX_APP_SECRET` 后重建容器。密钥只留服务端。
2. 在正式小程序后台配置 request/downloadFile 合法域名；是否满足该账号要求，以微信后台校验为准。HTTPS IP 地址不能代替微信合法域名配置。
3. 补全小程序名称、图标、简介、实际服务类目、运营主体、隐私联系人和联系方式。根据实际经营地区与微信后台要求完成备案。
4. 配置用户隐私保护指引：剪贴板用于用户主动粘贴，相册用于用户主动保存；同步更新 `web/privacy.html` 中运营主体和联系人。项目默认说明不能直接作为正式运营者的完整隐私政策。
5. 真机验证功能及真实平台链接。平台凭证、跨地区访问或平台限制未解决前，不在介绍中承诺“全平台稳定无水印”。
6. 登录官方微信开发者工具，开启服务端口后执行：

```bash
WX_APP_ID=你的正式AppID RELEASE_VERSION=1.0.0 npm run upload
```

Windows/Linux 设置 `WX_DEVTOOLS_CLI` 为官方开发者工具 CLI 路径。上传脚本会拒绝当前测试号。上传成功只生成开发版本；管理员仍需在微信后台提交审核，通过后点击发布。

## 服务端维护

在部署目录执行 `bash scripts/deploy-server.sh` 构建启动。要求现有 `starbridge_default` 网络和私有 `.env`。数据目录独立；默认上限 200 MB、20 分钟、保留一小时，线上并发设为 1。机器内存不足时先降低并发与大小上限。

```bash
docker compose -p liuying -f deploy/compose.yaml logs --tail=100 liuying
docker compose -p liuying -f deploy/compose.yaml restart liuying
```

任务保存在内存中。重启会丢弃任务并清理专用数据目录中的旧视频；不适用于需要长期保管素材的场景。扩展多实例前须将会话密钥保持一致，并改为共享任务和文件存储。

访问码仅供网页工作台测试，通过服务器 `PREVIEW_ACCESS_KEY` 配置。小程序通过微信登录取得七天会话。下载地址签名最长十五分钟，历史记录重新打开会刷新地址。不要公开传播带签名的地址。

后端限制支持的输入域名；工作进程校验实际 DNS 结果，拒绝私网及元数据地址；不执行外部网络下载器；容器使用非 root、只读文件系统、进程和内存限制。不要将解析服务当作任意 URL 代理。

## 验证

```bash
npm run check
npm test
python3 -m unittest discover -s tests -p '*_test.py'
```

测试覆盖链接校验、签名过期、用户任务隔离、范围下载、文件删除和私网访问拦截。真实平台可用性需使用获授权的实际链接单独验收。微信相册保存必须在真机确认，网页测试不能替代。

参考：[微信下载 API](https://developers.weixin.qq.com/miniprogram/dev/api/network/download/wx.downloadFile.html)、[保存视频到相册](https://developers.weixin.qq.com/miniprogram/dev/api/media/video/wx.saveVideoToPhotosAlbum.html)、[合法域名配置](https://developers.weixin.qq.com/miniprogram/dev/framework/ability/network.html)、[yt-dlp 支持站点与限制](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md)。
