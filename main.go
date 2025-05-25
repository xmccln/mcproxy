package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	libnet "github.com/fatedier/golib/net"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var config = viper.New()

func initConfig() {
	//获取运行路径
	path, err := os.Getwd()
	if err != nil {
		path = "."
	}

	path = path + "/config"
	config.SetConfigType("yaml")
	config.SetConfigName("config") // 设置配置文件名
	config.AddConfigPath(path)
	config.AddConfigPath(".") // 添加当前目录作为备用路径

	// 设置默认值
	config.SetDefault("listen_port", 25570)
	config.SetDefault("default_target", "p2.example.com:25569")

	// 设置默认的域名映射
	config.SetDefault("routes", map[string]string{
		"p3.example.com": "127.0.0.1:25569",
		"localhost":      "127.0.0.1:25565",
	})

	// 读取配置文件
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("配置文件未找到，使用默认配置")
			// Create default configuration file
			// Ensure the directory exists
			if err := os.MkdirAll(path, 0755); err != nil {
				fmt.Printf("无法创建配置目录: %v\n", err)
			} else {
				configPath := path + "/config.yaml"
				if err := config.WriteConfigAs(configPath); err != nil {
					fmt.Printf("无法创建默认配置文件: %v\n", err)
				} else {
					fmt.Printf("已创建默认配置文件: %s\n", configPath)
				}
			}

		} else {
			fmt.Printf("读取配置文件错误: %v\n", err)
		}
	} else {
		fmt.Printf("使用配置文件: %s\n", config.ConfigFileUsed())
	}
	// 启用配置文件监听
	config.WatchConfig()
	config.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("配置文件已更新: %s\n", e.Name)
		routes := config.GetStringMapString("routes")
		fmt.Printf("当前路由配置: %+v\n", routes)
	})
}

func readVarInt(conn io.Reader) (int, error) {
	var value int
	var shift uint

	for {
		b := make([]byte, 1)
		if _, err := conn.Read(b); err != nil {
			return 0, err
		}

		value |= int(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("VarInt值过大")
		}
	}

	return value, nil
}

func readString(conn io.Reader, maxLength int) (string, error) {
	length, err := readVarInt(conn)
	if err != nil {
		return "", err
	}

	if length > maxLength {
		return "", fmt.Errorf("字符串长度超过最大值: %d > %d", length, maxLength)
	}

	if length == 0 {
		return "", nil
	}

	data := make([]byte, length)
	_, err = conn.Read(data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// 从握手包中读取服务器地址
func readServerAddressFromHandshake(conn io.Reader) (string, error) {
	// 读取包长度
	_, err := readVarInt(conn)
	if err != nil {
		return "", fmt.Errorf("读取包长度失败: %v", err)
	}

	// 读取包ID
	packetID, err := readVarInt(conn)
	if err != nil {
		return "", fmt.Errorf("读取包ID失败: %v", err)
	}

	// 确认是握手包
	if packetID != 0x00 {
		return "", fmt.Errorf("不是mc握手包，包ID: 0x%02X", packetID)
	}

	// 跳过协议版本
	_, err = readVarInt(conn)
	if err != nil {
		return "", fmt.Errorf("读取协议版本失败: %v", err)
	}

	// 读取服务器地址
	serverAddress, err := readString(conn, 255)
	if err != nil {
		return "", fmt.Errorf("读取服务器地址失败: %v", err)
	}

	return serverAddress, nil
}

func main() {
	// 初始化配置
	initConfig()

	// 从配置中获取监听端口
	listenPort := config.GetInt("listen_port")

	// 监听端口
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		log.Fatal("启动TCP监听器失败:", err)
	}
	defer listener.Close()

	fmt.Printf("TCP服务器已启动，监听端口: %d\n", listenPort)

	for {
		// 接受客户端连接
		conn, err := listener.Accept()
		if err != nil {
			log.Println("接受连接失败:", err)
			continue
		}

		// 为每个连接启动新的goroutine处理
		go handleConnection(conn)
	}
}

func proxyHandler(conn net.Conn, toconn string) {
	defer conn.Close()

	// 连接到目标服务器
	targetConn, err := net.Dial("tcp", toconn)
	if err != nil {
		log.Printf("连接到目标服务器 %s 失败: %v\n", toconn, err)
		return
	}
	defer targetConn.Close()

	fmt.Printf("已连接到目标服务器: %s\n", toconn)

	var wg sync.WaitGroup
	wg.Add(2)

	// 客户端到服务器的数据转发
	go func() {
		defer wg.Done()
		_, err := io.Copy(targetConn, conn)
		if err != nil && err != io.EOF {
			// 检查是否是连接重置或正常关闭
			if !isConnectionClosed(err) {
				log.Printf("数据转发到目标服务器失败: %v\n", err)
			}
		}
		// 关闭目标连接的写入端，通知服务器客户端已断开
		if tcpConn, ok := targetConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// 服务器到客户端的数据转发
	go func() {
		defer wg.Done()
		_, err := io.Copy(conn, targetConn)
		if err != nil && err != io.EOF {
			// 检查是否是连接重置或正常关闭

		}
		// 关闭客户端连接的写入端，通知客户端服务器已断开
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// 等待两个转发goroutine完成
	wg.Wait()
	fmt.Printf("代理连接关闭: %s <-> %s\n", conn.RemoteAddr(), toconn)
}

func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 检查常见的连接关闭错误
	return err == io.EOF ||
		err == io.ErrUnexpectedEOF ||
		errStr == "use of closed network connection" ||
		errStr == "wsasend: An established connection was aborted by the software in your host machine." ||
		errStr == "connection reset by peer" ||
		errStr == "broken pipe"
}

// 处理客户端连接
func handleConnection(conn net.Conn) {
	defer conn.Close()
	sc, rd := libnet.NewSharedConn(conn)

	// 读取服务器地址
	serverAddress, err := readServerAddressFromHandshake(rd)
	if err != nil {
		fmt.Printf("解析握手包失败: %v\n", err)
		return
	}

	fmt.Printf("客户端请求连接到: %s\n", serverAddress)

	// 从配置获取路由映射
	routes := config.GetStringMapString("routes")
	defaultTarget := config.GetString("default_target")

	// 查找对应的目标服务器
	var targetServer string
	if target, exists := routes[serverAddress]; exists {
		targetServer = target
		fmt.Printf("找到域名映射: %s -> %s\n", serverAddress, targetServer)
	} else {
		targetServer = defaultTarget
		fmt.Printf("使用默认目标: %s\n", targetServer)
	}

	proxyHandler(sc, targetServer)
}
