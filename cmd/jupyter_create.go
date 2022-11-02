package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/pei"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/RoboEpics/phx/client"
)

// jupyterCreateCmd represents the jupyterCreate command
var jupyterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a jupyter kernel",
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
			ID:   newID(name),
			Name: name,
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

		proxyKey := util.RandomStr(util.CharsetHex, 32)

		jobID := newID(name)
		jobObj := client.Object{
			ID:   jobID,
			Name: name,
			Annotations: map[string]string{
				"type": "jupyter",
			},
			Value: map[string]any{
				"cluster": cluster,
				"flavor":  flavor,
				"cmd":     "jupyter",
				"args": []string{
					"notebook",
					"--port=8888",
					"--ip=*", "--NotebookApp.allow_origin=*",
					"--NotebookApp.token=", "--NotebookApp.password=",
					"--NotebookApp.port_retries=0",
					"--allow-root",
				},
				"proxy_key": proxyKey,
				"repo":      bucketID,
			},
		}
		if err := jobClient.Create(jobObj); err != nil {
			log.Fatalln("Cannot create Job:", err)
		}

		fmt.Println("bucket:", bucketID)
		fmt.Println("jupyter:", jobID)
		if !viper.GetBool("quiet") {
			fmt.Println(`
Try to attach to your jupyter, run:
 $ phx jupyter attach
In order to get jupyter statuses, run:
 $ phx jupyter status`)
		}
	},
}

func init() {
	jupyterCreateCmd.Flags().StringP("cluster", "c", "", "Cluster name")
	jupyterCreateCmd.MarkFlagRequired("cluster")
	jupyterCreateCmd.Flags().StringP("flavor", "f", "", "Flavor name")
	jupyterCreateCmd.MarkFlagRequired("flavor")
	jupyterCreateCmd.Flags().StringP("name", "n", "", "name")

	jupyterCmd.AddCommand(jupyterCreateCmd)
}
