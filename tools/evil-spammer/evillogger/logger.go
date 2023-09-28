package evillogger

import (
	"fmt"

	"github.com/izuc/zipp.foundation/app/configuration"
	appLogger "github.com/izuc/zipp.foundation/app/logger"
	"github.com/izuc/zipp.foundation/logger"
)

var New = logger.NewLogger

func init() {
	config := configuration.New()
	err := config.Set(logger.ConfigurationKeyOutputPaths, []string{"evil-spammer.log", "stdout"})
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = appLogger.InitGlobalLogger(config); err != nil {
		panic(err)
	}
	logger.SetLevel(logger.LevelInfo)
}
