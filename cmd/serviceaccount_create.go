package cmd

import (
	"fmt"
	"log"

	"github.com/RoboEpics/phx/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serviceAccountCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create new ServiceAccount",
	Aliases: []string{"new"},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			name     = viper.GetString("name")
			saClient = client.ServiceAccountClient(baseClient)
		)
		id := newID(name)
		saObject := client.Object{
			ID:   id,
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
		fmt.Println("serviceAccount:", id)
		if !viper.GetBool("quiet") {
			fmt.Println(`
You can use this ServiceAccount by:
$ phx run --sa $SERVICEACCOUNT_ID ...
$ phx jupyter create --sa $SERVICEACCOUNT_ID ...`)
		}
	},
}

func init() {
	serviceaccountCmd.AddCommand(serviceAccountCreateCmd)
	serviceAccountCreateCmd.Flags().StringP("name", "n", "", "name")
}
