package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"crawler/config_vars"
	"crawler/index"
	"shared/config"
	"shared/facebook"
	"shared/graph"
	"shared/msg_queue"
)

func main() {
	var configuration configVars.Configuration
	config.GetConfigVars(&configuration)

	server := mux.NewRouter()

	facebook.InitFbClient(configuration.FbAppId, configuration.FbAppSecret)
	graph.InitGraph(configuration.DbUrl)

	startIndexRequestHandler(server, &configuration)

	log.Fatalln(http.ListenAndServe(":"+configuration.Port, server))
}

func startIndexRequestHandler(server *mux.Router, configuration *configVars.Configuration) {
	indexingJobCompletionQueue, indexingJobCompletionQueueCreationErr := msgQueue.CreateDispatcherQueue("indexing_job_completion_queue")
	indexingJobQueue, indexingJobQueueCreationErr := msgQueue.CreateRecieverQueue("indexing_job_queue", configuration.BaseUrl, server)

	if indexingJobQueueCreationErr != nil {
		log.Panicln(indexingJobQueueCreationErr)
	} else if indexingJobCompletionQueueCreationErr != nil {
		log.Panicln(indexingJobCompletionQueueCreationErr)
	}

	indexingJobQueue.RegisterCallback("INDEX_REQUEST", handleIndexRequest(indexingJobCompletionQueue))
}

type SuccessfulIndexMessage struct {
	UserId string `json:"userId"`
}

type FailedIndexMessage struct {
	UserId  string `json:"userId"`
	Message string `json:"message"`
}

func handleIndexRequest(indexingJobCompletionQueue *msgQueue.DispatcherQueue) func(map[string]interface{}) {
	return func(payload map[string]interface{}) {
		var completionMessageType string
		var completionMessagePayload interface{}

		indexingErr := index.IndexVolunteer(payload["userId"].(string), payload["accessToken"].(string))
		if indexingErr != nil {
			completionMessageType = "FAILURE"
			completionMessagePayload = FailedIndexMessage{UserId: payload["userId"].(string), Message: indexingErr.Error()}
			log.Println(indexingErr)
		} else {
			completionMessageType = "SUCCESS"
			completionMessagePayload = SuccessfulIndexMessage{UserId: payload["userId"].(string)}
		}

		pushErr := indexingJobCompletionQueue.PushMessage(completionMessageType, completionMessagePayload)
		if pushErr != nil {
			log.Println(pushErr)
		}
	}
}
