package cmd

import (
	"log"
	"strconv"
	"strings"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/proxy"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/RoboEpics/phx/client"
)

// tunnelCmd represents the tunnel command
var tunnelCmd = &cobra.Command{
	Use:   "tunnel $JOB_ID [$LOCAL_PORT:]$REMOTE_PORT",
	Short: "tunnel job TCP network traffic to your localhost",
	Args:  cobra.ExactArgs(2),
	Run:   runTunnelCmd,
}

func runTunnelCmd(cmd *cobra.Command, args []string) {
	var (
		jobID       = args[0]
		localRemote = strings.SplitN(args[1], ":", 2)
		gateway     = viper.GetString("gateway")

		jobClient = client.JobClient(baseClient)
	)

	localStr := localRemote[0]
	remoteStr := localStr
	if len(localRemote) >= 2 {
		remoteStr = localRemote[1]
	}

	local, err := strconv.Atoi(localStr)
	if err != nil {
		log.Fatalln(err)
	}
	remote, err := strconv.Atoi(remoteStr)
	if err != nil {
		log.Fatalln(err)
	}

	job, err := jobClient.Get(jobID)
	if err != nil {
		log.Fatalln("Cannot get job:", err)
	}

	proxyKey := mustln[string](
		"proxy_key not provided for job", job.ID)(
		job.V("proxy_key"))
	log.Printf("Listening on 127.0.0.1:%d...\n", local)
	node := proxy.Node{
		DialersCount:         2,
		MinConns:             4,
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
}

func init() {
	tunnelCmd.Flags().StringP("gateway", "g", "ws://gateway.phoenix.roboepics.com:2131", "Gateway URL")
	rootCmd.AddCommand(tunnelCmd)
}
