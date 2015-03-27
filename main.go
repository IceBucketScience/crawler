package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"crawler/config_vars"
	"shared/config"
	"shared/msg_queue"
)

func main() {
	var configuration configVars.Configuration
	config.GetConfigVars(&configuration)

	server := mux.NewRouter()

	indexingJobCompletionQueue := msgQueue.CreateDispatcherQueue("indexing_job_completion_queue")

	indexingJobQueue := msgQueue.CreateRecieverQueue("indexing_job_queue", configuration.BaseUrl, server)

	indexingJobQueue.RegisterCallback("INDEX_REQUEST", func(payload map[string]interface{}) {
		log.Println(payload)

		indexingJobCompletionQueue.PushMessage("SUCCESS", struct {
			Test string
		}{Test: "derp"})
	})

	log.Fatal(http.ListenAndServe(":"+configuration.Port, server))
}
