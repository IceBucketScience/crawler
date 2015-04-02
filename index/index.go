package index

import (
	"log"

	"shared/facebook"
	"shared/graph"
)

func IndexVolunteer(userId string, accessToken string) error {
	session := facebook.CreateSession(accessToken)

	userInfo, sessionErr := session.GetInfo()
	if sessionErr != nil {
		return sessionErr
	}

	log.Println("[INDEXING STARTED] ", userInfo.Name)

	volunteer, volunteerErr := graph.CreateVolunteer(userId, userInfo.Name, accessToken)
	if volunteerErr != nil {
		return volunteerErr
	}

	facebookIndexingErr := indexFacebookNetwork(session)
	if facebookIndexingErr != nil {
		return facebookIndexingErr
	}

	volunteer.MarkAsIndexed()

	log.Println("[INDEXING COMPLETED] ", userInfo.Name)

	return nil
}
