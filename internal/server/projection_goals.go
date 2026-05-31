package server

import (
	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetProjectionGoals(db *gorm.DB) gin.H {
	_, cfg, lifeGoalMetadata := buildProjectionSimulationConfig(db, SimulateRequest{})
	probabilities := []simulator.GoalProbability{}
	if len(cfg.Goals) > 0 {
		probabilities = simulator.Run(cfg).GoalProbabilities
	}

	return gin.H{
		"goals": buildLifeGoalResponse(lifeGoalMetadata, probabilities),
	}
}
