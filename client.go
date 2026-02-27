package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	BaseURL = "https://api.itick.org"
	WSSURL = "wss://api.itick.org"
	
	// WebSocket constants
	PingInterval = 30 * time.Second
	ReconnectInterval = 5 * time.Second
	MaxReconnectAttempts = 10
)

type Client struct {
	token string
	client *http.Client
	wsClient *websocket.Conn
	wsPath string
	wsConnected bool
	wsMutex sync.Mutex
	reconnectChan chan struct{}
	closeChan chan struct{}
	messageHandler func([]byte)
	errorHandler func(error)
}

type Response struct {
	Code int         `json:"code"`
	Msg  interface{} `json:"msg"`
	Data interface{} `json:"data"`
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		reconnectChan: make(chan struct{}),
		closeChan: make(chan struct{}),
	}
}

func (c *Client) get(path string, params map[string]string, result interface{}) error {
	url := BaseURL + path
	if len(params) > 0 {
		url += "?"
		for k, v := range params {
			url += fmt.Sprintf("%s=%s&", k, v)
		}
		url = url[:len(url)-1]
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("token", c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if response.Code != 0 {
		return fmt.Errorf("API error: %v", response.Msg)
	}

	data, err := json.Marshal(response.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}

// WebSocket methods with enhanced functionality

// SetMessageHandler sets the callback for received WebSocket messages
func (c *Client) SetMessageHandler(handler func([]byte)) {
	c.messageHandler = handler
}

// SetErrorHandler sets the callback for WebSocket errors
func (c *Client) SetErrorHandler(handler func(error)) {
	c.errorHandler = handler
}

// ConnectWebSocket establishes a WebSocket connection with automatic reconnection
func (c *Client) ConnectWebSocket(path string) error {
	c.wsPath = path
	return c.connectWebSocket()
}

func (c *Client) connectWebSocket() error {
	c.wsMutex.Lock()
	defer c.wsMutex.Unlock()

	url := WSSURL + c.wsPath
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{
		"token": []string{c.token},
	})
	if err != nil {
		if c.errorHandler != nil {
			c.errorHandler(err)
		}
		return err
	}

	c.wsClient = conn
	c.wsConnected = true

	// Start ping goroutine
	go c.pingLoop()
	// Start read goroutine
	go c.readLoop()
	// Start reconnect goroutine
	go c.reconnectLoop()

	return nil
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.wsMutex.Lock()
			if c.wsClient != nil && c.wsConnected {
				err := c.wsClient.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					c.wsConnected = false
					c.reconnectChan <- struct{}{}
				}
			}
			c.wsMutex.Unlock()
		case <-c.closeChan:
			return
		}
	}
}

func (c *Client) readLoop() {
	for {
		c.wsMutex.Lock()
		conn := c.wsClient
		connected := c.wsConnected
		c.wsMutex.Unlock()

		if !connected || conn == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.wsMutex.Lock()
			c.wsConnected = false
			c.wsMutex.Unlock()

			if c.errorHandler != nil {
				c.errorHandler(err)
			}
			c.reconnectChan <- struct{}{}
			continue
		}

		if c.messageHandler != nil {
			c.messageHandler(message)
		}
	}
}

func (c *Client) reconnectLoop() {
	reconnectAttempts := 0

	for {
		select {
		case <-c.reconnectChan:
			if reconnectAttempts >= MaxReconnectAttempts {
				if c.errorHandler != nil {
					c.errorHandler(fmt.Errorf("max reconnect attempts reached"))
				}
				continue
			}

			time.Sleep(ReconnectInterval)
			err := c.connectWebSocket()
			if err != nil {
				reconnectAttempts++
			} else {
				reconnectAttempts = 0
			}
		case <-c.closeChan:
			return
		}
	}
}

func (c *Client) SendWebSocketMessage(message []byte) error {
	c.wsMutex.Lock()
	defer c.wsMutex.Unlock()

	if !c.wsConnected || c.wsClient == nil {
		return fmt.Errorf("websocket not connected")
	}

	return c.wsClient.WriteMessage(websocket.TextMessage, message)
}

func (c *Client) CloseWebSocket() error {
	c.wsMutex.Lock()
	defer c.wsMutex.Unlock()

	close(c.closeChan)

	if c.wsClient == nil {
		return nil
	}

	err := c.wsClient.Close()
	c.wsConnected = false
	c.wsClient = nil

	return err
}

// IsWebSocketConnected returns the current WebSocket connection status
func (c *Client) IsWebSocketConnected() bool {
	c.wsMutex.Lock()
	defer c.wsMutex.Unlock()
	return c.wsConnected
}

// 基础模块

// GetSymbolList 获取符号列表
func (c *Client) GetSymbolList() (interface{}, error) {
	var result interface{}
	err := c.get("/symbol/list", map[string]string{}, &result)
	return result, err
}

