package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/market"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/model"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/repository"
	"github.com/mgcha85/TQQQ-InfiniteTrader/backend/internal/service"
)

type Handler struct {
	Repo       *repository.DB
	Strategy   *service.Strategy
	MarketSvc  *market.MarketDataService
	MarketRepo *market.MarketRepository
}

func NewHandler(repo *repository.DB, strat *service.Strategy, mSvc *market.MarketDataService, mRepo *market.MarketRepository) *Handler {
	return &Handler{
		Repo:       repo,
		Strategy:   strat,
		MarketSvc:  mSvc,
		MarketRepo: mRepo,
	}
}

// GetDashboard returns summary data
func (h *Handler) GetDashboard(c *gin.Context) {
	// Calculate totals from DB or Sync live
	// For speed, read DB CycleStatus
	var cycles []model.CycleStatus
	h.Repo.Find(&cycles)

	// Just return cycles for now
	c.JSON(http.StatusOK, gin.H{
		"cycles": cycles,
	})
}

// GetSettings
func (h *Handler) GetSettings(c *gin.Context) {
	var settings model.UserSettings
	if err := h.Repo.First(&settings).Error; err != nil {
		// Return default
		c.JSON(http.StatusOK, model.UserSettings{
			Principal:  10000,
			SplitCount: 40,
			TargetRate: 0.10,
			IsActive:   false,
			Symbols:    "TQQQ",
		})
		return
	}
	c.JSON(http.StatusOK, settings)
}

// UpdateSettings
func (h *Handler) UpdateSettings(c *gin.Context) {
	var input model.UserSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Upsert
	var settings model.UserSettings
	if err := h.Repo.First(&settings).Error; err != nil {
		h.Repo.Create(&input)
	} else {
		settings.Principal = input.Principal
		settings.SplitCount = input.SplitCount
		settings.TargetRate = input.TargetRate
		settings.IsActive = input.IsActive
		settings.Symbols = input.Symbols
		h.Repo.Save(&settings)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) TriggerSync(c *gin.Context) {
	if err := h.Strategy.SyncState(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "synced"})
}

// GetRebalancePreview
func (h *Handler) GetRebalancePreview(c *gin.Context) {
	plan, err := h.Strategy.CalculateRebalancePlan()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// ExecuteRebalance
func (h *Handler) ExecuteRebalance(c *gin.Context) {
	// Optional dry_run param
	dryRun := c.Query("dry_run") == "true"

	if err := h.Strategy.ExecuteRebalance(dryRun); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "executed", "dry_run": dryRun})
}

// ExecuteCustomRebalance accepts a custom plan from the frontend
func (h *Handler) ExecuteCustomRebalance(c *gin.Context) {
	dryRun := c.Query("dry_run") == "true"

	var customPlan service.RebalancePlan
	if err := c.ShouldBindJSON(&customPlan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan format: " + err.Error()})
		return
	}

	// Basic validation
	if len(customPlan.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plan must contain at least one item"})
		return
	}

	if err := h.Strategy.ExecuteCustomRebalance(&customPlan, dryRun); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "custom plan executed", "dry_run": dryRun})
}
