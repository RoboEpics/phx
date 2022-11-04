package cmd

import (
	"fmt"
	"log"

	"github.com/RoboEpics/phx/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serviceAccountStatusCmd represents the serviceAccountStatus command
var serviceAccountStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "status of ServiceAccounts",
	Aliases: []string{"ls", "list"},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			saClient = client.ServiceAccountClient(baseClient)
		)
		sas, err := saClient.List(map[string]string{
			"owner": baseClient.Token.UUID(),
		})
		if err != nil {
			log.Fatalln(err)
		}
		for _, sa := range sas {
			fmt.Println(sa.ID)
		}
		if !viper.GetBool("quiet") {
			fmt.Println(len(sas), "items retured.")
			fmt.Println(`
You can use these ServiceAccounts by:
$ phx run --sa $SERVICEACCOUNT_ID ...
$ phx jupyter create --sa $SERVICEACCOUNT_ID ...`)
		}
	},
}

func init() {
	serviceaccountCmd.AddCommand(serviceAccountStatusCmd)
}
