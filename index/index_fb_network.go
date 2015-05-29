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
