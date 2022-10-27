package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/phx/client"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/pei"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync $JOB_ID",
	Short: "sync remote job results with local",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		if !loggedIn {
			fmt.Println("❌ You should first log in to your Phoenix account!")
			return
		}
		if !isProjectInitialized() {
			fmt.Println(`❌ You should run this command in a project that contains the ".phoenix" directory!
  If this is indeed your project, please run "phx init" first.`)
			return
		}

		var (
			jobid = args[0]

			jobClient    = client.JobClient(baseClient)
			bucketClient = client.BucketClient(baseClient)
		)
		job, err := jobClient.Get(jobid)
		if err != nil {
			log.Fatalln("Could not get job:", err)
		}

		resultID, ok := castFst[string](job.V("result"))
		if !ok {
			log.Fatalln("Job not done yet.")
		}

		dir, err := os.MkdirTemp("", "result")
		if err != nil {
			log.Fatalln("Could not create temp dir:", err)
		}
		filename := path.Join(dir, resultID+".tar.gz")

		f, err := os.Create(filename)
		if err != nil {
			log.Fatalln("Could not open file:", err)
		}
		defer f.Close()

		resultBucket, err := bucketClient.Get(resultID)
		if err != nil {
			log.Fatalln("Could not get result:", err)
		}
		err = bucketClient.PopBucket(*resultBucket, f)
		if err != nil {
			log.Fatalln("Could not download result:", err)
		}

		p, err := pei.LoadPEI(".phoenix")
		if err != nil {
			log.Fatalln("Cannot load PEI, check .phoenix directory:", err)
		}
		_, err = p.Do(pei.Unpack{
			EggPack: filename,
		})
		if err != nil {
			log.Fatalln("Cannot run TAR:", err)
		}

		if !viper.GetBool("quiet") {
			fmt.Println("synced successfully.")
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
