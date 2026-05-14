package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/itick-org/go-sdk/sdk"
)

func main() {
	// 初始化客户端
	token := "8850*****************ee4127087"
	client := sdk.NewClient(token)

	// 设置 WebSocket 消息处理器
	client.SetMessageHandler(func(message []byte) {
		fmt.Printf("Received WebSocket message: %s\n", message)
	})

	// 设置 WebSocket 错误处理器
	client.SetErrorHandler(func(err error) {
		log.Printf("WebSocket error: %v\n", err)
	})

	// 测试 WebSocket 连接
	err := client.ConnectCryptoWebSocket()
	if err != nil {
		log.Printf("ConnectForexWebSocket error: %v", err)
		// Continue even if WebSocket fails
	} else {
		defer client.CloseWebSocket()

		// 发送订阅消息
		// subscribeMsg := []byte(`{"ac": "subscribe", "params": "BTCUSDT$ba","types":"quote,tick"}`)
		err = client.Subscribe([]string{"BTCUSDT$ba"}, []string{"quote", "tick"})
		if err != nil {
			log.Printf("SendWebSocketMessage error: %v", err)
		}

		err = client.Subscribe([]string{"ETHUSDT$ba"}, []string{"quote", "tick"})
		if err != nil {
			log.Printf("SendWebSocketMessage error: %v", err)
		}

		// 等待接收消息
		fmt.Println("Waiting for WebSocket messages...")
		time.Sleep(10 * time.Second)

		// 检查连接状态
		fmt.Printf("WebSocket connected: %v\n", client.IsWebSocketConnected())

		symbols, types := client.GetSubcribe()
		fmt.Printf("GetSubcribe: %s ,%s\n", strings.Join(symbols, ","), strings.Join(types, ","))

		// Wait for interrupt signal to gracefully close
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		fmt.Println("Waiting for WebSocket messages... Press Ctrl+C to exit")
		<-sigCh
		fmt.Println("\nShutting down...")

	}
}
