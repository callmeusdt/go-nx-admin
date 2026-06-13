package commands

import (
	"github.com/spf13/cobra"
	"go-nx-admin/app"
)

func ServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "启动 HTTP 服务（监听端口，默认 0.0.0.0:19500）",
		Run:   runServe,
	}
}

func runServe(cmd *cobra.Command, args []string) {
	app.Run(app.Options{EmbedFS: embedFS})
}
