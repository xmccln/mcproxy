# MCProxy - Minecraft 代理服务器

一个用Go语言编写的高性能Minecraft代理服务器，支持基于域名的路由转发功能。

## 🚀 功能特性

- **智能路由**: 根据客户端连接的域名自动转发到不同的Minecraft服务器
- **配置热重载**: 支持配置文件实时监听和热重载，无需重启服务
- **高性能**: 使用Go语言编写，支持高并发连接
- **简单配置**: 通过YAML配置文件轻松管理路由规则
- **默认路由**: 支持设置默认目标服务器，处理未匹配的域名
- **连接管理**: 智能处理连接关闭和错误恢复

## 📋 系统要求

- Go 1.21.0 或更高版本
- 支持的操作系统: Windows, Linux, macOS

## 🛠️ 安装与使用

### 1. 克隆项目

```bash
git clone <your-repository-url>
cd mcproxy
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 配置文件

编辑 `config/config.yaml` 文件：

```yaml
# 监听端口
listen_port: 25570

# 默认目标服务器（当域名不匹配任何路由时使用）
default_target: "mc.example.com:25565"

# 域名路由映射
routes:
  "localhost": "127.0.0.1:25565"
  "test.example.com": "mc.example.com:25565"
  "survival.myserver.com": "192.168.1.100:25565"
  "creative.myserver.com": "192.168.1.101:25565"
```

### 4. 运行服务器

```bash
go run main.go
```

或者编译后运行：

```bash
go build -o mcproxy.exe
./mcproxy.exe
```

## ⚙️ 配置说明

### 配置文件结构

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `listen_port` | int | 25570 | 代理服务器监听端口 |
| `default_target` | string | "mc.example.com:25565" | 默认目标服务器地址 |
| `routes` | map | - | 域名到服务器的映射关系 |

### 路由配置

在 `routes` 部分，你可以配置域名到Minecraft服务器的映射：

```yaml
routes:
  "域名": "目标服务器:端口"
  "creative.myserver.com": "192.168.1.100:25565"
  "survival.myserver.com": "192.168.1.101:25565"
```

## 🎮 使用示例

### 场景1: 多服务器管理
如果你有多个Minecraft服务器（如生存、创造、小游戏等），可以通过不同域名访问：

```yaml
routes:
  "survival.myserver.com": "192.168.1.100:25565"
  "creative.myserver.com": "192.168.1.101:25565"
  "minigames.myserver.com": "192.168.1.102:25565"
```

### 场景2: 开发环境
在开发环境中，可以轻松在本地和远程服务器之间切换：

```yaml
routes:
  "localhost": "127.0.0.1:25565"
  "dev.myserver.com": "dev-server.example.com:25565"
  "prod.myserver.com": "prod-server.example.com:25565"
```

## 🔧 工作原理

1. **连接监听**: 代理服务器监听指定端口的传入连接
2. **握手解析**: 解析Minecraft客户端的握手包，提取目标服务器域名
3. **路由匹配**: 根据域名查找对应的目标服务器
4. **数据转发**: 建立到目标服务器的连接，双向转发所有数据包
5. **连接管理**: 处理连接关闭和错误情况

## 📝 日志输出

运行时会显示详细的日志信息：

```
TCP服务器已启动，监听端口: 25570
使用配置文件: d:\poject\Go\mcproxy\config\config.yaml
客户端请求连接到: survival.myserver.com
找到域名映射: survival.myserver.com -> 192.168.1.100:25565
已连接到目标服务器: 192.168.1.100:25565
```

## 🛡️ 注意事项

- 确保目标Minecraft服务器正常运行且可访问
- 防火墙需要允许代理服务器的监听端口
- 建议在生产环境中使用进程管理工具（如systemd、PM2等）
- 配置文件修改后会自动重载，无需重启服务

## 🤝 依赖项

- `github.com/fatedier/golib` - 网络工具库
- `github.com/fsnotify/fsnotify` - 文件监听
- `github.com/spf13/viper` - 配置管理

## 📄 许可证

[在此添加许可证信息]

## 🐛 问题反馈

如果遇到问题或有改进建议，请创建Issue或Pull Request。

---

**注意**: 这是一个Minecraft服务器代理工具，请确保遵守相关服务条款和法律法规。