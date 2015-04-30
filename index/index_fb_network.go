package index

import (
	"log"

	"shared/facebook"
	"shared/graph"
)

func loadFacebookNetwork(session *facebook.Session) (graph.Graph, *graph.RelationshipMap, *graph.RelationshipMap, error) {
	currNetwork, getNetworkErr := graph.GetNetwork()
	if getNetworkErr != nil {
		return nil, nil, nil, getNetworkErr
	}

	log.Println("CURRENT NETWORK RETRIEVED")

	friends, getFriendsErr := session.GetFriends()
	if getFriendsErr != nil {
		return nil, nil, nil, getFriendsErr
	}

	log.Println("FRIENDS RETRIEVED")

	newGraph := createNodesForFriends(friends, currNetwork)

	log.Println("NODES CREATED FOR FRIENDS", len(newGraph))

	newFriendships, newLinks, linkFriendsErr := linkNodesToNetwork(newGraph, currNetwork)
	if linkFriendsErr != nil {
		return nil, nil, nil, linkFriendsErr
	}

	log.Println("NEW FRIENDS LINKED")

	return newGraph, newFriendships, newLinks, nil
}

func commitFacebookNetwork(volunteer *graph.Volunteer, newGraph graph.Graph) error {
	volunteerFriends := graph.CreateRelationshipMap("FRIENDS")

	for _, friend := range newGraph {
		volunteerFriends.AddRelationship(volunteer.FbId, friend.FbId)
	}

	newGraphCommitErr := newGraph.Commit()
	if newGraphCommitErr != nil {
		return newGraphCommitErr
	}

	commitVolunteerFriendsErr := volunteerFriends.Commit()
	if commitVolunteerFriendsErr != nil {
		return commitVolunteerFriendsErr
	}

	return nil
}

func commitFacebookRelationships(newFriendships *graph.RelationshipMap, newLinks *graph.RelationshipMap) error {
	newFriendshipsErr := newFriendships.Commit()
	if newFriendshipsErr != nil {
		return newFriendshipsErr
	}

	newLinksErr := newLinks.Commit()
	if newLinksErr != nil {
		return newLinksErr
	}

	return nil
}

func createNodesForFriends(friends []*facebook.Person, totalGraph graph.Graph) graph.Graph {
	newGraph := graph.Graph{}

	for _, friend := range friends {
		newPerson := graph.CreatePersonNode(friend.UserId, friend.Name)
		newGraph[friend.UserId] = newPerson

		if _, friendExists := totalGraph[friend.UserId]; !friendExists {
			totalGraph[friend.UserId] = newPerson
		}
	}

	return newGraph
}

/*func createNodesForFriends(friends []*facebook.Person) (graph.Graph, error) {
	createdNodeCh := make(chan *graph.Person, len(friends))
	createdNodes := map[string]*graph.Person{}
	errCh := make(chan error, 1)

	for _, friend := range friends {
		//go func(friend *facebook.Person) {
		person, err := getExistingOrCreateNewPerson(friend.UserId, friend.Name)
		if err != nil {
			errCh <- err
		}

		createdNodeCh <- person
		//}(friend)
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
}*/

/*func getExistingOrCreateNewPerson(userId string, name string) (*graph.Person, error) {
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
}*/

func linkNodesToNetwork(newPeople graph.Graph, totalGraph graph.Graph) (*graph.RelationshipMap, *graph.RelationshipMap, error) {
	volunteers, getVolunteerErr := graph.GetVolunteers()
	if getVolunteerErr != nil {
		return nil, nil, getVolunteerErr
	}

	linkedMap, getLinkedMapErr := graph.GetLinked()
	if getLinkedMapErr != nil {
		return nil, nil, getLinkedMapErr
	}

	newLinks := graph.CreateRelationshipMap("LINKED")
	newFriendships := graph.CreateRelationshipMap("FRIENDS")

	visitedFriendCh := make(chan *graph.Person, len(newPeople)*len(volunteers))
	visitedFriends := []*graph.Person{}
	errCh := make(chan error)

	throttle := make(chan bool, maxConcurrentFbRequests)

	for _, friend := range newPeople {
		for _, volunteer := range volunteers {
			throttle <- true

			go func(friend *graph.Person, volunteer *graph.Volunteer) {
				linkErr := linkFriendToVolunteer(friend, volunteer, totalGraph, linkedMap, newFriendships, newLinks)
				if linkErr != nil {
					errCh <- linkErr
				}

				visitedFriendCh <- friend

				<-throttle
			}(friend, volunteer)
		}
	}

	for len(newPeople) > 0 {
		select {
		case friend := <-visitedFriendCh:
			visitedFriends = append(visitedFriends, friend)
		case err := <-errCh:
			//return nil, nil, err
			log.Println("[INDEXING ERROR]", err)
		}

		if len(visitedFriends)/len(volunteers) == len(newPeople) {
			break
		}
	}

	return newFriendships, newLinks, nil
}

