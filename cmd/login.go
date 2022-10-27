package cmd

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/token"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logs you in to Phoenix platform",
	Run: func(cmd *cobra.Command, args []string) {
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
	},
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

func init() {
	rootCmd.AddCommand(loginCmd)
}
