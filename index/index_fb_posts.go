package index

import (
	"strings"
	"time"

	"shared/facebook"
	"shared/graph"
)

func indexFacebookPosts(volunteer *graph.Volunteer) error {
	g, getFriendsErr := volunteer.GetFriends()
	if getFriendsErr != nil {
		return getFriendsErr
	}

	//adds the volunteer to the network so its posts can also get indexed
	g[volunteer.FbId] = &volunteer.Person

	session := facebook.CreateSession(volunteer.AccessToken)

	indexedPeople := []*graph.Person{}
	indexedPeopleCh := make(chan *graph.Person, len(g))
	errCh := make(chan error)

	throttle := make(chan int, maxConcurrentDbRequests)

	for _, person := range g {
		throttle <- 1

		go func(person *graph.Person, throttle chan int) {
			postIndexingErr := indexPostsOf(person, session)
			if postIndexingErr != nil {
				errCh <- postIndexingErr
			}

			indexedPeopleCh <- person

			<-throttle
		}(person, throttle)
	}

	for len(g) > 0 {
		select {
		case indexedPerson := <-indexedPeopleCh:
			indexedPeople = append(indexedPeople, indexedPerson)
		case err := <-errCh:
			return err
		}

		if len(indexedPeople) == len(g) {
			return nil
		}
	}

	return nil
}

func indexPostsOf(person *graph.Person, session *facebook.Session) error {
	rawPosts, getPostsErr := session.GetUsersPostsBetween(
		person.FbId,
		time.Date(2014, 5, 15, 0, 0, 0, 0, time.UTC),   //beginning May 15, 2014
		time.Date(2014, 10, 1, 11, 59, 0, 0, time.UTC)) //end October 1, 2014
	if getPostsErr != nil {
		return getPostsErr
	}

	indexedPosts := []*facebook.Post{}
	indexedPostsCh := make(chan *facebook.Post, len(rawPosts))
	errCh := make(chan error)

	for _, post := range rawPosts {
		//go func(post *facebook.Post) {
		if isIceBucketChallengePost(post) {
			_, err := getExistingOrCreateNewPost(post)
			if err != nil {
				errCh <- err
			}
		}

		indexedPostsCh <- post
		//}(post)
	}

	for len(rawPosts) > 0 {
		select {
		case indexedPost := <-indexedPostsCh:
			indexedPosts = append(indexedPosts, indexedPost)
		case err := <-errCh:
			return err
		}

		if len(indexedPosts) == len(rawPosts) {
			return nil
		}
	}

	return nil
}

func isIceBucketChallengePost(postData *facebook.Post) bool {
	if strings.Contains(postData.Message, "ALS") {
		return true
	}

	normalizedMessage := strings.ToUpper(postData.Message)
	keywords := []string{
		"#STRIKEOUTALS",
		"ICE WATER CHALLENGE",
		"ICE BUCKET CHALLENGE",
		"ICEBUCKETCHALLENGE",
		"NOMINAT",
		"CHALLENG",
		"24 HOURS",
	}

	for _, keyword := range keywords {
		if strings.Contains(normalizedMessage, keyword) {
			return true
		}
	}

	return false
}

func getExistingOrCreateNewPost(postData *facebook.Post) (*graph.Post, error) {
	post, err := graph.CreatePost(postData.Id, postData.Message, postData.CreatedTime)
	if err != nil {
		return nil, err
	}

	addPosterErr := addPosterToPost(postData.Poster.UserId, post)
	if addPosterErr != nil {
		return nil, addPosterErr
	}

	if postData.Tagged != nil {
		addTagsErr := addTaggedToPost(postData.Tagged, post)
		if addTagsErr != nil {
			return nil, addTagsErr
		}
	}

	return post, nil
}

func addPosterToPost(posterId string, post *graph.Post) error {
	poster, getPosterErr := graph.GetPerson(posterId)
	if getPosterErr != nil {
		return getPosterErr
	}

	if poster != nil {
		addPosterErr := post.AddPoster(poster.FbId)
		if addPosterErr != nil {
			return addPosterErr
		}
	}

	return nil
}

func addTaggedToPost(peopleToTag []*facebook.Person, post *graph.Post) error {
	taggedPeople := []*graph.Person{}
	taggedPeopleCh := make(chan *graph.Person, len(peopleToTag))
	errCh := make(chan error)

	for _, person := range peopleToTag {
		//go func(person *facebook.Person) {
		tagged, err := addTaggedPersonToPost(person.UserId, post)
		if err != nil {
			errCh <- err
		}

		taggedPeopleCh <- tagged
		//}(person)
	}

	for len(peopleToTag) > 0 {
		select {
		case taggedPerson := <-taggedPeopleCh:
			taggedPeople = append(taggedPeople, taggedPerson)
		case err := <-errCh:
			return err
		}
		if len(taggedPeople) == len(peopleToTag) {
			return nil
		}
	}

	return nil
}

func addTaggedPersonToPost(taggedId string, post *graph.Post) (*graph.Person, error) {
	tagged, getTaggedErr := graph.GetPerson(taggedId)
	if getTaggedErr != nil {
		return nil, getTaggedErr
	}

	if tagged != nil {
		addTaggedErr := post.AddTagged(tagged.FbId)
		if addTaggedErr != nil {
			return nil, addTaggedErr
		}
	}

	return tagged, nil
}
