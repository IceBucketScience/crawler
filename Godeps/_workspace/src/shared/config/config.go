package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

//TODO: Lift this into a shared package

func GetConfigVars(configVars interface{}) {
	err := envconfig.Process("", configVars)
	if err != nil {
		log.Panicln(err)
	}

	//TODO: checks for missing env variables
}
