package cmd

import (
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/phx/client"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/proxy"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/util"
)

// jupyteAttachCmd represents the jupyteAttach command
var jupyterAttachCmd = &cobra.Command{
	Use:   "attach $JUPYTER_ID [$LOCALPORT]",
	Short: "Attach remote running jupyter kernel to your Localhost",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			jobID   = args[0]
			gateway = viper.GetString("gateway")

			jobClient = client.JobClient(baseClient)
		)

		localStr := "8888"
		if len(args) >= 2 {
			localStr = args[1]
		}

		local, err := strconv.Atoi(localStr)
		if err != nil {
			log.Fatalln(err)
		}
		remote := 8888

		job, err := jobClient.Get(jobID)
		if err != nil {
			log.Fatalln("Cannot get job:", err)
		}

		proxyKey := mustln[string](
			"Proxy key not provided.")(
			job.V("proxy_key"))
		log.Printf("Copy http://localhost:%d/ into your Google Colab Local Kernel dialog.\n", local)
		node := proxy.Node{
			DialersCount:         2,
			MinConns:             2,
			Key:                  []byte(proxyKey),
			DisableIncomingConns: true,
		}
		defer node.Close()

		var (
			name   = util.RandomStr(util.CharsetHex, 32)
			secret = util.RandomStr(util.CharsetHex, 32)
			d      = proxy.WebsocketDialer(gateway, name, secret)
		)
		if err := node.Connect("root", d); err != nil {
			log.Fatalln("Cannot connect remote gateway:", err)
		}

		if err := node.ListenProxy(
			[]string{"root", jobID},
			proxy.IPPort{Port: local, IP: "127.0.0.1"},
			proxy.IPPort{Port: remote, IP: "127.0.0.1"},
		); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	jupyterAttachCmd.Flags().StringP("gateway", "g", "", "Gateway URL")
	jupyterCmd.AddCommand(jupyterAttachCmd)
}
