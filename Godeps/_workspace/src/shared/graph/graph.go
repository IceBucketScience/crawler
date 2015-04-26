package graph

import (
	"log"

	"github.com/jmcvetta/neoism"
)

var db *neoism.Database

func InitGraph(dbUrl string) {
	dbObj, err := neoism.Connect(dbUrl)
	if err != nil {
		//TODO: switch to returning err
		log.Fatalln(err)
	}

	db = dbObj

	//constraint creation to avoid duplicate Posts from MERGE calls caused by race condition
	res := []struct{}{}
	constraintErr := db.Cypher(&neoism.CypherQuery{
		Statement: `CREATE CONSTRAINT ON (p:Post) ASSERT p.fbId IS UNIQUE`,
		Result:    &res,
	})
	if constraintErr != nil {
		log.Println(constraintErr)
	}
}

type Graph map[string]*Person
