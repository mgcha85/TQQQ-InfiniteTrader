package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/config"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/kis"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/repository"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
)

func loadEnv() {
	// Try multiple paths
	paths := []string{"../.env", "../../.env", ".env"}
	var file *os.File
	var err error

	for _, p := range paths {
		file, err = os.Open(p)
		if err == nil {
			log.Printf("Loaded env from %s", p)
			break
		}
	}

	if file == nil {
		log.Println("Note: Could not open .env file (tried ../.env, ../../.env, .env), relying on system env vars")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, `"'`)
			// Remove comments at end of line if any (simple case)
			if idx := strings.Index(value, " #"); idx != -1 {
				value = strings.TrimSpace(value[:idx])
			}
			os.Setenv(key, value)
		}
	}
}

func main() {
	log.Println("=== STARTING TRADE TEST ===")

	// 0. Load Env
	loadEnv()

	// 1. Config
	cfg := config.Load()

	maskKey := "EMPTY"
	if len(cfg.KisAppKey) > 5 {
		maskKey = cfg.KisAppKey[:5] + "..."
	}
	log.Printf("Config loaded. Account: %s, AppKey: %s", cfg.KisAccountNum, maskKey)

	// 2. DB
	db, err := repository.NewDB("data/db.sqlite")
	if err != nil {
		log.Fatal("DB init failed:", err)
	}
	log.Println("DB connected.")

	// Seed Settings if not exists
	var settings struct{ ID uint }
	if err := db.Table("user_settings").First(&settings).Error; err != nil {
		log.Println("Seeding default settings...")
		db.Exec(`INSERT INTO user_settings (is_active, principal, split_count, target_rate, created_at, updated_at) 
				VALUES (1, 1000000, 40, 0.1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	}

	// 3. KIS Client
	client := kis.NewClient(cfg)

	// 4. Test Buying Power API
	log.Println("Testing GetBuyingPower API...")
	bp, bpErr := client.GetBuyingPower()
	if bpErr != nil {
		log.Printf("✗ GetBuyingPower failed: %v", bpErr)
	} else {
		log.Printf("✓ Available Cash: $%s", bp.Output.OvrsOrdPsblAmt)
	}

	// 5. Test Order Error Handling
	log.Println("Testing PlaceOrder Error Handling (0 qty)...")
	errReq := kis.OrderReq{ExchCode: "NASD", Symbol: "TQQQ", OrdType: "00", Side: "BUY", Qty: 0, Price: 50.0}
	if err := client.PlaceOrder(errReq); err != nil {
		log.Printf("✓ Correctly caught error: %v", err)
	} else {
		log.Printf("✗ Failed to catch error (returned nil)")
	}

	// 6. Update Settings with Real Principal
	if bpErr == nil {
		totalCash, _ := strconv.ParseFloat(bp.Output.OvrsOrdPsblAmt, 64)
		// Assume we want to use available cash as Principal for this test
		// Principal = Cash + Invested (but for now let's just use Cash as if starting fresh or keep existing logic)
		// Better: Set Principal to a value that makes 1/40 split possible.
		// If cash is $15,000, 1/40 is $375.

		log.Printf("Updating Settings: Principal=$%.2f, SplitCount=40 (1/40 = $%.2f)", totalCash, totalCash/40.0)

		// Update DB
		db.Exec("UPDATE user_settings SET principal = ?, split_count = 40, is_active = 1 WHERE id = 1", totalCash)
	}

	// 7. Strategy
	strat := service.NewStrategy(db, client)

	// 8. Execute
	log.Println("Triggering ExecuteDaily()...")
	strat.ExecuteDaily()

	log.Println("=== TEST FINISHED ===")

	log.Println("=== TEST FINISHED ===")
}
