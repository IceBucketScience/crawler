package index

import (
	"shared/graph"
)

func mapIceBucketChallenge(volunteer *graph.Volunteer) error {
	orderedPosts, getOrderedPostsErr := graph.GetPostsInOrder(volunteer.FbId)
	if getOrderedPostsErr != nil {
		return getOrderedPostsErr
	}

	for _, post := range orderedPosts {
		poster, getPosterErr := post.GetPoster()
		if getPosterErr != nil {
			return getPosterErr
		}

		if poster != nil && !poster.HasCompleted {
			poster.AddCompletionTime(post.TimeCreated)
		}

		tagged, getTaggedErr := post.GetTagged()
		if getTaggedErr != nil {
			return getTaggedErr
		}

		handleTaggedErr := handleTagged(post, poster, tagged)
		if handleTaggedErr != nil {
			return handleTaggedErr
		}
	}

	return nil
}

func handleTagged(post *graph.Post, poster *graph.Person, tagged []*graph.Person) error {
	for _, taggedPerson := range tagged {
		if !taggedPerson.HasBeenNominated || int(post.TimeCreated.Unix()) < taggedPerson.TimeNominated {
			addNominationTimeErr := taggedPerson.AddNominationTime(post.TimeCreated)
			if addNominationTimeErr != nil {
				return addNominationTimeErr
			}
		}

		if poster != nil && !taggedPerson.HasCompleted {
			addNominationErr := taggedPerson.AddNomination(poster, post.TimeCreated)
			if addNominationErr != nil {
				return addNominationErr
			}
		}
	}

	return nil
}
