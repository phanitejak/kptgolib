package main

import (
	"os"
	"path/filepath"

	"github.com/phanitejak/gopkg/tools/service-generator/generator"
	"github.com/phanitejak/gopkg/tools/service-generator/utils"
)

// osExit allows overwritting os.Exit for testing purposes.
var osExit = os.Exit

func main() {
	/* svcDirectory, err := utils.GetSvcDirectoryPath()
	if err != nil {
		utils.Log(err.Error(), utils.ERROR)
		osExit(1)
		return
	}

	if !utils.Exists(svcDirectory) {
		utils.Log(fmt.Sprintf("%v is not go monorepo", svcDirectory), utils.ERROR)
		osExit(1)
		return
	} */

	svcDirectory := "./svc"
	var conf utils.Config

	if len(os.Args) == 1 {
		var err error
		conf, err = utils.ConfigFromPrompt(os.Stdin, os.Stdout)
		if err != nil {
			utils.Log(err.Error(), utils.ERROR)
			osExit(1)
			return
		}
	} else {
		var err error
		conf, err = utils.ConfigFromFlags(os.Args)
		if err != nil {
			utils.Log(err.Error(), utils.ERROR)
			osExit(1)
			return
		}
	}

	userName, userEmail := utils.GetGitInfo()
	conf.UserName = userName
	conf.UserEmail = userEmail

	if appDir := filepath.Join(svcDirectory, conf.ServiceName); utils.Exists(appDir) {
		if err := generator.Regenerate(svcDirectory, conf); err != nil {
			utils.Log(err.Error(), utils.ERROR)
			osExit(1)
			return
		}
		return
	}

	if err := generator.Generate(svcDirectory, conf); err != nil {
		utils.Log(err.Error(), utils.ERROR)
		osExit(1)
		return
	}
}
