package kis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	return &Client{
		Config: cfg,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *Client) EnsureToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.AccessToken != "" && time.Now().Before(c.TokenExp.Add(-10*time.Minute)) {
		return nil
	}

	url := fmt.Sprintf("%s/oauth2/tokenP", c.Config.KisBaseURL)
	body := map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.Config.KisAppKey,
		"appsecret":  c.Config.KisAppSecret,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("auth failed with status: %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	c.AccessToken = authResp.AccessToken
	// Usually expires in 86400 seconds (24 hours)
	c.TokenExp = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	log.Printf("KIS API Token refreshed, expires at %s", c.TokenExp)

	return nil
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

func (c *Client) GetCurrentPrice(exchCode, symbol string) (float64, error) {
	if err := c.EnsureToken(); err != nil {
		return 0, err
	}

	url := fmt.Sprintf("%s/uapi/overseas-price/v1/quotations/price?AUTH=&EXCD=%s&SYMB=%s", c.Config.KisBaseURL, exchCode, symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", "HHDFS76200200") // Overseas Stock Price

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var pResp PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&pResp); err != nil {
		return 0, err
	}

	if pResp.RtCd != "0000" {
		return 0, fmt.Errorf("api error: %s", pResp.Msg1)
	}

	// Parse price (string to float)
	var price float64
	fmt.Sscanf(pResp.Output.Last, "%f", &price)
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

func (c *Client) PlaceOrder(o OrderReq) error {
	if err := c.EnsureToken(); err != nil {
		return err
	}

	trID := "TTTT1002U" // Buy (Real)
	if o.Side == "SELL" {
		trID = "TTTT1006U" // Sell (Real)
	}
	// TODO: Handle Virtual Trading TR IDs if needed (JTTT1002U, JTTT1006U)

	url := fmt.Sprintf("%s/uapi/overseas-stock/v1/trading/order", c.Config.KisBaseURL)

	body := map[string]string{
		"CANO":            c.Config.KisAccountNum[:8],
		"ACNT_PRDT_CD":    c.Config.KisAccountNum[8:],
		"OVRS_EXCG_CD":    o.ExchCode,
		"PDNO":            o.Symbol,
		"ORD_QTY":         fmt.Sprintf("%d", o.Qty),
		"OVRS_ORD_UNPR":   fmt.Sprintf("%.2f", o.Price),
		"ORD_SVR_DVSN_CD": "0",
		"ORD_DVSN":        o.OrdType,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", trID)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Read body for error
		return fmt.Errorf("order failed status: %d", resp.StatusCode)
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
	} `json:"output1"`
	Output2 struct {
		TotalAmt string `json:"tot_evlu_pfls_amt"` // Total Profit/Loss
	} `json:"output2"`
	RtCd string `json:"rt_cd"`
	Msg1 string `json:"msg1"`
}

func (c *Client) GetBalance() (*BalanceResponse, error) {
	if err := c.EnsureToken(); err != nil {
		return nil, err
	}

	// Note: CTX_AREA keys might be needed for pagination, empty for first page
	url := fmt.Sprintf("%s/uapi/overseas-stock/v1/trading/inquire-balance?AUTH=&CANO=%s&ACNT_PRDT_CD=%s&OVRS_EXCG_CD=NAS&TR_CRCY_CD=USD&CTX_AREA_FK100=&CTX_AREA_NK100=", c.Config.KisBaseURL, c.Config.KisAccountNum[:8], c.Config.KisAccountNum[8:])
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("appkey", c.Config.KisAppKey)
	req.Header.Set("appsecret", c.Config.KisAppSecret)
	req.Header.Set("tr_id", "TTTS3012R") // Real: TTTS3012R

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("balance failed status: %d", resp.StatusCode)
	}

	var bResp BalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&bResp); err != nil {
		return nil, err
	}

	if bResp.RtCd != "0000" {
		return nil, fmt.Errorf("api error: %s", bResp.Msg1)
	}

	return &bResp, nil
}
