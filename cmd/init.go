package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .phoenix directory",
	Run: func(cmd *cobra.Command, args []string) {
		if info, err := os.Stat("./.phoenix"); err == nil && info.IsDir() {
			// TODO maybe it's better to walk the directory and only create the missing files
			fmt.Println("Phoenix is already initialized in this project! Nothing to do...")
			return
		}
		if err := copyFs(scripts, "scripts", "./.phoenix"); err != nil {
			log.Fatalln(err)
		}
		if !viper.GetBool("quiet") {
			fmt.Println(`Directory '.phoenix' written. Ready to GO!
 $ phx run --cluster $CLUSTER --flavor $FLAVOR $CMD [...$ARGS]`)
		}
	},
}

func copyFs(filesystem fs.FS, from, into string) error {
	fs.WalkDir(filesystem, from,
		func(path string, d fs.DirEntry, _ error) error {
			path2 := filepath.Join(into,
				strings.TrimPrefix(path, from))
			if name := d.Name(); name == "." || name == ".." {
				return nil
			}
			if d.IsDir() {
				return os.MkdirAll(path2, 0700)
			}

			err := func() error {
				f, err := filesystem.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				f2, err := os.Create(path2)
				if err != nil {
					return err
				}
				defer f2.Close()

				_, err = io.Copy(f2, f)
				if err != nil {
					return err
				}
				return err
			}()
			if err != nil {
				return err
			}
			return os.Chmod(path2, 0700)
		})
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
