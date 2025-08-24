package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"tinypng-cli/internal/config"
)

func Execute(version string) {

	var (
		showVersion, debugMode bool
	)

	rootCmd := &cobra.Command{
		Use:   "tinypng",
		Short: "A tiny CLI for tinypng",
		Long: `This tool is developed with TinyPNG's web api. 
You can find official documentation here: https://tinypng.com/developers/reference.
This tool requires a API key from TinyPNG, you can get it here: https://tinify.com/developers.
API key can be set by flag --api-key or env key TINYPNG_API_KEY`,

		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debugMode {
				log.SetFlags(log.LstdFlags | log.Lshortfile)
			} else {
				log.SetFlags(0)
				log.SetOutput(io.Discard)
			}
			return
		},
	}

	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "cli version")
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "enable debug")
	rootCmd.PersistentFlags().StringVarP(&config.APIKey, "api-key", "k", "", "tinypng api key")

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version)
		}
		return nil
	}

	rootCmd.AddCommand(CompressCmd())

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s\n", r)
			os.Exit(1)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
