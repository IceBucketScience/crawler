package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"

	"shared/msg_queue"
)

type Configuration struct {
	BaseUrl string `envconfig:"BASE_URL"`
	Port    string `envconfig:"PORT"`

	FbAppId     string `envconfig:"FB_APP_ID"`
	FbAppSecret string `envconfig:"FB_APP_SECRET"`
}

//TODO: Lift this into a shared package

func GetConfigVars() *Configuration {
	var config Configuration
	err := envconfig.Process("", &config)
	if err != nil {
		log.Panicln(err)
	}

	//TODO: checks for missing env variables

	return &config
}

func main() {
	configVars := GetConfigVars()

	server := mux.NewRouter()

	indexingJobCompletionQueue := msgQueue.CreateDispatcherQueue("indexing_job_completion_queue")

	indexingJobQueue := msgQueue.CreateRecieverQueue("indexing_job_queue", configVars.BaseUrl, server)

	indexingJobQueue.RegisterCallback("INDEX_REQUEST", func(payload map[string]interface{}) {
		log.Println(payload)

		indexingJobCompletionQueue.PushMessage("SUCCESS", struct {
			Test string
		}{Test: "derp"})
	})

	log.Fatal(http.ListenAndServe(":"+configVars.Port, server))
}
