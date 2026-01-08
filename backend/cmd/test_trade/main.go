package main

import (
	"bufio"
	"log"
	"os"
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

	// Try to find valid product code
	originalAcc := cfg.KisAccountNum
	if len(originalAcc) == 8 {
		log.Println("Account number is 8 digits, trying to find valid product code...")
		codes := []string{"01", "22", "11", "02", "10", "21"}

		found := false
		for _, code := range codes {
			// Update config with appended product code
			// e.g. 64796122 -> 6479612201
			client.Config.KisAccountNum = originalAcc + code
			log.Printf("Trying Product Code: %s (Full: %s)", code, client.Config.KisAccountNum)

			// Try GetBalance
			bal, err := client.GetBalance()
			if err == nil && bal != nil {
				log.Printf("SUCCESS! Found valid product code: %s", code)
				log.Printf("Balance: %s", bal.Output2.TotalAmt)
				found = true
				break
			} else {
				log.Printf("Failed with code %s: %v", code, err)
			}

			// Sleep a bit to avoid rate limits?
		}

		if !found {
			log.Println("Could not find valid product code in common list. Reverting to 01 default logic for execution attempt.")
			client.Config.KisAccountNum = originalAcc // Revert
		}
	}

	// 4. Strategy
	strat := service.NewStrategy(db, client)

	// 5. Execute
	log.Println("Triggering ExecuteDaily()...")
	strat.ExecuteDaily()

	log.Println("=== TEST FINISHED ===")
}
