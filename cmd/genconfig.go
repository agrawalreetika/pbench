package cmd

import (
	"github.com/spf13/cobra"
	"pbench/cmd/genconfig"
)

var genConfigCmd = &cobra.Command{
	Use:                   `genconfig [flags] [directory to search recursively for config.json]`,
	DisableFlagsInUseLine: true,
	Run:                   genconfig.Run,
	Args:                  cobra.ExactArgs(1),
	Short:                 "Generate benchmark cluster configurations",
}

func init() {
	rootCmd.AddCommand(genConfigCmd)
	genConfigCmd.Flags().StringVarP(&genconfig.TemplatePath, "template-dir",
		"t", "", "Specifies the template directory. Use built-in template if not specified.")
	genConfigCmd.Flags().StringVarP(&genconfig.ParameterPath, "parameter-file",
		"p", "", "Specifies the parameter file. Use built-in defaults if not specified.")
	genConfigCmd.AddCommand(&cobra.Command{
		Use:                   "default",
		Short:                 "Print the built-in default generator parameter file.",
		DisableFlagsInUseLine: true,
		Run:                   genconfig.PrintDefaultParams,
	})
}
