package cmd

import (
	"embed"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/phx/client"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/token"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	scripts embed.FS

	baseClient client.Client
	loggedIn   bool

	defaults = map[string]any{
		"remote":  "https://staging.api.phoenix.roboepics.com",
		"gateway": "wss://staging.gateway.phoenix.roboepics.com",
	}
)

var rootCmd = &cobra.Command{
	Use: "phx",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
}

func Execute(scr embed.FS) {
	scripts = scr
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func isProjectInitialized() bool {
	info, err := os.Stat("./.phoenix")
	return err == nil && info.IsDir()
}

func init() {
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.phoenix")
	viper.SetEnvPrefix("PHX")
	viper.AutomaticEnv()
	viper.ReadInConfig()

	for k, v := range defaults {
		viper.SetDefault(k, v)
	}

	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringP("remote", "r", "", "Remote address")

	remotePeer := viper.GetString("remote")
	tokenObj := token.NewDefaultJWTToken()
	// tokenObj := token.NewStaticToken("$udo", "", nil)
	if tokenObj.IsLoggedIn() {
		loggedIn = true
	}
	baseClient = client.Client{
		Token:     tokenObj,
		APIServer: remotePeer,
		HTTP:      http.DefaultClient,
	}
	rand.Seed(time.Now().Unix())
}

func mustln[T any](msg ...any) func(v any, ok bool) T {
	return func(v any, ok bool) T {
		if !ok {
			log.Fatalln(msg...)
		}
		cast, ok := v.(T)
		if !ok {
			log.Fatalln(msg...)
		}
		return cast
	}
}

func castFst[T any](v any, ok bool) (out T, allok bool) {
	if !ok {
		return
	}
	out, allok = v.(T)
	return
}

func newID(parts ...string) string {
	const (
		length  = 32
		charset = "ABCDEF1234567890"
	)
	joined := strings.Join(parts, "-")
	if len(joined) > length {
		panic("too long parts")
	}
	if len(joined) < length {
		if len(joined) > 0 {
			joined += "-"
		}
		rnd := make([]byte, length-len(joined))
		for i := range rnd {
			rnd[i] = charset[rand.Intn(len(charset))]
		}
		joined += string(rnd)
	}
	return joined
}
