package index

import (
	"shared/facebook"
	"shared/graph"
)

func IndexVolunteer(userId string, accessToken string) error {
	session := facebook.CreateSession(accessToken)

	userInfo, sessionErr := session.GetInfo()
	if sessionErr != nil {
		return sessionErr
	}

	volunteer, volunteerErr := graph.CreateVolunteer(userId, userInfo.Name, accessToken)
	if volunteerErr != nil {
		return volunteerErr
	}

	facebookIndexingErr := indexFacebookNetwork(session)
	if facebookIndexingErr != nil {
		return facebookIndexingErr
	}

	volunteer.MarkAsIndexed()

	return nil
}
