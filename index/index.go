package index

import (
	"errors"
	"log"

	"shared/facebook"
	"shared/graph"
)

func IndexVolunteer(userId string, accessToken string) error {
	session := facebook.CreateSession(accessToken)

	checkPermissionsErr := checkPermissions(session, userId)
	if checkPermissionsErr != nil {
		return checkPermissionsErr
	}

	userInfo, sessionErr := session.GetInfo()
	if sessionErr != nil {
		return sessionErr
	}

	name := userInfo.Name

	log.Println("[INDEXING STARTED]", name)

	volunteer, volunteerErr := graph.CreateVolunteer(userId, name, accessToken)
	if volunteerErr != nil {
		return volunteerErr
	}

	log.Println("[VOLUNTEER CREATED]", name)

	facebookIndexingErr := indexFacebookNetwork(session)
	if facebookIndexingErr != nil {
		return facebookIndexingErr
	}

	log.Println("[NETWORK INDEXED]", name)

	postIndexingErr := indexFacebookPosts(volunteer)
	if postIndexingErr != nil {
		return postIndexingErr
	}

	log.Println("[POSTS INDEXED]", name)

	iceBucketMappingErr := mapIceBucketChallenge(volunteer)
	if iceBucketMappingErr != nil {
		return iceBucketMappingErr
	}

	log.Println("[CHALLENGE MAPPED]", name)

	volunteer.MarkAsIndexed()

	log.Println("[INDEXING COMPLETED]", name)

	return nil
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
