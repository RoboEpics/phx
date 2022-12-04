package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/token"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logs you in to Phoenix platform",
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("static") {
			uuid, err := promptUUID()
			if err != nil {
				fmt.Printf("❌ UUID prompt failed: %v\n", err)
				return
			}

			tokenStr, err := promptToken()
			if err != nil {
				fmt.Printf("❌ Token prompt failed: %v\n", err)
				return
			}

			viper.Set("uuid", uuid)
			viper.Set("token", tokenStr)
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			configDir := homeDir + "/.phoenix"
			if err := os.MkdirAll(configDir, 0700); err != nil {
				fmt.Printf("❌ Failed to create config directory %s: %v", configDir, err)
				return
			}
			err = viper.WriteConfigAs(configDir + "/config.yaml")
			if err != nil {
				fmt.Printf("❌ Writing config failed: %v\n", err)
				return
			}

			loggedIn = true
			fmt.Printf("✅ Successfully logged in as: %s\n", uuid)
		} else {
			username, err := promptUsername()
			if err != nil {
				fmt.Printf("❌ Username prompt failed: %v\n", err)
				return
			}

			password, err := promptPassword()
			if err != nil {
				fmt.Printf("❌ Password prompt failed: %v\n", err)
				return
			}

			err = baseClient.Token.(*token.JWTToken).Login(username, password)
			if err != nil {
				fmt.Printf("❌ Login Error: %v\n", err)
				return
			}

			loggedIn = true
			fmt.Printf("✅ Successfully logged in as: %s\n", username)
		}
	},
}

func promptUsername() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}: ",
		Valid:   "{{ . | bold }}: ",
		Invalid: "{{ . | bold }}: ",
		Success: "{{ . | bold }}: ",
	}

	prompt := promptui.Prompt{
		Label:     "Username/Email",
		Templates: templates,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func promptPassword() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}: ",
		Valid:   "{{ . | bold }}: ",
		Invalid: "{{ . | bold }}: ",
		Success: "{{ . | bold }}: ",
	}

	prompt := promptui.Prompt{
		Label:       "Password",
		HideEntered: true,
		Templates:   templates,
		Mask:        '*',
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func promptUUID() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}: ",
		Valid:   "{{ . | bold }}: ",
		Invalid: "{{ . | bold }}: ",
		Success: "{{ . | bold }}: ",
	}

	prompt := promptui.Prompt{
		Label:     "UUID",
		Templates: templates,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func promptToken() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }}: ",
		Valid:   "{{ . | bold }}: ",
		Invalid: "{{ . | bold }}: ",
		Success: "{{ . | bold }}: ",
	}

	prompt := promptui.Prompt{
		Label:       "Token",
		HideEntered: true,
		Templates:   templates,
		Mask:        '*',
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func init() {
	loginCmd.Flags().BoolP("static", "s", false, "Login with static token")

	rootCmd.AddCommand(loginCmd)
}
