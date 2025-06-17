package cmd

import (
	transcode "vimai/ads-transcode/internal"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	command := &cobra.Command{
		Use:   "serve",
		Short: "serve transcode server",
		Long:  `serve transcode server`,
		Run:   transcode.Service.ServeCommand,
	}

	cobra.OnInitialize(func() {
		transcode.Service.ServerConfig.Set()
		transcode.Service.Preflight()
	})

	if err := transcode.Service.ServerConfig.Init(command); err != nil {
		log.Panic().Err(err).Msg("unable to run serve command")
	}

	root.AddCommand(command)
}
