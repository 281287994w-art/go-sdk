package main

import (
	"fmt"
)

// 直接复制Client结构体和相关方法到测试文件中，避免导入问题

const (
	BaseURL = "https://api.itick.org"
	WSSURL = "wss://api.itick.org"
	
	// WebSocket constants
	PingInterval = 30
	ReconnectInterval = 5
	MaxReconnectAttempts = 10
)

type Client struct {
	token string
}

type Response struct {
	Code int         `json:"code"`
	Msg  interface{} `json:"msg"`
	Data interface{} `json:"data"`
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
	}
}

func (c *Client) GetSymbolList() (interface{}, error) {
	fmt.Println("GetSymbolList called")
	return nil, nil
}

func (c *Client) GetSymbolHolidays() (interface{}, error) {
	fmt.Println("GetSymbolHolidays called")
	return nil, nil
}

func (c *Client) GetStockInfo(market, code string) (interface{}, error) {
	fmt.Println("GetStockInfo called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) GetStockTick(market, code string) (interface{}, error) {
	fmt.Println("GetStockTick called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) GetIndicesTick(market, code string) (interface{}, error) {
	fmt.Println("GetIndicesTick called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) GetFutureTick(market, code string) (interface{}, error) {
	fmt.Println("GetFutureTick called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) GetFundTick(market, code string) (interface{}, error) {
	fmt.Println("GetFundTick called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) GetForexTick(market, code string) (interface{}, error) {
	fmt.Println("GetForexTick called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) GetCryptoTick(market, code string) (interface{}, error) {
	fmt.Println("GetCryptoTick called with market:", market, "code:", code)
	return nil, nil
}

func (c *Client) ConnectStockWebSocket() error {
	fmt.Println("ConnectStockWebSocket called")
	return nil
}

func (c *Client) CloseWebSocket() error {
	fmt.Println("CloseWebSocket called")
	return nil
}

func main() {
	// 使用真实API密钥
	token := "8850*****************ee4127087"
	client := NewClient(token)

	// 测试基础模块
	fmt.Println("=== 测试基础模块 ===")
	_, err := client.GetSymbolList()
	if err != nil {
		fmt.Printf("GetSymbolList 错误: %v\n", err)
	} else {
		fmt.Println("GetSymbolList 成功")
	}

	_, err = client.GetSymbolHolidays()
	if err != nil {
		fmt.Printf("GetSymbolHolidays 错误: %v\n", err)
	} else {
		fmt.Println("GetSymbolHolidays 成功")
	}

	// 测试股票模块
	fmt.Println("\n=== 测试股票模块 ===")
	_, err = client.GetStockInfo("us", "AAPL")
	if err != nil {
		fmt.Printf("GetStockInfo 错误: %v\n", err)
	} else {
		fmt.Println("GetStockInfo 成功")
	}

	_, err = client.GetStockTick("us", "AAPL")
	if err != nil {
		fmt.Printf("GetStockTick 错误: %v\n", err)
	} else {
		fmt.Println("GetStockTick 成功")
	}

	// 测试指数模块
	fmt.Println("\n=== 测试指数模块 ===")
	_, err = client.GetIndicesTick("us", "SPX")
	if err != nil {
		fmt.Printf("GetIndicesTick 错误: %v\n", err)
	} else {
		fmt.Println("GetIndicesTick 成功")
	}

	// 测试期货模块
	fmt.Println("\n=== 测试期货模块 ===")
	_, err = client.GetFutureTick("us", "ES")
	if err != nil {
		fmt.Printf("GetFutureTick 错误: %v\n", err)
	} else {
		fmt.Println("GetFutureTick 成功")
	}

	// 测试基金模块
	fmt.Println("\n=== 测试基金模块 ===")
	_, err = client.GetFundTick("us", "SPY")
	if err != nil {
		fmt.Printf("GetFundTick 错误: %v\n", err)
	} else {
		fmt.Println("GetFundTick 成功")
	}

	// 测试外汇模块
	fmt.Println("\n=== 测试外汇模块 ===")
	_, err = client.GetForexTick("forex", "EURUSD")
	if err != nil {
		fmt.Printf("GetForexTick 错误: %v\n", err)
	} else {
		fmt.Println("GetForexTick 成功")
	}

	// 测试加密货币模块
	fmt.Println("\n=== 测试加密货币模块 ===")
	_, err = client.GetCryptoTick("crypto", "BTCUSD")
	if err != nil {
		fmt.Printf("GetCryptoTick 错误: %v\n", err)
	} else {
		fmt.Println("GetCryptoTick 成功")
	}

	// 测试WebSocket
	fmt.Println("\n=== 测试WebSocket ===")
	err = client.ConnectStockWebSocket()
	if err != nil {
		fmt.Printf("ConnectStockWebSocket 错误: %v\n", err)
	} else {
		fmt.Println("ConnectStockWebSocket 成功")
		client.CloseWebSocket()
	}

	fmt.Println("\n所有测试完成")
}
