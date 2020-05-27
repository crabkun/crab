## 关于Crab
一个用GO语言编写的内网穿透工具。与其他内网穿透工具不同，本工具仅需要公网服务器开放一个端口就可以支撑多个用户同时穿透多个端口，

## 特性
 1. 服务器仅需开放一个端口就可以支撑多个用户同时穿透多个端口  
 1. 穿透后的端口不会暴露在公网上，以免遭受攻击或入侵
 1. 支持流量加密（目前仅支持aes-128-cfb加密方式）
 1. 支持流量压缩（目前仅支持[s2](https://github.com/klauspost/compress/tree/master/s2#s2-compression)和[zstd](https://github.com/klauspost/compress/tree/master/zstd#zstd)两种压缩算法）

## 说明
本工具有三种身份：master、server、client

### master
master运行在你的公网服务器上，负责中转流量。仅需监听一个端口供多个server和多个client连接

### server
server运行在你的内网电脑上，在配置文件中定义好要穿透的端口以及其port key之后，server会将端口注册到master，并供持有相同port key的client连接使用

### client
client运行在你另一个网络的电脑上，client需要跟server配置同样的port key，然后client会在本地监听相同的端口，当有连接到端口时，client会将流量通过master穿透转发到server的端口上


## 使用场景举例
3台电脑：
 - 公网服务器
 - 家里电脑（开放了3389、3306端口）
 - 外面电脑（想要连接家里电脑的3389和3306端口）
 
公网服务器运行master  
家里电脑运行server，并把3389、3306端口注册到master上面。3389的port key是ABC123，3306的port key是CDE456  
外面电脑运行client，配置ABC123监听本地13389端口，配置CDE456监听本地13306端口。此时外面电脑连接他本地的13389、13306端口，就相当于连接到家里的3389、3306端口

亦或者  
你家电脑开游戏服务端(端口27015)和server，然后你配置好client发送给你朋友，你朋友连接他本地的127.0.0.1:27015，就会转发到你电脑的27015端口

亦或者  
你家电脑开了某些不想给外人访问的服务(端口80)和server，然后你配置好client发送给你朋友，你朋友连接他本地的127.0.0.1:80，就会转发到你电脑的80端口

## 配置文件

程序在默认情况下会自动加载运行目录下面的config.json文件，当然你也可以通过--config xxxx参数来指定配置文件路径  
config-examples目录下面有各角色的配置文件，可以参考

### master配置
```json
{
  "mode": "master",
  "log_level": "debug",
  "listen_at": "0.0.0.0:51324",
  "master_key": "crabserver"
}
```
|字段|解释|
| --- | ---|
|mode|程序的运行角色（可选master、server、client）|
|log_level|日志级别（可选debug、info、error）|
|listen_at|master角色特有配置，表示master监听在哪个端口上面|
|master_key|master/server角色特有配置，表示master密钥，当server与master配置一样的时候，server才能把端口注册到master|

### server配置
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

|字段|解释|
| --- | ---|
|mode|程序的运行角色（可选master、server、client）|
|log_level|日志级别（可选debug、info、error）|
|master|master服务器的地址|
|master_key|master/server角色特有配置，表示master密钥，当server与master配置一样的时候，server才能把端口注册到master|
|ports|需要注册到master的端口列表|
|ports.mark|端口备注（用于日志排错用）|
|ports.local_address|需要穿透的本地端口|
|ports.port_key|端口的port_key，当server与client配置一样的时候，client才能把流量转发到此端口|
|ports.encrypt_method|加密方式（可选plain、aes-128-cfb）|
|ports.compress_method|压缩方式（可选null、s2、zstd）|




### client配置
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

|字段|解释|
| --- | ---|
|mode|程序的运行角色（可选master、server、client）|
|log_level|日志级别（可选debug、info、error）|
|master|master服务器的地址|
|ports|需要连接的端口列表|
|ports.mark|端口备注（用于日志排错用）|
|ports.local_address|此端口穿透成功后在本地监听的地址|
|ports.port_key|端口的port_key，当server与client配置一样的时候，client才能把流量转发到此端口|
|ports.encrypt_method|加密方式（可选plain、aes-128-cfb），必须与server一致|
|ports.compress_method|压缩方式（可选null、s2、zstd），必须与server一致|


上面三个配置文件，运行起来最终效果是：client连接127.0.0.1:13389或127.0.0.1:122，就相当于连接到master的3389或22端口

## 压缩算法的选择
资源占用（CPU、内存）：zstd > s2  
压缩率：zstd > s2  
PS：资源占用指的是server和client的资源占用，master不参与压缩和解压所以压缩算法对master无影响  

总的来说，zstd资源占用高，但压缩率也很高。我家35兆上传带宽，在外面穿透回家下载某个PS2游戏ISO文件，s2跑出了64兆的平均速度，zstd跑出了104兆的平均速度。当然这个与文件有关，如果你传输的数据原本就已经被压缩过，压缩算法对速度的提升就非常小了。  
如果server的机器配置比较低，建议还是用s2压缩或者不压缩

## 常见问题
### 1.我都有公网IP服务器了，为啥不用其他端口穿透工具？  
这个要看你个人需求，就我个人来讲：  
 1. 传统的端口穿透工具会让穿透后的端口暴露在公网。考虑到安全问题，我不想把端口暴露在公网。而且我穿透端口不是为了分享给别人连接，是为了能让自己在外面能连接回来
 1. 我购买的服务器是便宜且大带宽NAT服务器，但缺点是只能开放10个端口。 传统的端口穿透工具每穿透一个端口都要占用一个端口，再加上其本身的控制端口，就剩下更少了。所以我需要一个仅占用一个端口就能映射N个端口的工具

### 2.我公网上的master服务器，可以让我朋友的server也注册上来吗？
可以，只需要你朋友的server配置好你的master地址和相同的master key，他也能把端口注册到你的master上面来

### 3.后续更新功能？
 1. master的port key管理优化，支持接入第三方数据库
 1. master针对port key进行限速、限流量

## 遇到了问题？
欢迎提issue或Pull Request

## 开源协议
BSD 3-Clause License
