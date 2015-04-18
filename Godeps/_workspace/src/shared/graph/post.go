package graph

import (
	"errors"
	"sort"
	"time"

	"github.com/jmcvetta/neoism"
)

type Post struct {
	node *neoism.Node

	FbId, Message string
	TimeCreated   time.Time
}

func CreatePost(postId string, message string, timeCreated time.Time) (*Post, error) {
	res := []struct {
		P neoism.Node
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MERGE (p:Post {fbId: {fbId}, message: {message}, timeCreated: {timeCreated}}) RETURN p
        `,
		Parameters: neoism.Props{"fbId": postId, "message": message, "timeCreated": timeCreated.Unix()},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	} else if len(res) < 1 {
		return nil, nil
	}

	// adds a db object to each node
	for index, postData := range res {
		postData.P.Db = db
		res[index] = postData
	}

	return getPostFromNode(&res[0].P)
}

func (post *Post) AddPoster(fbId string) error {
	res := []struct {
		P neoism.Relationship
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (person:Person), (post:Post)
            WHERE person.fbId = {personId} AND post.fbId = {postId}
            MERGE (person)-[p:POSTED]->(post)
            RETURN p
        `,
		Parameters: neoism.Props{"personId": fbId, "postId": post.FbId},
		Result:     &res,
	})
	if err != nil {
		return err
	} else if len(res) < 1 {
		return errors.New("no new posted relationship created or existing posted relationship found")
	}

	return nil
}

func (post *Post) GetPoster() (*Person, error) {
	res := []struct {
		P neoism.Node
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (p:Person)-[:POSTED]->(:Post {fbId: {postId}}) RETURN p
        `,
		Parameters: neoism.Props{"postId": post.FbId},
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

func (post *Post) GetTagged() ([]*Person, error) {
	res := []struct {
		P neoism.Node
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (:Post {fbId: {postId}})-[:TAGGED]->(p:Person) RETURN p
        `,
		Parameters: neoism.Props{"postId": post.FbId},
		Result:     &res,
	})

	if err != nil {
		return nil, err
	} else if len(res) < 1 {
		return []*Person{}, nil
	}

	tagged := []*Person{}

	for _, personData := range res {
		personNode := personData.P
		personNode.Db = db

		person, getPersonErr := getPersonFromNode(&personNode)
		if getPersonErr != nil {
			return nil, getPersonErr
		}

		tagged = append(tagged, person)
	}

	return tagged, nil
}

func (post *Post) AddTagged(fbId string) error {
	res := []struct {
		T neoism.Relationship
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (post:Post), (person:Person)
            WHERE person.fbId = {personId} AND post.fbId = {postId}
            MERGE (post)-[t:TAGGED]->(person)
            RETURN t
        `,
		Parameters: neoism.Props{"personId": fbId, "postId": post.FbId},
		Result:     &res,
	})
	if err != nil {
		return err
	} else if len(res) < 1 {
		return errors.New("no new tagged relationship created or existing tagged relationship found")
	}

	return nil
}

func getPostFromNode(node *neoism.Node) (*Post, error) {
	props, err := node.Properties()
	if err != nil {
		return nil, err
	}

	return &Post{node: node, FbId: props["fbId"].(string), Message: props["message"].(string), TimeCreated: time.Unix(int64(props["timeCreated"].(float64)), 0)}, nil
}

type OrderedPosts []*Post

func (posts OrderedPosts) Len() int {
	return len(posts)
}

func (posts OrderedPosts) Less(i int, j int) bool {
	return posts[i].TimeCreated.Unix() < posts[j].TimeCreated.Unix()
}

func (posts OrderedPosts) Swap(i int, j int) {
	temp := posts[i]
	posts[i] = posts[j]
	posts[j] = temp
}

func GetPostsInOrder(userId string) ([]*Post, error) {
	res := []struct {
		P           neoism.Node `json:"posts"`
		TimeCreated int         `json:"posts.timeCreated"`
	}{}

	err := db.Cypher(&neoism.CypherQuery{
		Statement: `
            MATCH (:Person {fbId: {fbId}})-[:POSTED]->(posts:Post) RETURN posts, posts.timeCreated
            UNION MATCH (:Person {fbId: {fbId}})<-[:TAGGED]-(posts:Post) RETURN posts, posts.timeCreated
            UNION MATCH (:Person {fbId: {fbId}})-[:FRIENDS]->(friend:Person)-[:POSTED]->(posts:Post) RETURN posts, posts.timeCreated
            UNION MATCH (:Person {fbId: {fbId}})-[:FRIENDS]->(friend:Person)<-[:TAGGED]-(posts:Post) RETURN posts, posts.timeCreated
        `,
		Parameters: neoism.Props{"fbId": userId},
		Result:     &res,
	})
	if err != nil {
		return nil, err
	} else if len(res) < 1 {
		return []*Post{}, nil
	}

	posts := OrderedPosts{}

	for _, postData := range res {
		postNode := &postData.P
		postNode.Db = db

		post, getPostErr := getPostFromNode(postNode)
		if getPostErr != nil {
			return nil, getPostErr
		}

		posts = append(posts, post)
	}

	sort.Sort(posts)

	return posts, nil
}
