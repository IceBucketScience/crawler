package graph

import (
	"errors"
	"strconv"
	"time"

	"github.com/jmcvetta/neoism"
)

type Person struct {
	node *neoism.Node

	FbId, Name                     string
	HasBeenNominated, HasCompleted bool
	TimeNominated                  int
	TimeCompleted                  int
}

func CreatePersonNode(userId string, name string) *Person {
	return &Person{FbId: userId, Name: name}
}

func GetNetwork() (Graph, error) {
	res := []struct {
		N             neoism.Node
		FbId          string `json:"fbId"`
		Name          string `json:"name"`
		TimeNominated int    `json:"timeNominated"`
		TimeCompleted int    `json:"timeCompleted"`
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (p:Person) RETURN p, 
            	p.fbId AS fbId, 
            	p.name AS name, 
            	p.timeNominated AS timeNominated, 
            	p.timeCompleted AS timeCompleted
        `,
		Result: &res,
	})
	if err != nil {
		return nil, err
	}

	network := Graph{}

	for _, personData := range res {
		personNode := personData.N
		personNode.Db = db

		person := &Person{
			node:          &personNode,
			FbId:          personData.FbId,
			Name:          personData.Name,
			TimeNominated: personData.TimeNominated,
			TimeCompleted: personData.TimeCompleted,
		}

		network[person.FbId] = person
	}

	return network, nil
}

func GetLinked() (*RelationshipMap, error) {
	res := []struct {
		PersonId    string `json:"personId"`
		VolunteerId string `json:"volunteerId"`
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (p:Person)-[:LINKED]->(v:Volunteer) RETURN p.fbId AS personId, v.fbId AS volunteerId
        `,
		Result: &res,
	})
	if err != nil {
		return nil, err
	}

	linkedMap := CreateRelationshipMap("LINKED")

	for _, link := range res {
		linkedMap.AddMutualRelationship(link.PersonId, link.VolunteerId)
	}

	return linkedMap, nil
}

func CreatePerson(userId string, name string) (*Person, error) {
	node, err := db.CreateNode(neoism.Props{"fbId": userId, "name": name})
	if err != nil {
		return nil, err
	}

	//TODO: check for errors on adding labels

	node.AddLabel("Person")

	return &Person{node: node, FbId: userId, Name: name}, nil
}

func (person *Person) GetQuery() *neoism.CypherQuery {
	return &neoism.CypherQuery{
		Statement: `
			MERGE (p:Person {fbId: {fbId}, name: {name}}) RETURN p
		`,
		Parameters: neoism.Props{"fbId": person.FbId, "name": person.Name},
	}
}

func GetCreateRelationshipQuery(relName string, p1Id string, p2Id string) *neoism.CypherQuery {
	return &neoism.CypherQuery{
		Statement: `
            MATCH (p1:Person), (p2:Person)
            WHERE p1.fbId = {p1Id} AND p2.fbId = {p2Id}
            MERGE (p1)-[f:` + relName + `]-(p2)
            RETURN f
        `,
		Parameters: neoism.Props{"p1Id": p1Id, "p2Id": p2Id},
	}
}

func (person *Person) addRelationshipTo(fbId string, relName string) error {
	res := []struct {
		F neoism.Relationship
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (p1:Person), (p2:Person)
            WHERE p1.fbId = {p1Id} AND p2.fbId = {p2Id}
            MERGE (p1)-[f:` + relName + `]-(p2)
            RETURN f
        `,
		Parameters: neoism.Props{"p1Id": person.FbId, "p2Id": fbId},
		Result:     &res,
	})
	if err != nil {
		return err
	} else if len(res) < 1 {
		return errors.New("no new relationship created or existing relationship found")
	}

	return nil
}

func (person *Person) AddFriendshipWith(friendId string) error {
	return person.addRelationshipTo(friendId, "FRIENDS")
}

func (person *Person) MarkAsLinkedTo(volunteer *Volunteer) error {
	return person.addRelationshipTo(volunteer.FbId, "LINKED")
}

func relationshipExists(person1Id string, person2Id string, relName string) (bool, error) {
	res := []struct {
		L neoism.Relationship
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (p1:Person)-[l:` + relName + `]-(p2:Person)
            WHERE p1.fbId = {p1Id} AND p2.fbId = {p2Id}
            RETURN l
        `,
		Parameters: neoism.Props{"p1Id": person1Id, "p2Id": person2Id},
		Result:     &res,
	})
	if err != nil {
		return false, err
	} else if len(res) < 1 {
		return false, nil
	}

	return true, nil
}

func (person *Person) IsLinkedTo(volunteer *Volunteer) (bool, error) {
	return relationshipExists(person.FbId, volunteer.FbId, "LINKED")
}

//TODO: switch return val to pointer
func (person *Person) GetFriends() (Graph, error) {
	res := []struct {
		P neoism.Node `json:"friends"`
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement:  `MATCH (p:Person)-[:FRIENDS]-(friends) WHERE p.fbId = {fbId} RETURN friends`,
		Parameters: neoism.Props{"fbId": person.FbId},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	}

	friends := Graph{}

	for _, personData := range res {
		personNode := personData.P
		personNode.Db = db

		person, getPersonErr := getPersonFromNode(&personNode)
		if getPersonErr != nil {
			return nil, getPersonErr
		}

		friends[person.FbId] = person
	}

	return friends, nil
}

