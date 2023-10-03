package evillogger

import (
	"fmt"

	"github.com/izuc/zipp.foundation/core/configuration"
	"github.com/izuc/zipp.foundation/core/logger"
)

var New = logger.NewLogger

func init() {
	config := configuration.New()
	err := config.Set(logger.ConfigurationKeyOutputPaths, []string{"evil-spammer.log"})
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = logger.InitGlobalLogger(config); err != nil {
		panic(err)
	}
	logger.SetLevel(logger.LevelInfo)
}
