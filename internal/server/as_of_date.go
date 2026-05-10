package server

import (
	"net/http"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
)

func parseAsOfDate(c *gin.Context) (time.Time, bool) {
	raw := c.Query("as_of_date")
	today := utils.ToDate(utils.Now())
	if raw == "" {
		return today, true
	}

	asOfDate, err := time.ParseInLocation("2006-01-02", raw, config.TimeZone())
	if err != nil {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid as_of_date format, expected YYYY-MM-DD")
		return time.Time{}, false
	}

	asOfDate = utils.ToDate(asOfDate)
	if asOfDate.After(today) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "as_of_date cannot be in the future")
		return time.Time{}, false
	}
	return asOfDate, true
}
