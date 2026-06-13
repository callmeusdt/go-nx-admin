package commands

import (
	"io/fs"

	"github.com/spf13/cobra"
)

var embedFS fs.FS

func Execute(spaFS fs.FS) {
	embedFS = spaFS
	rootCmd := &cobra.Command{
		Use:   "nx-admin",
		Short: "NX Admin - Go 通用后台管理系统",
		Run:   runServe,
	}
	rootCmd.AddCommand(ServeCmd())
	rootCmd.AddCommand(ResetPasswordCmd())
	rootCmd.Execute()
}
