# 网关内存限制与 OOM 排查

gateway 的 Docker 内存硬上限默认为 1 GiB，Go 内存软目标仍为 384 MiB。旧版 512 MiB 硬上限在生产故障中曾触发 OOM：内核记录 `Memory cgroup out of memory` 并杀死 `sub2api`，Docker 随后重启容器，Caddy 因 EOF、`gateway` 暂时不可解析或连接被拒绝而返回空的 502。

增加硬上限为短时分配提供余量，但不能证明或修复内存泄漏。`GOMEMLIMIT` 是 Go 运行时的软目标，不是进程 RSS 硬上限；活跃对象、运行时之外的内存和分配高峰仍可能超过它。持续增长应结合具体请求和内存采样另行定位，不应反复扩大容器限制。

## 持久化配置

在部署 `.env` 中配置：

```dotenv
GATEWAY_MEMORY_LIMIT=1g
GOMEMLIMIT=384MiB
```

宿主机还需容纳 PostgreSQL、Redis、Caddy、操作系统及其他服务。不要把宿主机全部内存分配给 gateway，也不要把 Go 软目标直接设为容器硬上限。`GATEWAY_MEMORY_LIMIT` 使用 Docker 内存单位，`GOMEMLIMIT` 使用 Go 支持的单位，例如 `MiB`。

GitHub 自动部署只替换镜像，服务器上的 Compose 文件不会随镜像自动更新。先由服务器管理员更新 `/opt/starbridge/deploy/starbridge/compose.yaml`，将 gateway 的两项设置改为：

```yaml
    mem_limit: ${GATEWAY_MEMORY_LIMIT:-1g}
    environment:
      GOMEMLIMIT: ${GOMEMLIMIT:-384MiB}
```

保留现有 `environment` 的其他条目。执行 `docker compose config --quiet` 验证配置；不要把包含密钥的完整 `docker compose config` 输出发到聊天或日志。以后需要改变 Go 软目标时，仅重建 gateway 即可：

```sh
cd /opt/starbridge/deploy/starbridge
docker compose -p starbridge --env-file .env -f compose.yaml config --quiet
docker compose -p starbridge --env-file .env -f compose.yaml up -d --no-deps --no-build gateway
```

重建会中断在途请求，应在合适的维护时段执行；不要重建数据库或清理数据卷。

## 现有容器在线增加硬上限

已确认宿主机有余量时，可以先更新现有 gateway 的硬上限，不重建容器。以下数值对应 1 GiB 内存、最多 2 GiB 内存与交换空间合计；交换空间是否可用还取决于宿主机和内核设置。

```sh
docker inspect --format 'Memory={{.HostConfig.Memory}} MemorySwap={{.HostConfig.MemorySwap}} RestartCount={{.RestartCount}} StartedAt={{.State.StartedAt}}' starbridge-gateway-1
docker update --memory 1g --memory-swap 2g starbridge-gateway-1
docker inspect --format 'Memory={{.HostConfig.Memory}} MemorySwap={{.HostConfig.MemorySwap}} RestartCount={{.RestartCount}} StartedAt={{.State.StartedAt}}' starbridge-gateway-1
```

同时更新 Compose 和 `.env`，避免下次部署恢复到旧限制。上述命令不会更改运行中进程的 `GOMEMLIMIT`。确认更新前后的启动时间、重启次数没有变化，再检查 `/health`。验证真实业务请求后，还应确认没有新增 OOM 和重启；一次健康检查不能证明高峰负载稳定。

## 确认退出原因

```sh
docker inspect --format 'OOMKilled={{.State.OOMKilled}} ExitCode={{.State.ExitCode}} RestartCount={{.RestartCount}}' starbridge-gateway-1
dmesg -T | grep -Ei 'oom|out of memory|killed process' | tail -n 24
```

容器重启后，当前 `OOMKilled=false` 或 `ExitCode=0` 不能否定历史 OOM，应与内核杀进程记录、Docker 退出事件和 Caddy 错误时间关联。没有普通 panic 日志也不能排除 OOM，内核强制杀进程时应用往往来不及记录错误。
