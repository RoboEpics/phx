package cmd

import (
	"fmt"
	"log"
	"sort"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/phx/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status of running jobs",
	Run: func(cmd *cobra.Command, args []string) {
		if !loggedIn {
			fmt.Println("❌ You should first log in to your Phoenix account!")
			return
		}

		var (
			jobClient = client.JobClient(baseClient)
		)

		jobs, err := jobClient.List(map[string]string{
			"owner": baseClient.Token.UUID(),
		})
		if err != nil {
			log.Fatalln("Cannot list Jobs:", err)
		}

		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].CreatedAt.After(jobs[i].CreatedAt)
		})
		for _, job := range jobs {
			var (
				result, _      = castFst[string](job.V("result"))
				exitCode, exOk = castFst[int](job.V("exit_code"))
			)
			if !exOk {
				fmt.Printf("%s: %s", job.ID, "RUNNING\n")
			} else if result == "" {
				fmt.Printf("%s: EXITED (exit code %d)\n", job.ID, exitCode)
			} else {
				fmt.Printf("%s: DONE (exit code %d)\n", job.ID, exitCode)
			}
		}
		if !viper.GetBool("quiet") {
			fmt.Printf("%d items returned\n", len(jobs))
			fmt.Println(`
All results of 'DONE' jobs can be synced with your local 
project files. In order to do that write:
 $ phx sync $JOB_ID`)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
