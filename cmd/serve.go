package cmd

import (
	"os"

	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/server"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)


var port int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve the WEB UI",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := utils.OpenDB()
		if err != nil {
			log.Fatal(err)
		}

		if err := migration.RunMigrations(db); err != nil {
			log.Fatal(err)
		}

		if os.Getenv("PAISA_DEBUG") == "true" {
			db = db.Debug()
		}

		// Pre-warm the price and rate BTree caches in the background so that
		// the first API request does not pay the full cold-start cost.
		service.WarmCaches(db)

		server.Listen(db, port)
	},

}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&port, "port", "p", 7500, "port to listen on")
}
