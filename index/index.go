package index

import (
	"errors"
	"log"

	"shared/facebook"
	"shared/graph"
)

var maxConcurrentFbRequests int

func InitIndexer(maxConcFbRequests int) {
	maxConcurrentFbRequests = maxConcFbRequests
}

func IndexVolunteer(userId string, accessToken string) (*graph.Volunteer, error) {
	session := facebook.CreateSession(accessToken)

	checkPermissionsErr := checkPermissions(session, userId)
	if checkPermissionsErr != nil {
		return nil, checkPermissionsErr
	}

	userInfo, sessionErr := session.GetInfo()
	if sessionErr != nil {
		return nil, sessionErr
	}

	name := userInfo.Name

	log.Println("[INDEXING STARTED]", name)

	volunteer, volunteerErr := graph.CreateVolunteer(userId, name, accessToken)
	if volunteerErr != nil {
		return volunteer, volunteerErr
	}

	log.Println("[VOLUNTEER CREATED]", name)

	newGraph, newFriendships, newLinks, facebookIndexingErr := loadFacebookNetwork(session)
	if facebookIndexingErr != nil {
		return volunteer, facebookIndexingErr
	}

	commitFriendsErr := commitFacebookNetwork(newGraph)
	if commitFriendsErr != nil {
		return volunteer, commitFriendsErr
	}

	log.Println("[NETWORK LOADED]", name)

	postIndexingErr := indexFacebookPosts(volunteer)
	if postIndexingErr != nil {
		return volunteer, postIndexingErr
	}

	log.Println("[POSTS INDEXED]", name)

	commitRelationshipsErr := commitFacebookRelationships(newFriendships, newLinks)
	if commitRelationshipsErr != nil {
		return volunteer, commitRelationshipsErr
	}

	log.Println("[NETWORK INDEXED]", name)

	iceBucketMappingErr := mapIceBucketChallenge(volunteer)
	if iceBucketMappingErr != nil {
		return volunteer, iceBucketMappingErr
	}

	log.Println("[CHALLENGE MAPPED]", name)

	volunteer.MarkAsIndexed()

	log.Println("[INDEXING COMPLETED]", name)

	return volunteer, nil
}

func checkPermissions(session *facebook.Session, userId string) error {
	permissions, getPermissionsErr := session.GetPermissions(userId)
	if getPermissionsErr != nil {
		return getPermissionsErr
	}

	if !permissions["user_friends"] || !permissions["read_stream"] {
		return errors.New("permissions missing")
	}

	return nil
}
