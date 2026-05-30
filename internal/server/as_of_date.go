package server

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
)

func parseAsOfDateHelper(c *gin.Context, capFuture bool) (time.Time, bool) {
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
		if capFuture {
			asOfDate = today
		} else {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "as_of_date cannot be in the future")
			return time.Time{}, false
		}
	}
	return asOfDate, true
}

func parseAsOfDate(c *gin.Context) (time.Time, bool) {
	return parseAsOfDateHelper(c, false)
}

var financialYearRegex = regexp.MustCompile(`^\d{4}(\s*-\s*\d{2})?$`)

func parseAsOfDateOrYear(c *gin.Context) (time.Time, bool) {
	if c.Query("as_of_date") != "" {
		return parseAsOfDateHelper(c, true)
	}

	rawYear := strings.TrimSpace(c.Query("year"))
	if rawYear == "" {
		return utils.ToDate(utils.Now()), true
	}

	if !financialYearRegex.MatchString(rawYear) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid year format, expected YYYY or YYYY - YY")
		return time.Time{}, false
	}

	_, asOfDate := utils.ParseFY(rawYear)
	asOfDate = utils.ToDate(asOfDate)
	today := utils.ToDate(utils.Now())
	if asOfDate.After(today) {
		asOfDate = today
	}

	return asOfDate, true
}
