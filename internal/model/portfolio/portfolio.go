package portfolio

import (
	"context"

	"github.com/ananthakumaran/paisa/internal/config"
	dbutil "github.com/ananthakumaran/paisa/internal/db"
	sqlcdb "github.com/ananthakumaran/paisa/internal/db/sqlc"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Portfolio struct {
	ID                uint                 `gorm:"primaryKey" json:"id"`
	CommodityType     config.CommodityType `json:"commodity_type"`
	ParentCommodityID string               `json:"parent_commodity_id"`
	SecurityID        string               `json:"security_id"`
	SecurityName      string               `json:"security_name"`
	SecurityType      string               `json:"security_type"`
	SecurityRating    string               `json:"security_rating"`
	SecurityIndustry  string               `json:"security_industry"`
	Percentage        decimal.Decimal      `json:"percentage"`
}

func insertPortfolioParams(commodityType config.CommodityType, parentCommodityID string, portfolio *Portfolio) sqlcdb.InsertPortfolioParams {
	return sqlcdb.InsertPortfolioParams{
		CommodityType:     commodityType,
		ParentCommodityID: dbutil.NullString(parentCommodityID),
		SecurityID:        dbutil.NullString(portfolio.SecurityID),
		SecurityName:      dbutil.NullString(portfolio.SecurityName),
		SecurityType:      dbutil.NullString(portfolio.SecurityType),
		SecurityRating:    dbutil.NullString(portfolio.SecurityRating),
		SecurityIndustry:  dbutil.NullString(portfolio.SecurityIndustry),
		Percentage:        portfolio.Percentage,
	}
}

func UpsertAll(db *gorm.DB, commodityType config.CommodityType, parentCommodityID string, portfolios []*Portfolio) error {
	return db.Transaction(func(tx *gorm.DB) error {
		queries := dbutil.Queries(tx)
		if err := queries.DeletePortfoliosByTypeAndParent(context.Background(), sqlcdb.DeletePortfoliosByTypeAndParentParams{
			CommodityType:     commodityType,
			ParentCommodityID: dbutil.NullString(parentCommodityID),
		}); err != nil {
			return err
		}
		for _, portfolio := range portfolios {
			if err := queries.InsertPortfolio(context.Background(), insertPortfolioParams(commodityType, parentCommodityID, portfolio)); err != nil {
				return err
			}
		}

		return nil
	})
}

func mapPortfolio(row sqlcdb.Portfolio) Portfolio {
	return Portfolio{
		ID:                uint(row.ID),
		CommodityType:     row.CommodityType,
		ParentCommodityID: row.ParentCommodityID.String,
		SecurityID:        row.SecurityID.String,
		SecurityName:      row.SecurityName.String,
		SecurityType:      row.SecurityType.String,
		SecurityRating:    row.SecurityRating.String,
		SecurityIndustry:  row.SecurityIndustry.String,
		Percentage:        row.Percentage,
	}
}

func GetPortfolios(db *gorm.DB, parentCommodityID string) []Portfolio {
	rows, err := dbutil.Queries(db).ListPortfoliosByParent(context.Background(), dbutil.NullString(parentCommodityID))
	if err != nil {
		log.Fatal(err)
	}
	return makePortfolios(rows)
}

func makePortfolios(rows []sqlcdb.Portfolio) []Portfolio {
	portfolios := make([]Portfolio, 0, len(rows))
	for _, row := range rows {
		portfolios = append(portfolios, mapPortfolio(row))
	}
	return portfolios
}

func GetAllParentCommodityIDs(db *gorm.DB) []string {
	rows, err := dbutil.Queries(db).ListPortfolioParentCommodityIDs(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	parentCommodityIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Valid {
			parentCommodityIDs = append(parentCommodityIDs, row.String)
		}
	}
	return parentCommodityIDs
}
