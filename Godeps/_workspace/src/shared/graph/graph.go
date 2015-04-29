package graph

import (
	"log"

	"github.com/jmcvetta/neoism"
)

var maxConcurrentDbRequests int
var db *neoism.Database

func InitGraph(dbUrl string, maxConcDbRequests int) {
	maxConcurrentDbRequests = maxConcDbRequests

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

type Transaction struct {
	queries []*neoism.CypherQuery
}

func InitTransaction() *Transaction {
	return &Transaction{queries: []*neoism.CypherQuery{}}
}

func (t *Transaction) AddQuery(q *neoism.CypherQuery) {
	t.queries = append(t.queries, q)
}

func (t *Transaction) Commit() error {
	currTransactionQueries := []*neoism.CypherQuery{}
	log.Println(len(t.queries), "queries to run")
	for _, query := range t.queries {
		if len(currTransactionQueries) == maxConcurrentDbRequests {
			log.Println("len curr txs", len(currTransactionQueries))
			txErr := CommitTransaction(currTransactionQueries)
			if txErr != nil {
				return txErr
			}
			log.Println("finished tx")
			currTransactionQueries = []*neoism.CypherQuery{}
		}

		currTransactionQueries = append(currTransactionQueries, query)
	}
	log.Println("last tx", len(currTransactionQueries))
	//to clean up any extra queries that fell in under the maxConcurrentDbRequests limit
	return CommitTransaction(currTransactionQueries)
}

func CommitTransaction(qs []*neoism.CypherQuery) error {
	tx, txErr := db.Begin(qs)
	if txErr != nil {
		return txErr
	}

	log.Println("tx errs", tx.Errors)

	//return tx.Commit()
	commitErr := tx.Commit()
	if commitErr != nil {
		log.Println("commit errs", tx.Errors)
	}

	return commitErr
}

type Graph map[string]*Person

func (g *Graph) Commit() error {
	tx := InitTransaction()

	for _, person := range *g {
		tx.AddQuery(person.GetQuery())
	}

	return tx.Commit()
}

type RelationshipMap struct {
	Type   string
	relMap map[string]map[string]bool
}

func CreateRelationshipMap(relName string) *RelationshipMap {
	return &RelationshipMap{
		Type:   relName,
		relMap: map[string]map[string]bool{},
	}
}

func (r *RelationshipMap) AddRelationship(sourceId string, targetId string) {
	if r.relMap[sourceId] == nil {
		r.relMap[sourceId] = map[string]bool{}
	}
	r.relMap[sourceId][targetId] = true
}

func (r *RelationshipMap) AddMutualRelationship(id1 string, id2 string) {
	r.AddRelationship(id1, id2)
	r.AddRelationship(id2, id1)
}

func (r *RelationshipMap) RelationshipExists(id1 string, id2 string) bool {
	return r.relMap[id1][id2] || r.relMap[id1][id2]
}

func (r *RelationshipMap) Commit() error {
	tx := InitTransaction()

	for sourceId, relationshipMap := range r.relMap {
		for targetId, _ := range relationshipMap {
			tx.AddQuery(GetCreateRelationshipQuery(r.Type, sourceId, targetId))
		}
	}

	return tx.Commit()
}
