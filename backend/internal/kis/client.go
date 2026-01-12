package kis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
)

type Client struct {
	Config      *config.Config
	AccessToken string
	TokenExp    time.Time
	mu          sync.Mutex
	Client      *http.Client
}

func NewClient(cfg *config.Config) *Client {
	log.Printf("[KIS] Initializing KIS API Client (BaseURL: %s)", cfg.KisBaseURL)
	return &Client{
		Config: cfg,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// logKIS logs a message with KIS prefix and timestamp
func logKIS(format string, v ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[%s][KIS] "+format, append([]interface{}{timestamp}, v...)...)
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

const TokenFile = "kis_token.json"

func (c *Client) EnsureToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Check memory
	if c.AccessToken != "" && time.Now().Before(c.TokenExp.Add(-10*time.Minute)) {
		logKIS("Token still valid (memory) (expires at %s, %v remaining)",
			c.TokenExp.Format("15:04:05"),
			time.Until(c.TokenExp).Round(time.Second))
		return nil
	}

	// 2. Check file
	if c.AccessToken == "" {
		if err := c.loadTokenFromFile(); err == nil {
			if time.Now().Before(c.TokenExp.Add(-10 * time.Minute)) {
				logKIS("Token load from file valid (expires at %s, %v remaining)",
					c.TokenExp.Format("15:04:05"),
					time.Until(c.TokenExp).Round(time.Second))
				return nil
			}
		}
	}

	logKIS("Token expired or not set, requesting new token...")

	url := fmt.Sprintf("%s/oauth2/tokenP", c.Config.KisBaseURL)
	body := map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.Config.KisAppKey,
		"appsecret":  c.Config.KisAppSecret,
	}
	jsonBody, _ := json.Marshal(body)

	logKIS("POST %s (requesting OAuth token)", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logKIS("✗ Failed to create token request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		logKIS("✗ Token request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logKIS("✗ Auth failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("auth failed with status: %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		logKIS("✗ Failed to decode token response: %v", err)
		return err
	}

	c.AccessToken = authResp.AccessToken
	// Usually expires in 86400 seconds (24 hours)
	c.TokenExp = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	// Save to file
	c.saveTokenToFile()

	logKIS("✓ Token refreshed successfully")
	logKIS("  Token expires at: %s (%v from now)",
		c.TokenExp.Format("2006-01-02 15:04:05"),
		time.Until(c.TokenExp).Round(time.Second))

	return nil
}

func (c *Client) loadTokenFromFile() error {
	file, err := os.Open(TokenFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var data struct {
		AccessToken string    `json:"access_token"`
		TokenExp    time.Time `json:"token_exp"`
	}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	c.AccessToken = data.AccessToken
	c.TokenExp = data.TokenExp
	return nil
}

func (c *Client) saveTokenToFile() {
	data := struct {
		AccessToken string    `json:"access_token"`
		TokenExp    time.Time `json:"token_exp"`
	}{
		AccessToken: c.AccessToken,
		TokenExp:    c.TokenExp,
	}

	file, err := os.Create(TokenFile)
	if err != nil {
		logKIS("Warning: Failed to save token to file: %v", err)
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(data)
}

// Price Response
type PriceResponse struct {
	Output struct {
		Last string `json:"last"` // Last price
		Base string `json:"base"` // Previous close
	} `json:"output"`
	RtCd string `json:"rt_cd"`
	Msg1 string `json:"msg1"`
}

// getAccountParts splits account number into CANO (8 digits) and ACNT_PRDT_CD (2 digits)
func (c *Client) getAccountParts() (string, string) {
	acc := strings.TrimSpace(c.Config.KisAccountNum)
	if len(acc) == 8 {
		return acc, "01" // Default to 01 if only CANO is provided
	}
	if len(acc) >= 10 {
		return acc[:8], acc[8:10]
	}
	// Fallback/Error case, though Config should validate
	return acc, "01"
}

func (c *Client) GetCurrentPrice(exchCode, symbol string) (float64, error) {
	logKIS("GetCurrentPrice: Fetching price for %s:%s", exchCode, symbol)

	if err := c.EnsureToken(); err != nil {
		logKIS("✗ GetCurrentPrice: Token error: %v", err)
		return 0, err
	}

	url := fmt.Sprintf("%s/uapi/overseas-price/v1/quotations/price?AUTH=&EXCD=%s&SYMB=%s", c.Config.KisBaseURL, exchCode, symbol)
	logKIS("GET %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logKIS("✗ GetCurrentPrice: Failed to create request: %v", err)
		return 0, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", "HHDFS76200200") // Overseas Stock Price

	resp, err := c.Client.Do(req)
	if err != nil {
		logKIS("✗ GetCurrentPrice: Request failed: %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logKIS("✗ GetCurrentPrice: Bad status %d: %s", resp.StatusCode, string(bodyBytes))
		return 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// Read response for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	logKIS("GetCurrentPrice: Raw response: %s", string(bodyBytes))

	var pResp PriceResponse
	if err := json.Unmarshal(bodyBytes, &pResp); err != nil {
		logKIS("✗ GetCurrentPrice: Failed to decode response: %v", err)
		return 0, err
	}

	// Success codes: "0" or "0000"
	if pResp.RtCd != "0" && pResp.RtCd != "0000" {
		logKIS("✗ GetCurrentPrice: API error (RtCd=%s): %s", pResp.RtCd, pResp.Msg1)
		return 0, fmt.Errorf("api error: %s", pResp.Msg1)
	}

	// Parse price (string to float)
	var price float64
	fmt.Sscanf(pResp.Output.Last, "%f", &price)
	logKIS("✓ GetCurrentPrice: %s:%s = $%.2f (base: %s)", exchCode, symbol, price, pResp.Output.Base)
	return price, nil
}

// Order Request
type OrderReq struct {
	ExchCode string // NAS, NYS, AMS
	Symbol   string
	Qty      int
	Price    float64
	OrdType  string // 00: Limit, 01: Market, 34: LOC ? (Need verification)
	Side     string // BUY or SELL
}

// OrderResponse represents the response from order API
type OrderResponse struct {
	RtCd   string `json:"rt_cd"`
	Msg1   string `json:"msg1"`
	MsgCd  string `json:"msg_cd"`
	Output struct {
		KRX_FWDG_ORD_ORGNO string `json:"KRX_FWDG_ORD_ORGNO"`
		ODNO               string `json:"ODNO"`
		ORD_TMD            string `json:"ORD_TMD"`
	} `json:"output"`
}

func (c *Client) PlaceOrder(o OrderReq) error {
	logKIS("PlaceOrder: %s %d shares of %s:%s at $%.2f (type: %s)",
		o.Side, o.Qty, o.ExchCode, o.Symbol, o.Price, o.OrdType)

	if err := c.EnsureToken(); err != nil {
		logKIS("✗ PlaceOrder: Token error: %v", err)
		return err
	}

	trID := "TTTT1002U" // Buy (Real)
	if o.Side == "SELL" {
		trID = "TTTT1006U" // Sell (Real)
	}
	logKIS("PlaceOrder: Using TR_ID=%s for %s order", trID, o.Side)
	// TODO: Handle Virtual Trading TR IDs if needed (JTTT1002U, JTTT1006U)

	url := fmt.Sprintf("%s/uapi/overseas-stock/v1/trading/order", c.Config.KisBaseURL)
	logKIS("POST %s", url)

	cano, prdt := c.getAccountParts()

	body := map[string]string{
		"CANO":            cano,
		"ACNT_PRDT_CD":    prdt,
		"OVRS_EXCG_CD":    o.ExchCode,
		"PDNO":            o.Symbol,
		"ORD_QTY":         fmt.Sprintf("%d", o.Qty),
		"OVRS_ORD_UNPR":   fmt.Sprintf("%.2f", o.Price),
		"ORD_SVR_DVSN_CD": "0",
		"ORD_DVSN":        o.OrdType,
	}
	jsonBody, _ := json.Marshal(body)
	logKIS("PlaceOrder: Request body: %s", string(jsonBody))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logKIS("✗ PlaceOrder: Failed to create request: %v", err)
		return err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", trID)

	resp, err := c.Client.Do(req)
	if err != nil {
		logKIS("✗ PlaceOrder: Request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Read response body for logging
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		logKIS("✗ PlaceOrder: Failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("order failed status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response for detailed logging
	var orderResp OrderResponse
	if err := json.Unmarshal(bodyBytes, &orderResp); err == nil {
		if orderResp.RtCd == "0" {
			logKIS("✓ PlaceOrder: SUCCESS - Order ID: %s, Time: %s, Msg: %s",
				orderResp.Output.ODNO, orderResp.Output.ORD_TMD, orderResp.Msg1)
		} else {
			logKIS("⚠ PlaceOrder: Response Code: %s, Msg: %s", orderResp.RtCd, orderResp.Msg1)
		}
	} else {
		logKIS("✓ PlaceOrder: Completed (raw response: %s)", string(bodyBytes))
	}

	return nil
}

// Balance Response
type BalanceResponse struct {
	Output1 []struct {
		ExchCode string `json:"ovrs_excg_cd"`
		Symbol   string `json:"ovrs_pdno"`
		Qty      string `json:"ovrs_cblc_qty"` // Holding Qty
		AvgPrice string `json:"pchs_avg_pric"` // Avg Purchase Price
		NowPrice string `json:"now_pric2"`     // Current Price
	} `json:"output1"`
	Output2 struct {
		TotalAmt      string `json:"tot_evlu_pfls_amt"`  // Total Evaluation Amount
		TotalPurchase string `json:"frcr_pchs_amt1"`     // Total Purchase Amount (invested)
		TotalPL       string `json:"ovrs_tot_pfls"`      // Total Profit/Loss
		TotalPLRate   string `json:"tot_pftrt"`          // Total P/L Rate (%)
		RealizedPL    string `json:"ovrs_rlzt_pfls_amt"` // Realized P/L
	} `json:"output2"`
	RtCd string `json:"rt_cd"`
	Msg1 string `json:"msg1"`
}

func (c *Client) GetBalance() (*BalanceResponse, error) {
	logKIS("GetBalance: Fetching portfolio balance...")

	if err := c.EnsureToken(); err != nil {
		logKIS("✗ GetBalance: Token error: %v", err)
		return nil, err
	}

	cano, prdt := c.getAccountParts()

	// Note: CTX_AREA keys are for pagination, empty for first page
	// API requires FK200/NK200, not FK100/NK100
	url := fmt.Sprintf("%s/uapi/overseas-stock/v1/trading/inquire-balance?AUTH=&CANO=%s&ACNT_PRDT_CD=%s&OVRS_EXCG_CD=NASD&TR_CRCY_CD=USD&CTX_AREA_FK200=&CTX_AREA_NK200=", c.Config.KisBaseURL, cano, prdt)
	logKIS("GET %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logKIS("✗ GetBalance: Failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", "TTTS3012R") // Real: TTTS3012R

	resp, err := c.Client.Do(req)
	if err != nil {
		logKIS("✗ GetBalance: Request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logKIS("✗ GetBalance: Bad status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("balance failed status: %d", resp.StatusCode)
	}

	// Read body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	logKIS("GetBalance: Raw response: %s", string(bodyBytes))

	var bResp BalanceResponse
	if err := json.Unmarshal(bodyBytes, &bResp); err != nil {
		logKIS("✗ GetBalance: Failed to decode response: %v", err)
		return nil, err
	}

	// Success codes: "0" or "0000"
	if bResp.RtCd != "0" && bResp.RtCd != "0000" {
		logKIS("✗ GetBalance: API error (RtCd=%s): %s", bResp.RtCd, bResp.Msg1)
		return nil, fmt.Errorf("api error: %s", bResp.Msg1)
	}

	logKIS("✓ GetBalance: Found %d holdings, Total P/L: %s", len(bResp.Output1), bResp.Output2.TotalAmt)
	for i, h := range bResp.Output1 {
		logKIS("  [%d] %s:%s - Qty: %s, AvgPrice: %s", i+1, h.ExchCode, h.Symbol, h.Qty, h.AvgPrice)
	}

	return &bResp, nil
}

// PresentBalanceResponse for overseas stock present balance (cash inquiry)
type PresentBalanceResponse struct {
	Output struct {
		FrcrDncaTotAmt string `json:"frcr_dnca_tot_amt"`  // Foreign currency deposit total (USD)
		FrcrBuyAmt     string `json:"frcr_buy_amt_smtl"`  // Foreign currency buy amount
		FrcrSellAmt    string `json:"frcr_sell_amt_smtl"` // Foreign currency sell amount
		OvrsOrdPsblAmt string `json:"ovrs_ord_psbl_amt"`  // Overseas order available amount
	} `json:"output2"`
	RtCd string `json:"rt_cd"`
	Msg1 string `json:"msg1"`
}

// GetPresentBalance fetches available cash balance for overseas stock trading
func (c *Client) GetPresentBalance() (*PresentBalanceResponse, error) {
	logKIS("GetPresentBalance: Fetching available cash...")

	if err := c.EnsureToken(); err != nil {
		logKIS("✗ GetPresentBalance: Token error: %v", err)
		return nil, err
	}

	cano, prdt := c.getAccountParts()

	// Use present balance API
	url := fmt.Sprintf("%s/uapi/overseas-stock/v1/trading/inquire-present-balance?CANO=%s&ACNT_PRDT_CD=%s&WCRC_FRCR_DVSN_CD=02&NATN_CD=840&TR_MKET_CD=00&INQR_DVSN_CD=00",
		c.Config.KisBaseURL, cano, prdt)
	logKIS("GET %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", "CTRP6504R") // Present balance inquiry

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	logKIS("GetPresentBalance: Raw response: %s", string(bodyBytes))

	var pResp PresentBalanceResponse
	if err := json.Unmarshal(bodyBytes, &pResp); err != nil {
		return nil, err
	}

	if pResp.RtCd != "0" && pResp.RtCd != "0000" {
		logKIS("✗ GetPresentBalance: API error (RtCd=%s): %s", pResp.RtCd, pResp.Msg1)
		return nil, fmt.Errorf("api error: %s", pResp.Msg1)
	}

	logKIS("✓ GetPresentBalance: Available Cash: $%s", pResp.Output.OvrsOrdPsblAmt)
	return &pResp, nil
}
