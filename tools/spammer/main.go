package main

import (
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/izuc/zipp/client"
)

const (
	cfgNodeURI = "nodeURIs"
	cfgRate    = "rate"
	cfgEnable  = "enable"
	cfgImif    = "imif"
	cfgUnit    = "unit"
)

func init() {
	flag.StringSlice(cfgNodeURI, []string{"http://127.0.0.1:8080"}, "the URI of the node APIs")
	flag.Int(cfgRate, 100, "spam count in blocks per time unit")
	flag.Bool(cfgEnable, false, "enable/disable spammer")
	flag.String(cfgImif, "uniform", "inter block issuing function: uniform or poisson")
	flag.String(cfgUnit, "mps", "time unit of the spam rate: mpm or mps")
}

func main() {
	// example usage:
	//   go run main.go --nodeURIs=http://127.0.0.1:8080 --rate=1000 --enable=true --imif=uniform --unit=mpm
	flag.Parse()
	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		panic(err)
	}

	rate := viper.GetInt(cfgRate)
	if rate <= 0 {
		panic("invalid value for `rate` [>0]")
	}
	enableSpammer := viper.GetBool(cfgEnable)
	imif := viper.GetString(cfgImif)
	unit := viper.GetString(cfgUnit)

	var apis []*client.ZIPPAPI
	for _, api := range viper.GetStringSlice(cfgNodeURI) {
		apis = append(apis, client.NewZIPPAPI(api))
	}

	for _, api := range apis {
		resp, err := api.ToggleSpammer(enableSpammer, rate, unit, imif)
		if err != nil {
			fmt.Println(strings.ReplaceAll(err.Error(), "\n", ""))
			continue
		}
		fmt.Println(resp)
	}
}
