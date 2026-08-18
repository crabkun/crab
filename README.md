# Crab

用 Go 写的内网穿透工具。公网机器只开 **一个 TCP 端口**，就能同时给多个用户穿透多个内网端口；被穿透的服务不会暴露在公网上。

常见穿透工具每映射一个端口就要在公网上再开一个端口，连上就能被扫到。Crab 不是这种模型：流量始终从你自己的 client 走进去，外人看不到、也连不上家里的 3389 或 22。

## 它适合什么场景

家里有公网服务器，但你并不想把远程桌面、数据库、游戏服直接挂到公网上。或者你买的是便宜大带宽 NAT 机器，只能开十来个端口，传统工具穿透几个服务就把配额吃光了。

典型用法：

| 你在哪 | 跑什么 | 结果 |
| --- | --- | --- |
| 公网服务器 | master | 只监听一个端口，给所有人中转 |
| 家里电脑 | server | 把 3389、3306、22 等注册上去 |
| 外面的电脑 | client | 连本地的 13389，就等于连家里的 3389 |

也可以只把配好的 client 发给朋友。朋友连自己电脑上的 `127.0.0.1:27015`，流量进你家的游戏服；公网上看不到 27015。不想对外开放的 Web、SSH 也是同一套做法。

## 三个角色

同一个二进制，靠配置里的 `mode` 区分身份。

**master** 跑在公网，只负责转发。server 和 client 都连它。它不碰加密和压缩，也不接触你内网的真实端口。

**server** 跑在有服务的那台内网机器上。配置里写清要穿透的地址和 `port_key`，启动后向 master 注册。只有拿着同一把 `port_key` 的 client 才能连进来。

**client** 跑在你想发起连接的那一侧。配上同样的 `port_key`，在本地听一个端口。有人连这个本地端口时，流量经 master 转到 server 背后的真实地址。

两把钥匙：

- `master_key`：server 向 master 注册时用。对上了才能挂端口，朋友的 server 要用你的 master，把这把钥匙给他就行。
- `port_key`：每一路穿透自己的钥匙。client 和 server 必须一致，也用作加密密钥的原料。不同端口请用不同的 key。

## 下载

