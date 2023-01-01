package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/util"

	"github.com/RoboEpics/phx/client"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/pei"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run $CMD [...$ARGS]",
	Short: "Run your job remotely",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		for _, required := range []string{
			"cluster", "flavor",
		} {
			val := viper.Get(required)
			if val == nil {
				return fmt.Errorf("required config %s not provided", required)
			}
		}
		return nil
	},
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
			name        = viper.GetString("name")
			cluster     = viper.GetString("cluster")
			flavor      = viper.GetString("flavor")
			sa          = viper.GetString("sa")
			createSA    = viper.GetBool("create-sa")
			enableProxy = viper.GetBool("enable-proxy")

			jobClient    = client.JobClient(baseClient)
			bucketClient = client.BucketClient(baseClient)
			saClient     = client.ServiceAccountClient(baseClient)
		)

		bucketID := newID(name)
		bucketObj := client.Object{
			ID: bucketID,
			Value: map[string]any{
				"file":   bucketID,
				"bucket": bucketID,
			},
		}
		bucketObj.Annotations = map[string]string{
			"owner": baseClient.Token.UUID(),
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

		if sa == "" && createSA {
			sa = newID(name)
			saObject := client.Object{
				ID:   sa,
				Name: name,
				Annotations: map[string]string{
					"owner": baseClient.Token.UUID(),
				},
				Value: map[string]any{},
			}
			err := saClient.Create(saObject)
			if err != nil {
				log.Fatalln(err)
			}
		}

		jobID := newID(name)
		var jobValue = map[string]any{
			"cluster":         cluster,
			"flavor":          flavor,
			"cmd":             args[0],
			"args":            args[1:],
			"repo":            bucketID,
			"service_account": sa,
		}

		if enableProxy {
			proxyKey := util.RandomStr(util.CharsetHex, 32)
			jobValue["proxy_key"] = proxyKey
		}

		jobObj := client.Object{
			ID:    jobID,
			Name:  name,
			Value: jobValue,
		}
		if err := jobClient.Create(jobObj); err != nil {
			log.Fatalln("Cannot create Job:", err)
		}

		fmt.Println("Bucket:", bucketID)
		if createSA {
			fmt.Println("Service Account:", sa)
		}
		fmt.Println("Job:", jobID)
		if !viper.GetBool("quiet") {
			fmt.Println(`
In order to get job statuses, run:
 $ phx status`)
		}
	},
}

func init() {
	runCmd.Flags().StringP("cluster", "c", "", "Cluster name")
	runCmd.Flags().StringP("flavor", "f", "", "Flavor name")
	runCmd.Flags().StringP("name", "n", "", "name")
	runCmd.Flags().String("sa", "", "ServiceAccount name")
	runCmd.Flags().Bool("create-sa", false, "Create new ServiceAccount for this job")
	runCmd.Flags().Bool("enable-proxy", false, "Enable proxy for this job")

	rootCmd.AddCommand(runCmd)
}
