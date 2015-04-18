package facebook

import (
	fb "github.com/huandu/facebook"
)

var fbClient *fb.App

func InitFbClient(appId string, appSecret string) {
	fb.Version = "v1.0"

	fbClient = fb.New(appId, appSecret)
	//TODO: check for valid fb credentials after client creation
}

func GetLongTermToken(shortTermToken string) (token string, expires int, err error) {
	return fbClient.ExchangeToken(shortTermToken)
}

func TokenIsValid(token string) (bool, error) {
	res, err := fb.Get("/debug_token", fb.Params{"input_token": token, "access_token": fbClient.AppAccessToken()})
	if err != nil {
		return false, err
	}

	data := res["data"].(map[string]interface{})

	if data == nil || !data["is_valid"].(bool) {
		return false, nil
	}

	return true, nil
}
