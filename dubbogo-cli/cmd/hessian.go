package cmd

import (
	"runtime"
)

import (
	"github.com/spf13/cobra"
)

import (
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/generator/sample"
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/generator/sample/hessian"
)

// hessianCmd represents the hessian-register-generator command
var hessianCmd = &cobra.Command{
	Use:   "hessian",
	Short: "automatic generate hessian pojo register statement",
	Run:   generateHessianRegistry,
}

func init() {
	rootCmd.AddCommand(hessianCmd)
	hessianCmd.Flags().StringP(hessian.CmdFlagInclude, "i", "./", "file scan directory path, default `./`")
	hessianCmd.Flags().IntP(hessian.CmdFlagThread, "t", runtime.NumCPU()*2, "worker thread limit, default (cpu core) * 2")
	hessianCmd.Flags().BoolP(hessian.CmdFlagOnlyError, "e", false, "only print error message, default false")
}

func generateHessianRegistry(cmd *cobra.Command, _ []string) {
	var include string
	var thread int
	var onlyError bool
	var err error

	include, err = cmd.Flags().GetString(hessian.CmdFlagInclude)
	if err != nil {
		panic(err)
	}

	thread, err = cmd.Flags().GetInt(hessian.CmdFlagThread)
	if err != nil {
		panic(err)
	}

	onlyError, err = cmd.Flags().GetBool(hessian.CmdFlagOnlyError)
	if err != nil {
		panic(err)
	}

	if err = sample.HessianRegistryGenerate(include, thread, onlyError); err != nil {
		panic(err)
	}
}
