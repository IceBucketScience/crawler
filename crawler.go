package main

import (
	"fmt"
	fb "github.com/huandu/facebook"
	"github.com/jmcvetta/neoism"
)

type Person struct {
	Name, Id string
	friends  []*Person
}

func main() {
	app := fb.New("#", "#")

	session := app.Session("#")

	if err := session.Validate(); err != nil {
		fmt.Println(err)
	}

	session.Version = "v1.0"

	network := getNetwork(session)

	db, err := neoism.Connect("http://localhost:7474/db/data")

	if err != nil {
		fmt.Println("db connect", err)
	}

	insertNetworkIntoDb(db, network)
}

func getNetwork(session *fb.Session) []*Person {
	people := getFriends(session, "me")

	res, err := session.Get("/me", nil)

	if err != nil {
		fmt.Println("getNetwork", err)
	}

	var me Person
	res.Decode(&me)
	people = append(people, &me)

	people = setupFriendships(session, people)

	return people
}

func getFriends(session *fb.Session, id string) []*Person {
	res, err := session.Get("/"+id+"/mutualfriends", nil)

	if err != nil {
		fmt.Println("getFriends", err)
	}

	var friends []*Person
	res.DecodeField("data", &friends)

	return friends
}

func setupFriendships(session *fb.Session, people []*Person) []*Person {
	peopleMap := createPeopleMap(people)
	ch := make(chan *Person)
	linkedPeople := []*Person{}

	for _, person := range people {
		go func(session *fb.Session, person *Person) {
			friends := getFriends(session, person.Id)
			person.friends = createFriendList(friends, peopleMap)
			ch <- person
		}(session, person)
	}

	for {
		person := <-ch
		linkedPeople = append(linkedPeople, person)

		if len(linkedPeople) == len(people) {
			return linkedPeople
		}
	}
}

func createPeopleMap(people []*Person) map[string]*Person {
	peopleMap := make(map[string]*Person)

	for _, person := range people {
		peopleMap[person.Id] = person
	}

	return peopleMap
}

func createFriendList(friends []*Person, peopleMap map[string]*Person) []*Person {
	friendList := []*Person{}

	for _, friend := range friends {
		friendList = append(friendList, peopleMap[friend.Id])
	}

	return friendList
}

func insertNetworkIntoDb(db *neoism.Database, network []*Person) {
	nodes := map[string]*neoism.Node{}

	for _, person := range network {
		nodes[person.Id] = createNodeFromPerson(db, person)
	}

	for _, person := range network {
		for _, friend := range person.friends {
			_, err := nodes[person.Id].Relate("FRIENDS", nodes[friend.Id].Id(), neoism.Props{})

			if err != nil {
				fmt.Println("relate", err)
			}
		}
	}
}

func createNodeFromPerson(db *neoism.Database, person *Person) *neoism.Node {
	node, err := db.CreateNode(neoism.Props{"name": person.Name, "fb_id": person.Id})

	if err != nil {
		fmt.Println("createNode", err)
	}

	node.AddLabel("Person")

	return node
}
