package graph

import (
	"github.com/jmcvetta/neoism"
)

type SurveyResponse struct {
	node *neoism.Node

	FbName          string
	DidDonate       bool
	IsFirstDonation bool
	DonationDate    int
}

func CreateSurveyResponse(fbName string, didDonate bool, isFirstDonation bool, donationDate int) (*SurveyResponse, error) {
	node, err := db.CreateNode(neoism.Props{
		"fbName":          fbName,
		"didDonate":       didDonate,
		"isFirstDonation": isFirstDonation,
		"donationDate":    donationDate,
	})
	if err != nil {
		return nil, err
	}

	//TODO: check for errors on adding labels

	node.AddLabel("SurveyResponse")

	return &SurveyResponse{
		node:            node,
		FbName:          fbName,
		DidDonate:       didDonate,
		IsFirstDonation: isFirstDonation,
		DonationDate:    donationDate,
	}, nil
}
