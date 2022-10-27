package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete $JOB_ID",
	Short: "Delete job by its ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !loggedIn {
			fmt.Println("❌ You should first log in to your Phoenix account!")
			return
		}

		var (
			jobID     = args[0]
			jobClient = baseClient.For("jobs")
		)

		job, err := jobClient.Get(jobID)
		if err != nil {
			log.Fatalln("Error getting job:", err)
		}

		err = jobClient.Delete(*job)
		if err != nil {
			log.Fatalln("Error deleting job:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