// GetSymbolHolidays 获取节假日信息
func (c *Client) GetSymbolHolidays() (interface{}, error) {
	var result interface{}
	err := c.get("/symbol/holidays", map[string]string{}, &result)
	return result, err
}

// 股票模块

// GetStockInfo 获取股票信息
func (c *Client) GetStockInfo(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/stock/info", params, &result)
	return result, err
}

// GetStockIPO 获取股票IPO信息
func (c *Client) GetStockIPO(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/stock/ipo", params, &result)
	return result, err
}

// GetStockSplit 获取股票分拆信息
func (c *Client) GetStockSplit(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/stock/split", params, &result)
	return result, err
}

// GetStockTick 获取股票实时成交
func (c *Client) GetStockTick(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/stock/tick", params, &result)
	return result, err
}

// GetStockQuote 获取股票实时报价
func (c *Client) GetStockQuote(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/stock/quote", params, &result)
	return result, err
}

// GetStockDepth 获取股票实时盘口
func (c *Client) GetStockDepth(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/stock/depth", params, &result)
	return result, err
}

// GetStockKline 获取股票历史K线
func (c *Client) GetStockKline(market, code string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/stock/kline", params, &result)
	return result, err
}

// GetStockTicks 获取股票批量实时成交
func (c *Client) GetStockTicks(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/stock/ticks", params, &result)
	return result, err
}

// GetStockQuotes 获取股票批量实时报价
func (c *Client) GetStockQuotes(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/stock/quotes", params, &result)
	return result, err
}

// GetStockDepths 获取股票批量实时盘口
func (c *Client) GetStockDepths(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/stock/depths", params, &result)
	return result, err
}

// GetStockKlines 获取股票批量历史K线
func (c *Client) GetStockKlines(market string, codes []string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/stock/klines", params, &result)
	return result, err
}

// ConnectStockWebSocket 连接股票WebSocket
func (c *Client) ConnectStockWebSocket() error {
	return c.ConnectWebSocket("/websocket/stock")
}

// 指数模块

// GetIndicesTick 获取指数实时成交
func (c *Client) GetIndicesTick(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/indices/tick", params, &result)
	return result, err
}

// GetIndicesQuote 获取指数实时报价
func (c *Client) GetIndicesQuote(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/indices/quote", params, &result)
	return result, err
}

// GetIndicesDepth 获取指数实时盘口
func (c *Client) GetIndicesDepth(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/indices/depth", params, &result)
	return result, err
}

// GetIndicesKline 获取指数历史K线
func (c *Client) GetIndicesKline(market, code string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/indices/kline", params, &result)
	return result, err
}

// GetIndicesTicks 获取指数批量实时成交
func (c *Client) GetIndicesTicks(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/indices/ticks", params, &result)
	return result, err
}

// GetIndicesQuotes 获取指数批量实时报价
func (c *Client) GetIndicesQuotes(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/indices/quotes", params, &result)
	return result, err
}

// GetIndicesDepths 获取指数批量实时盘口
func (c *Client) GetIndicesDepths(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/indices/depths", params, &result)
	return result, err
}

// GetIndicesKlines 获取指数批量历史K线
func (c *Client) GetIndicesKlines(market string, codes []string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/indices/klines", params, &result)
	return result, err
}

// ConnectIndicesWebSocket 连接指数WebSocket
func (c *Client) ConnectIndicesWebSocket() error {
	return c.ConnectWebSocket("/websocket/indices")
}

// 期货模块

// GetFutureTick 获取期货实时成交
func (c *Client) GetFutureTick(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/future/tick", params, &result)
	return result, err
}

// GetFutureQuote 获取期货实时报价
func (c *Client) GetFutureQuote(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/future/quote", params, &result)
	return result, err
}

// GetFutureDepth 获取期货实时盘口
func (c *Client) GetFutureDepth(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/future/depth", params, &result)
	return result, err
}

// GetFutureKline 获取期货历史K线
func (c *Client) GetFutureKline(market, code string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/future/kline", params, &result)
	return result, err
}

// GetFutureTicks 获取期货批量实时成交
func (c *Client) GetFutureTicks(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/future/ticks", params, &result)
	return result, err
}

// GetFutureQuotes 获取期货批量实时报价
func (c *Client) GetFutureQuotes(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/future/quotes", params, &result)
	return result, err
}

// GetFutureDepths 获取期货批量实时盘口
func (c *Client) GetFutureDepths(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/future/depths", params, &result)
	return result, err
}

// GetFutureKlines 获取期货批量历史K线
func (c *Client) GetFutureKlines(market string, codes []string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/future/klines", params, &result)
	return result, err
}

// ConnectFutureWebSocket 连接期货WebSocket
func (c *Client) ConnectFutureWebSocket() error {
	return c.ConnectWebSocket("/websocket/future")
}

