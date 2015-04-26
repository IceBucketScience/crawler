package msgQueue

import (
	"encoding/json"

	"github.com/iron-io/iron_go/mq"
)

type DispatcherQueue struct {
	Name      string
	IronQueue *mq.Queue
}

func CreateDispatcherQueue(name string) (*DispatcherQueue, error) {
	ironQueue := mq.New(name)

	_, err := ironQueue.Info()
	if err != nil {
		return nil, err
	}

	return &DispatcherQueue{Name: name, IronQueue: ironQueue}, nil
}

func (queue *DispatcherQueue) PushMessage(msgType string, payload interface{}) error {
	json, marshalErr := json.Marshal(Message{Type: msgType, Payload: payload})
	if marshalErr != nil {
		return marshalErr
	}

	_, pushErr := queue.IronQueue.PushString(string(json))
	if pushErr != nil {
		return pushErr
	}

	return nil
}
