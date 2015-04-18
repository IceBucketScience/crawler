package facebook

import (
	"encoding/json"
	"strconv"
	"time"

	fb "github.com/huandu/facebook"
)

type Session struct {
	fbSession *fb.Session
}

func CreateSession(accessToken string) *Session {
	return &Session{fbSession: fbClient.Session(accessToken)}
}

func (session *Session) GetPermissions(userId string) (map[string]bool, error) {
	res, fbErr := session.fbSession.Get("/"+userId+"/permissions", nil)
	if fbErr != nil {
		return nil, fbErr
	}

	rawPermissions := res["data"].([]interface{})[0].(map[string]interface{})
	permissions := map[string]bool{}

	for permissionName, permissionStatus := range rawPermissions {
		if permissionStatus.(json.Number) == "1" {
			permissions[permissionName] = true
		}
	}

	return permissions, nil
}

func (session *Session) GetInfo() (*Person, error) {
	res, fbErr := session.fbSession.Get("/me", nil)
	if fbErr != nil {
		return nil, fbErr
	}

	var info Person
	decodeErr := res.Decode(&info)
	if decodeErr != nil {
		return nil, decodeErr
	}

	return &info, nil
}

type Person struct {
	UserId string `facebook:"id"`
	Name   string
}

func (session *Session) GetFriends() ([]*Person, error) {
	res, fbErr := session.fbSession.Get("/me/friends", nil)
	if fbErr != nil {
		return nil, fbErr
	}

	var friendsList []*Person
	decodeErr := res.DecodeField("data", &friendsList)
	if decodeErr != nil {
		return nil, decodeErr
	}

	return friendsList, nil
}

func (session *Session) IsFriendsWith(userId string) (bool, error) {
	res, fbErr := session.fbSession.Get("/me/friends/"+userId, nil)
	if fbErr != nil {
		return false, fbErr
	}

	var friend []*Person
	decodeErr := res.DecodeField("data", &friend)
	if decodeErr != nil {
		return false, decodeErr
	}

	return len(friend) > 0, nil
}

func (session *Session) GetMutualFriendsWith(userId string) ([]*Person, error) {
	res, fbErr := session.fbSession.Get("/"+userId+"/mutualfriends", nil)
	if fbErr != nil {
		return nil, fbErr
	}

	var mutualFriends []*Person
	decodeErr := res.DecodeField("data", &mutualFriends)
	if decodeErr != nil {
		return nil, decodeErr
	}

	return mutualFriends, nil
}

type RawPost struct {
	Id            string  `facebook:"id"`
	Message       string  `facebook:"message"`
	Poster        *Person `facebook:"from"`
	To            *To     `facebook:"to"`
	WithTagged    *Tags   `facebook:"with_tags"`
	MessageTagged *Tags   `facebook:"message_tags"`
	CreatedTime   string  `facebook:"created_time"`
}

type To struct {
	Data []*Person `facebook:"data"`
}

type Tags struct {
	Data []*Person `facebook:"data"`
}

type Feed []*RawPost

func toFbTimeString(t time.Time) string {
	return "" + strconv.Itoa(t.Year()) + "-" + strconv.Itoa(int(t.Month())) + "-" + strconv.Itoa(t.Day()) + "T" + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + ":" + strconv.Itoa(t.Second())
}

func (session *Session) GetUsersPostsBetween(userId string, startTime time.Time, endTime time.Time) ([]*Post, error) {
	var feed Feed
	res, getPostsErr := session.fbSession.Get("/"+userId+"/feed", fb.Params{
		"since": toFbTimeString(startTime),
		"until": toFbTimeString(endTime),
	})
	if getPostsErr != nil {
		return nil, getPostsErr
	}

	decodeErr := res.DecodeField("data", &feed)
	if decodeErr != nil {
		return nil, decodeErr
	}

	posts := []*Post{}

	for _, rawPost := range feed {
		posts = append(posts, rawPost.ConvertToPost())
	}

	return posts, nil
}

type Post struct {
	Id          string
	Message     string
	Poster      *Person
	Tagged      []*Person
	CreatedTime time.Time
}

func (rawPost *RawPost) ConvertToPost() *Post {
	return &Post{
		Id:      rawPost.Id,
		Message: rawPost.Message,
		Poster:  rawPost.Poster,
		Tagged:  consolidateTags(rawPost),
		//rounding in time is to account for inconsistent timestamps returned by Facebook
		CreatedTime: fromFbTimeStringToTime(rawPost.CreatedTime).Round(time.Minute * 5),
	}
}

func fromFbTimeStringToTime(timeStr string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05-0700", timeStr)
	return t
}

func consolidateTags(rawPost *RawPost) []*Person {
	tagged := []*Person{}
	alreadyTagged := map[string]bool{}

	if rawPost.To != nil {
		for _, person := range rawPost.To.Data {
			if !alreadyTagged[person.UserId] {
				tagged = append(tagged, person)
				alreadyTagged[person.UserId] = true
			}
		}
	}

	if rawPost.WithTagged != nil {
		for _, person := range rawPost.WithTagged.Data {
			if !alreadyTagged[person.UserId] {
				tagged = append(tagged, person)
				alreadyTagged[person.UserId] = true
			}
		}
	}

	if rawPost.MessageTagged != nil {
		for _, person := range rawPost.MessageTagged.Data {
			if !alreadyTagged[person.UserId] {
				tagged = append(tagged, person)
				alreadyTagged[person.UserId] = true
			}
		}
	}

	return tagged
}
