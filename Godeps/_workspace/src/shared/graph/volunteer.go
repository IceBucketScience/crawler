package graph

import (
	"github.com/jmcvetta/neoism"
	//"log"
)

type Volunteer struct {
	Person

	AccessToken string
	IsIndexed   bool
}

func CreateVolunteer(userId string, name string, accessToken string) (*Volunteer, error) {
	person, personErr := GetPerson(userId)

	if person == nil {
		person, personErr = CreatePerson(userId, name)
	}

	if personErr != nil {
		return nil, personErr
	}

	node := person.node

	//TODO: check for errors on adding labels and properties

	//TODO: due to a bug in neoism, right now I have to reset all properties. Once the bug is fixed,
	//then I can use the following code to set the extra Volunteer properties:
	/*person.SetProperty("accessToken", accessToken)
	person.SetProperty("isIndexed", false)*/

	//Temporary code to set properties until bug is fixed
	props, _ := node.Properties()

	props["accessToken"] = accessToken
	props["isIndexed"] = false
	node.SetProperties(props)
	//End temporary code

	node.AddLabel("Volunteer")

	return &Volunteer{Person: *person, AccessToken: accessToken, IsIndexed: false}, nil
}

func getVolunteerFromNode(node *neoism.Node) (*Volunteer, error) {
	props, err := node.Properties()
	if err != nil {
		return nil, err
	}

	person, getPersonErr := getPersonFromNode(node)
	if getPersonErr != nil {
		return nil, getPersonErr
	}

	return &Volunteer{
		Person:      *person,
		AccessToken: props["accessToken"].(string),
		IsIndexed:   props["isIndexed"].(bool),
	}, nil
}

func (volunteer *Volunteer) MarkAsIndexed() {
	node := volunteer.node
	//TODO: check for errors on adding labels and properties
	//TODO: once bug is fixed, use volunteer.SetProperty
	props, _ := node.Properties()

	props["isIndexed"] = true
	node.SetProperties(props)

	volunteer.IsIndexed = true
}

func FindVolunteer(userId string) (*Volunteer, error) {
	res := []struct {
		V neoism.Node
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement:  `MATCH (v:Volunteer) WHERE v.fbId = {userId} RETURN v`,
		Parameters: neoism.Props{"userId": userId},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	} else if len(res) < 1 {
		return nil, nil
	}

	// adds a db object to each node
	for index, volunteerData := range res {
		volunteerData.V.Db = db
		res[index] = volunteerData
	}

	return getVolunteerFromNode(&res[0].V)
}

func GetVolunteers() (map[string]*Volunteer, error) {
	res := []struct {
		V neoism.Node
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `MATCH (v:Volunteer) RETURN v`,
		Result:    &res,
	})
	if err != nil {
		return nil, err
	}

	volunteers := map[string]*Volunteer{}

	for _, volunteerData := range res {
		volunteerNode := &volunteerData.V
		volunteerNode.Db = db

		volunteer, getVolunteerErr := getVolunteerFromNode(volunteerNode)
		if getVolunteerErr != nil {
			return nil, getVolunteerErr
		}

		volunteers[volunteer.FbId] = volunteer
	}

	return volunteers, nil
}
