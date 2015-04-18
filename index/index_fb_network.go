package index

import (
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

	friendNodes, linkNodesToNetworkErr := linkNewNodesToNetwork(friendNodes)
	if linkNodesToNetworkErr != nil {
		return linkNodesToNetworkErr
	}

	return nil
}

func createNodesForFriends(friends []*facebook.Person) (graph.Graph, error) {
	createdNodeCh := make(chan *graph.Person)
	createdNodes := map[string]*graph.Person{}
	errCh := make(chan error)

	for _, friend := range friends {
		go func(friend *facebook.Person) {
			person, err := getExistingOrCreateNewPerson(friend.UserId, friend.Name)
			if err != nil {
				errCh <- err
			}

			createdNodeCh <- person
		}(friend)
	}

	for len(friends) > 0 {
		select {
		case node := <-createdNodeCh:
			createdNodes[node.FbId] = node
		case err := <-errCh:
			return nil, err
		}

		if len(createdNodes) == len(friends) {
			break
		}
	}

	return createdNodes, nil
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
		}

		return createdPerson, nil
	}

	return person, nil
}

func linkNewNodesToNetwork(newNodes graph.Graph) (graph.Graph, error) {
	volunteers, getVolunteersErr := graph.GetVolunteers()
	if getVolunteersErr != nil {
		return nil, getVolunteersErr
	}

	visitedNodeCh := make(chan *graph.Person)
	visitedNodes := []*graph.Person{}
	errCh := make(chan error)

	for _, node := range newNodes {
		for _, volunteer := range volunteers {
			go func(node *graph.Person, volunteer *graph.Volunteer) {
				linkErr := linkNodeToVolunteer(node, volunteer, newNodes)
				if linkErr != nil {
					errCh <- linkErr
				}

				visitedNodeCh <- node
			}(node, volunteer)
		}
	}

	for len(newNodes) > 0 {
		select {
		case node := <-visitedNodeCh:
			visitedNodes = append(visitedNodes, node)
		case err := <-errCh:
			return nil, err
		}
		if len(visitedNodes) == len(newNodes) {
			break
		}
	}

	return newNodes, nil
}

func linkNodeToVolunteer(node *graph.Person, volunteer *graph.Volunteer, g graph.Graph) error {
	if isLinked, linkCheckErr := node.IsLinkedTo(volunteer); linkCheckErr != nil {
		return linkCheckErr
	} else if !isLinked {
		fbSession := facebook.CreateSession(volunteer.AccessToken)

		friendshipAddedErr := addFriendshipIfFriends(node, volunteer, fbSession)
		if friendshipAddedErr != nil {
			return friendshipAddedErr
		}

		mutualFriendsErr := connectMutualFriends(node, g, fbSession)
		if mutualFriendsErr != nil {
			return mutualFriendsErr
		}

		markErr := node.MarkAsLinkedTo(volunteer)
		if markErr != nil {
			return markErr
		}
	}

	return nil
}

func addFriendshipIfFriends(node *graph.Person, volunteer *graph.Volunteer, fbSession *facebook.Session) error {
	if isFriends, checkFriendshipErr := fbSession.IsFriendsWith(node.FbId); checkFriendshipErr != nil {
		return checkFriendshipErr
	} else if isFriends {
		addFriendshipErr := volunteer.AddFriendshipWith(node.FbId)
		if addFriendshipErr != nil {
			return checkFriendshipErr
		}
	}

	return nil
}

func connectMutualFriends(node *graph.Person, g graph.Graph, fbSession *facebook.Session) error {
	mutualFriends, mutualFriendsErr := fbSession.GetMutualFriendsWith(node.FbId)
	if mutualFriendsErr != nil {
		return mutualFriendsErr
	}

	for _, mutualFriend := range mutualFriends {
		addFriendshipErr := node.AddFriendshipWith(mutualFriend.UserId)
		if addFriendshipErr != nil {
			return addFriendshipErr
		}
	}

	return nil
}