到 [GitHub Releases](https://github.com/crabkun/crab/releases) 按系统下载压缩包，解压即可用。Windows 是 `.zip`，其他平台是 `.tar.gz`。也提供 `SHA256SUMS` 做校验。

解压后把 `config-examples/` 里对应角色的样例拷成 `config.json`，改好地址和密钥再启动：

```bash
./crab --config config.json
```

## 从源码构建

默认读当前目录的 `config.json`，也可以 `--config` 另指定。需要 Go 1.26 及以上。

```bash
make build
./bin/crab --config config.json
```

顶层字段可以用环境变量覆盖，前缀 `CRAB_`，例如 `CRAB_MASTER_KEY`、`CRAB_LISTEN_AT`、`CRAB_LOG_LEVEL`。

公网 master 用 Docker 部署：

```bash
# 先改 deploy/master.json 里的 master_key
docker compose up -d --build
```

容器默认读 `/etc/crab/config.json`，以非 root 运行，映射宿主机 `51324`。也可以自己构建：

```bash
docker build -t crab:2.0.0 .
docker run --rm \
  -v "$PWD/deploy/master.json:/etc/crab/config.json:ro" \
  -p 51324:51324 \
  crab:2.0.0
```

想在本机把整条链路跑通（master + 一台只在内网的 nginx + server + client）：

```bash
docker compose --profile demo up -d --build
curl -I http://127.0.0.1:18080/
```

演示里的 nginx 不对外发布。你访问本机 `18080` 时，流量是 client → master → server → nginx。

## 配置

三种角色共用一份 JSON，按 `mode` 取自己需要的字段。下面这一组配完之后：外面连 `127.0.0.1:13389` / `127.0.0.1:122`，就等于连家里的远程桌面和 SSH。

### master

```json
{
  "mode": "master",
  "log_level": "debug",
  "listen_at": "0.0.0.0:51324",
  "master_key": "crabserver"
}
```

| 字段 | 说明 |
| --- | --- |
| `mode` | `master` / `server` / `client` |
| `log_level` | `debug` / `info` / `warn` / `error` |
| `listen_at` | master 监听地址 |
| `master_key` | 注册口令，server 必须相同 |

### server

```json
{
  "mode": "server",
  "log_level": "debug",
  "master": "crab.myserver.com:51324",
  "master_key": "crabserver",
  "ports": [
    {
      "mark": "远程桌面",
      "local_address": "127.0.0.1:3389",
      "port_key": "T6wjoGQaqfDFDs1tySHVe8RXYYxnjWo4",
      "encrypt_method": "aes-128-cfb",
      "compress_method": "zstd"
    },
    {
      "mark": "SSH",
      "local_address": "127.0.0.1:22",
      "port_key": "lm6dQFjcaLyuVoy1Q4DqnunmnVz9b7CE",
      "encrypt_method": "aes-128-cfb",
      "compress_method": "zstd"
    }
  ]
}
```

| 字段 | 说明 |
| --- | --- |
| `master` | master 地址 |
| `master_key` | 与 master 一致才能注册 |
| `ports` | 要挂到 master 上的端口 |
| `ports.mark` | 备注，只出现在日志里 |
| `ports.local_address` | 内网真实服务地址 |
| `ports.port_key` | 这一路的钥匙，client 必须相同 |
| `ports.encrypt_method` | `plain` 或 `aes-128-cfb` |
| `ports.compress_method` | `null`、`s2` 或 `zstd` |

### client

```json
{
  "mode": "client",
  "log_level": "debug",
  "master": "crab.myserver.com:51324",
  "ports": [
    {
      "mark": "远程桌面",
      "local_address": "127.0.0.1:13389",
      "port_key": "T6wjoGQaqfDFDs1tySHVe8RXYYxnjWo4",
      "encrypt_method": "aes-128-cfb",
      "compress_method": "zstd"
    },
    {
      "mark": "SSH",
      "local_address": "127.0.0.1:122",
      "port_key": "lm6dQFjcaLyuVoy1Q4DqnunmnVz9b7CE",
      "encrypt_method": "aes-128-cfb",
      "compress_method": "zstd"
    }
  ]
}
```

| 字段 | 说明 |
| --- | --- |
| `master` | master 地址 |
| `ports.local_address` | 穿透成功后在本机监听的地址 |
| `ports.port_key` | 与对应 server 端口一致 |
| `ports.encrypt_method` | 必须与 server 一致 |
| `ports.compress_method` | 必须与 server 一致 |

## 加密和压缩

加密、压缩都发生在 server 和 client 两端，master 只转发已经处理好的字节流。

加密：

- `plain`：不加密
- `aes-128-cfb`：用 `port_key` 派生密钥

压缩：

| | 占用（CPU / 内存） | 压缩率 |
| --- | --- | --- |
| `zstd` | 高 | 高 |
| `s2` | 低 | 低一些 |
| `null` | 无 | 无 |

家里上行只有几十兆的时候，传尚未压缩过的大文件，zstd 往往比 s2 更快，因为它先把体积压下去了。如果传的本身就是 zip、视频这类已经压过的数据，两种算法都帮不上多少忙。机器比较慢时用 `s2`，或者直接 `null`。

## 常见问题

**已经有公网 IP 了，为什么还要用这个？**

两件事。第一，传统工具会把端口摊在公网上。我穿透是为了自己在外面连回家，不是为了给全世界连，多一个端口就多一块攻击面。第二，不少便宜大带宽机器是 NAT，只能开有限几个端口。传统方案「一个内网端口 + 一个公网端口」，再加控制端口，额度很快就没了。Crab 用一个公网端口映射任意多个内网端口。

**朋友的 server 能挂到我的 master 上吗？**

可以。他把 master 地址和 `master_key` 配成跟你一样即可。每一路仍由各自的 `port_key` 隔离。

**还想做的：**

1. master 上的 port key 管理，能接到外部数据库
2. 按 port key 限速、限流量

## 许可

BSD 3-Clause。问题或改进欢迎提 issue / PR。
