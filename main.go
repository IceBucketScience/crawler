package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"crawler/config_vars"
	"shared/config"
	"shared/msg_queue"
)

type SuccessfulIndexMessage struct {
	UserId string `json:"userId"`
}

func main() {
	var configuration configVars.Configuration
	config.GetConfigVars(&configuration)

	server := mux.NewRouter()

	indexingJobCompletionQueue, indexingJobCompletionQueueCreationErr := msgQueue.CreateDispatcherQueue("indexing_job_completion_queue")
	indexingJobQueue, indexingJobQueueCreationErr := msgQueue.CreateRecieverQueue("indexing_job_queue", configuration.BaseUrl, server)

	if indexingJobQueueCreationErr != nil {
		log.Panicln(indexingJobQueueCreationErr)
	} else if indexingJobCompletionQueueCreationErr != nil {
		log.Panicln(indexingJobCompletionQueueCreationErr)
	}

	indexingJobQueue.RegisterCallback("INDEX_REQUEST", func(payload map[string]interface{}) {
		time.Sleep(0 * time.Second)

		indexingJobCompletionQueue.PushMessage("SUCCESS", SuccessfulIndexMessage{UserId: payload["userId"].(string)})
	})

	log.Fatal(http.ListenAndServe(":"+configuration.Port, server))
}