func NominationExists(person1Id string, person2Id string) (bool, error) {
	return relationshipExists(person1Id, person2Id, "NOMINATED")
}

func (person *Person) AddNomination(nominatedBy *Person, nominationTime time.Time) error {
	res := []struct {
		N neoism.Relationship
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (p1:Person {fbId: {p1Id}}), (p2:Person {fbId: {p2Id}})
            MERGE (p1)-[n:NOMINATED {timeNominated: {timeNominated}}]->(p2)
            RETURN n
        `,
		Parameters: neoism.Props{"p1Id": nominatedBy.FbId, "p2Id": person.FbId, "timeNominated": nominationTime.Unix()},
		Result:     &res,
	})
	if err != nil {
		return err
	} else if len(res) < 1 {
		return errors.New("no new relationship created or existing relationship found")
	}

	return nil
}

func (person *Person) AddNominationTime(nominationTime time.Time) error {
	person.HasBeenNominated = true

	props, _ := person.node.Properties()
	props["timeNominated"] = nominationTime.Unix()
	return person.node.SetProperties(props)
}

func (person *Person) AddCompletionTime(completionTime time.Time) error {
	person.HasBeenNominated = true
	person.HasCompleted = true

	props, _ := person.node.Properties()
	if props["timeNominated"] == nil {
		props["timeNominated"] = completionTime.Unix()
	}
	props["timeCompleted"] = completionTime.Unix()
	return person.node.SetProperties(props)
}

type Friendship struct {
	rel      *neoism.Relationship
	Id       int
	SourceId string
	TargetId string
}

type FriendshipRes struct {
	Id int `json:"id"`
}

func (friendship *Friendship) GetRelationshipId() int {
	//return friendship.rel.Id()
	return friendship.Id
}

func GetFriendshipsInNetwork(personId string) ([]*Friendship, error) {
	res := []struct {
		SourceId string              `json:"sourceId"`
		TargetId string              `json:"targetId"`
		F        neoism.Relationship `json:"f"`
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (v:Person {fbId: {fbId}})-[f:FRIENDS]-(friend:Person) 
                RETURN v.fbId AS sourceId, f, friend.fbId AS targetId
            UNION MATCH (v:Person {fbId: {fbId}})-[:FRIENDS]-(p:Person)-[f:FRIENDS]-(friend:Person) 
                RETURN p.fbId AS sourceId, f, friend.fbId AS targetId
        `,
		Parameters: neoism.Props{"fbId": personId},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	}

	friendships := []*Friendship{}

	for _, friendshipData := range res {
		friendshipRel := &friendshipData.F
		friendshipRel.Db = db

		friendship := &Friendship{Id: friendshipRel.Id(), SourceId: friendshipData.SourceId, TargetId: friendshipData.TargetId, rel: friendshipRel}

		friendships = append(friendships, friendship)
	}

	return friendships, nil
}

func GetFriendshipIdsWithNominations(personId string) ([]string, error) {
	res := []struct {
		Id int `json:"id"`
	}{}

	//TODO: fix query so that edges aren't returned that aren't friendships within person's network
	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
	          MATCH (v:Person {fbId: {personId}})-[:FRIENDS]-(p:Person)-[:NOMINATED]->(friends)-[f:FRIENDS]-(p)
	              RETURN Id(f) AS id
	          UNION MATCH (v:Person {fbId: {personId}})-[:NOMINATED]->(friends)-[f:FRIENDS]-(v)
	              RETURN Id(f) AS id
	      `,
		Parameters: neoism.Props{"personId": personId},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	}

	friendshipIds := []string{}

	for _, friendshipData := range res {
		friendshipIds = append(friendshipIds, strconv.Itoa(friendshipData.Id))
	}

	return friendshipIds, nil
}

func getPersonFromNode(node *neoism.Node) (*Person, error) {
	props, err := node.Properties()
	if err != nil {
		return nil, err
	}

	var timeNominated int
	if props["timeNominated"] != nil {
		timeNominated = int(props["timeNominated"].(float64))
	}

	var timeCompleted int
	if props["timeCompleted"] != nil {
		timeCompleted = int(props["timeCompleted"].(float64))
	}

	return &Person{
		node:             node,
		FbId:             props["fbId"].(string),
		Name:             props["name"].(string),
		HasBeenNominated: props["timeNominated"] != nil,
		TimeNominated:    timeNominated,
		HasCompleted:     props["timeCompleted"] != nil,
		TimeCompleted:    timeCompleted,
	}, nil
}

func GetPerson(userId string) (*Person, error) {
	res := []struct {
		P neoism.Node
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement:  `MATCH (p:Person) WHERE p.fbId = {userId} RETURN p`,
		Parameters: neoism.Props{"userId": userId},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	} else if len(res) < 1 {
		return nil, nil
	}

	// adds a db object to each node
	for index, personData := range res {
		personData.P.Db = db
		res[index] = personData
	}

	return getPersonFromNode(&res[0].P)
}