// 基金模块

// GetFundTick 获取基金实时成交
func (c *Client) GetFundTick(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/fund/tick", params, &result)
	return result, err
}

// GetFundQuote 获取基金实时报价
func (c *Client) GetFundQuote(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/fund/quote", params, &result)
	return result, err
}

// GetFundDepth 获取基金实时盘口
func (c *Client) GetFundDepth(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/fund/depth", params, &result)
	return result, err
}

// GetFundKline 获取基金历史K线
func (c *Client) GetFundKline(market, code string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/fund/kline", params, &result)
	return result, err
}

// GetFundTicks 获取基金批量实时成交
func (c *Client) GetFundTicks(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/fund/ticks", params, &result)
	return result, err
}

// GetFundQuotes 获取基金批量实时报价
func (c *Client) GetFundQuotes(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/fund/quotes", params, &result)
	return result, err
}

// GetFundDepths 获取基金批量实时盘口
func (c *Client) GetFundDepths(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/fund/depths", params, &result)
	return result, err
}

// GetFundKlines 获取基金批量历史K线
func (c *Client) GetFundKlines(market string, codes []string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/fund/klines", params, &result)
	return result, err
}

// ConnectFundWebSocket 连接基金WebSocket
func (c *Client) ConnectFundWebSocket() error {
	return c.ConnectWebSocket("/websocket/fund")
}

// 外汇模块

// GetForexTick 获取外汇实时成交
func (c *Client) GetForexTick(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/forex/tick", params, &result)
	return result, err
}

// GetForexQuote 获取外汇实时报价
func (c *Client) GetForexQuote(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/forex/quote", params, &result)
	return result, err
}

// GetForexDepth 获取外汇实时盘口
func (c *Client) GetForexDepth(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/forex/depth", params, &result)
	return result, err
}

// GetForexKline 获取外汇历史K线
func (c *Client) GetForexKline(market, code string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/forex/kline", params, &result)
	return result, err
}

// GetForexTicks 获取外汇批量实时成交
func (c *Client) GetForexTicks(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/forex/ticks", params, &result)
	return result, err
}

// GetForexQuotes 获取外汇批量实时报价
func (c *Client) GetForexQuotes(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/forex/quotes", params, &result)
	return result, err
}

// GetForexDepths 获取外汇批量实时盘口
func (c *Client) GetForexDepths(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/forex/depths", params, &result)
	return result, err
}

// GetForexKlines 获取外汇批量历史K线
func (c *Client) GetForexKlines(market string, codes []string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/forex/klines", params, &result)
	return result, err
}

// ConnectForexWebSocket 连接外汇WebSocket
func (c *Client) ConnectForexWebSocket() error {
	return c.ConnectWebSocket("/websocket/forex")
}

// 加密货币模块

// GetCryptoTick 获取加密货币实时成交
func (c *Client) GetCryptoTick(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/crypto/tick", params, &result)
	return result, err
}

// GetCryptoQuote 获取加密货币实时报价
func (c *Client) GetCryptoQuote(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/crypto/quote", params, &result)
	return result, err
}

// GetCryptoDepth 获取加密货币实时盘口
func (c *Client) GetCryptoDepth(market, code string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
	}
	err := c.get("/crypto/depth", params, &result)
	return result, err
}

// GetCryptoKline 获取加密货币历史K线
func (c *Client) GetCryptoKline(market, code string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"code":   code,
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/crypto/kline", params, &result)
	return result, err
}

// GetCryptoTicks 获取加密货币批量实时成交
func (c *Client) GetCryptoTicks(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/crypto/ticks", params, &result)
	return result, err
}

// GetCryptoQuotes 获取加密货币批量实时报价
func (c *Client) GetCryptoQuotes(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/crypto/quotes", params, &result)
	return result, err
}

// GetCryptoDepths 获取加密货币批量实时盘口
func (c *Client) GetCryptoDepths(market string, codes []string) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
	}
	err := c.get("/crypto/depths", params, &result)
	return result, err
}

// GetCryptoKlines 获取加密货币批量历史K线
func (c *Client) GetCryptoKlines(market string, codes []string, period, limit int, end *int64) (interface{}, error) {
	var result interface{}
	params := map[string]string{
		"market": market,
		"codes":  fmt.Sprintf("%v", codes),
		"period": fmt.Sprintf("%d", period),
		"limit":  fmt.Sprintf("%d", limit),
	}
	if end != nil {
		params["end"] = fmt.Sprintf("%d", *end)
	}
	err := c.get("/crypto/klines", params, &result)
	return result, err
}

// ConnectCryptoWebSocket 连接加密货币WebSocket
func (c *Client) ConnectCryptoWebSocket() error {
	return c.ConnectWebSocket("/websocket/crypto")
}
