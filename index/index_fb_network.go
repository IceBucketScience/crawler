package index

import (
	"log"

	"shared/facebook"
	"shared/graph"
)

func indexFacebookNetwork(session *facebook.Session) error {
	friends, getFriendsErr := session.GetFriends()
	if getFriendsErr != nil {
		return getFriendsErr
	}

	friendNodes, createFriendNodesErr := createNodesForFriends(friends)
	if createFriendNodesErr != nil {
		return createFriendNodesErr
	}

	log.Println(friendNodes)

	return nil
}

func createNodesForFriends(friends []*facebook.Person) ([]*graph.Person, error) {
	createdNodeCh := make(chan *graph.Person)
	createdNodes := []*graph.Person{}

	for _, friend := range friends {
		go func(friend *facebook.Person) {
			person, err := getExistingOrCreateNewPerson(friend.UserId, friend.Name)
			if err != nil {
				//TODO: handle err
			}

			createdNodeCh <- person
		}(friend)
	}

	for {
		node := <-createdNodeCh
		createdNodes = append(createdNodes, node)

		if len(createdNodes) == len(friends) {
			return createdNodes, nil
		}
	}
}

func getExistingOrCreateNewPerson(userId string, name string) (*graph.Person, error) {
	person, getErr := graph.GetPerson(userId)
	if getErr != nil {
		return nil, getErr
	}

	if person == nil {
		//if person is not already in the graph, create a new node for them
		createdPerson, createErr := graph.CreatePerson(userId, name)
		if createErr != nil {
			return nil, createErr
		} else {
			return createdPerson, nil
		}
	}

	return person, nil
}
