package cmd

import (
	"fmt"
	"log"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/phx/client"
)

// jupyteStatusCmd represents the jupyteStatus command
var jupyteStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		if !loggedIn {
			fmt.Println("❌ You should first log in to your Phoenix account!")
			return
		}

		var (
			jobClient = client.JobClient(baseClient)
		)

		jobs, err := jobClient.List(
			map[string]string{
				"owner": baseClient.Token.UUID(),
				"type":  "jupyter",
			})
		if err != nil {
			log.Fatalln("Cannot list jupyters:", err)
		}

		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].CreatedAt.After(jobs[i].CreatedAt)
		})
		for _, job := range jobs {
			var (
				result, _     = castFst[string](job.V("result"))
				_, exitcodeOk = castFst[int](job.V("exit_code"))
			)
			if result == "" || !exitcodeOk {
				fmt.Printf("%s: %s", job.ID, "RUNNING\n")
			} else {
				fmt.Printf("%s: EXITED \n", job.ID)
			}
		}
		if !viper.GetBool("quiet") {
			fmt.Printf("%d items returned\n", len(jobs))
		}
	},
}

func init() {
	jupyterCmd.AddCommand(jupyteStatusCmd)
}
