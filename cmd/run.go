package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/RoboEpics/phx/client"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/pei"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run --cluster $CLUSTER --flavor $FLAVOR [--name $NAME] $CMD [...$ARGS]",
	Short: "Run your job remotely",
	Args:  cobra.MinimumNArgs(1),
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
			name    = viper.GetString("name")
			cluster = viper.GetString("cluster")
			flavor  = viper.GetString("flavor")

			jobClient    = client.JobClient(baseClient)
			bucketClient = client.BucketClient(baseClient)
		)

		bucketID := newID(name)
		bucketObj := client.Object{
			ID: bucketID,
			Value: map[string]any{
				"file":   bucketID,
				"bucket": bucketID,
			},
		}
		err := bucketClient.Create(bucketObj)
		if err != nil {
			log.Fatalln("Cannot create bucket:", err)
		}

		dir, err := os.MkdirTemp("", "repo")
		if err != nil {
			log.Fatalln("Cannot create dir:", err)
		}
		filename := path.Join(dir, bucketID+".tar.gz")

		p, err := pei.LoadPEI(".phoenix")
		if err != nil {
			log.Fatalln("Cannot load PEI, check .phoenix directory:", err)
		}
		_, err = p.Do(pei.TAR{
			TarFile: filename,
		})
		if err != nil {
			log.Fatalln("Cannot run TAR:", err)
		}

		f, err := os.Open(filename)
		if err != nil {
			log.Fatalln("Cannot open file:", err)
		}
		defer f.Close()

		err = bucketClient.PushBucket(bucketObj, f)
		if err != nil {
			log.Fatalln("Cannot push bucket:", err)
		}

		jobID := newID(name)
		jobObj := client.Object{
			ID: jobID,
			Value: map[string]any{
				"cluster": cluster,
				"flavor":  flavor,
				"cmd":     args[0],
				"args":    args[1:],
				"repo":    bucketID,
			},
		}
		if err := jobClient.Create(jobObj); err != nil {
			log.Fatalln("Cannot create Job:", err)
		}

		fmt.Println("bucket:", bucketID)
		fmt.Println("job:", jobID)
		if !viper.GetBool("quiet") {
			fmt.Println(`
In order to get job statuses, run:
 $ phx status`)
		}
	},
}

func init() {
	runCmd.Flags().StringP("cluster", "c", "", "Cluster name")
	runCmd.MarkFlagRequired("cluster")
	runCmd.Flags().StringP("flavor", "f", "", "Flavor name")
	runCmd.MarkFlagRequired("flavor")
	runCmd.Flags().StringP("name", "n", "", "name")

	rootCmd.AddCommand(runCmd)
}
