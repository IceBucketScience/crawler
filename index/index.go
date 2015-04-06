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

	log.Println("[INDEXING STARTED]", userInfo.Name)

	volunteer, volunteerErr := graph.CreateVolunteer(userId, userInfo.Name, accessToken)
	if volunteerErr != nil {
		return volunteerErr
	}

	/*facebookIndexingErr := indexFacebookNetwork(session)
	if facebookIndexingErr != nil {
		return facebookIndexingErr
	}

	postIndexingErr := indexFacebookPosts(volunteer)
	if postIndexingErr != nil {
		return postIndexingErr
	}*/

	iceBucketMappingErr := mapIceBucketChallenge(volunteer)
	if iceBucketMappingErr != nil {
		return iceBucketMappingErr
	}

	volunteer.MarkAsIndexed()

	log.Println("[INDEXING COMPLETED]", userInfo.Name)

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