/*func linkNewNodesToNetwork(newNodes graph.Graph) (graph.Graph, error) {
	volunteers, getVolunteersErr := graph.GetVolunteers()
	if getVolunteersErr != nil {
		return nil, getVolunteersErr
	}

	visitedNodeCh := make(chan *graph.Person, len(newNodes)*len(volunteers))
	visitedNodes := []*graph.Person{}
	errCh := make(chan error)

	throttle := make(chan int, maxConcurrentDbRequests)

	for _, node := range newNodes {
		for _, volunteer := range volunteers {
			throttle <- 1

			go func(node *graph.Person, volunteer *graph.Volunteer, throttle chan int) {
				linkErr := linkNodeToVolunteer(node, volunteer, newNodes)
				if linkErr != nil {
					errCh <- linkErr
				}

				visitedNodeCh <- node

				<-throttle
			}(node, volunteer, throttle)
		}
	}

	for len(newNodes) > 0 {
		select {
		case node := <-visitedNodeCh:
			visitedNodes = append(visitedNodes, node)
		case err := <-errCh:
			return nil, err
		}

		log.Println("currently done linking", len(visitedNodes), "out of", len(newNodes)*len(volunteers), "; stats:", len(newNodes), len(volunteers))

		if len(visitedNodes)/len(volunteers) == len(newNodes) {
			break
		}
	}

	return newNodes, nil
}*/

func linkFriendToVolunteer(
	friend *graph.Person,
	volunteer *graph.Volunteer,
	totalGraph graph.Graph,
	linkedMap *graph.RelationshipMap,
	newFriendships *graph.RelationshipMap,
	newLinks *graph.RelationshipMap) error {
	if !linkedMap.RelationshipExists(friend.FbId, volunteer.FbId) {
		fbSession := facebook.CreateSession(volunteer.AccessToken)

		friendshipAddedErr := addFriendshipIfFriends(friend, volunteer, fbSession, newFriendships)
		if friendshipAddedErr != nil {
			return friendshipAddedErr
		}

		mutualFriendsErr := connectMutualFriends(friend, fbSession, newFriendships)
		if mutualFriendsErr != nil {
			return mutualFriendsErr
		}

		newLinks.AddRelationship(friend.FbId, volunteer.FbId)
	}

	return nil
}

/*func linkNodeToVolunteer(node *graph.Person, volunteer *graph.Volunteer, g graph.Graph) error {
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
}*/

func addFriendshipIfFriends(
	friend *graph.Person,
	volunteer *graph.Volunteer,
	fbSession *facebook.Session,
	newFriendships *graph.RelationshipMap) error {
	if isFriends, checkFriendshipErr := fbSession.IsFriendsWith(friend.FbId); checkFriendshipErr != nil {
		return checkFriendshipErr
	} else if isFriends {
		newFriendships.AddRelationship(friend.FbId, volunteer.FbId)
	}

	return nil
}

/*func addFriendshipIfFriends(node *graph.Person, volunteer *graph.Volunteer, fbSession *facebook.Session) error {
	if isFriends, checkFriendshipErr := fbSession.IsFriendsWith(node.FbId); checkFriendshipErr != nil {
		return checkFriendshipErr
	} else if isFriends {
		addFriendshipErr := volunteer.AddFriendshipWith(node.FbId)
		if addFriendshipErr != nil {
			return checkFriendshipErr
		}
	}

	return nil
}*/

func connectMutualFriends(friend *graph.Person, fbSession *facebook.Session, newFriendships *graph.RelationshipMap) error {
	mutualFriends, mutualFriendsErr := fbSession.GetMutualFriendsWith(friend.FbId)
	if mutualFriendsErr != nil {
		return mutualFriendsErr
	}

	for _, mutualFriend := range mutualFriends {
		newFriendships.AddRelationship(friend.FbId, mutualFriend.UserId)
	}

	return nil
}

/*func connectMutualFriends(node *graph.Person, g graph.Graph, fbSession *facebook.Session) error {
	mutualFriends, mutualFriendsErr := fbSession.GetMutualFriendsWith(node.FbId)
	if mutualFriendsErr != nil {
		return mutualFriendsErr
	}

	connectedFriends := []*facebook.Person{}
	connectedFriendsCh := make(chan *facebook.Person, len(mutualFriends))
	errCh := make(chan error)
	throttle := make(chan int, maxConcurrentDbRequests)

	for _, mutualFriend := range mutualFriends {
		throttle <- 1

		go func(mutualFriend *facebook.Person, throttle chan int) {
			addFriendshipErr := node.AddFriendshipWith(mutualFriend.UserId)
			if addFriendshipErr != nil {
				errCh <- addFriendshipErr
			}

			connectedFriendsCh <- mutualFriend

			<-throttle
		}(mutualFriend, throttle)
	}

	for len(mutualFriends) > 0 {
		select {
		case connectedFriend := <-connectedFriendsCh:
			connectedFriends = append(connectedFriends, connectedFriend)
		case err := <-errCh:
			return err
		}

		log.Println("mutual friends indexed for", node.FbId, " ", len(connectedFriends), "out of", len(mutualFriends))

		if len(connectedFriends) == len(mutualFriends) {
			break
		}
	}

	return nil
}*/
