package slackscot_test

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/test/capture"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultUpload(t *testing.T) {
	fileUploadCaptor := capture.NewFileUploader()
	uploader := slackscot.NewFileUploader(fileUploadCaptor)

	uploader.UploadFile(slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"})

	assert.Len(t, fileUploadCaptor.FileUploads, 1)
	assert.Equal(t, slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"}, fileUploadCaptor.FileUploads[0])
}

func TestUploadWithExistingTheadOption(t *testing.T) {
	fileUploadCaptor := capture.NewFileUploader()
	uploader := slackscot.NewFileUploader(fileUploadCaptor)

	uploader.UploadFile(slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"}, slackscot.UploadInThreadOption(&slackscot.IncomingMessage{Msg: slack.Msg{ThreadTimestamp: "100000"}}))

	assert.Len(t, fileUploadCaptor.FileUploads, 1)
	assert.Equal(t, slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look", ThreadTimestamp: "100000"}, fileUploadCaptor.FileUploads[0])
}

func TestUploadWithExistingTheadOptionButNoThreadInMsg(t *testing.T) {
	fileUploadCaptor := capture.NewFileUploader()
	uploader := slackscot.NewFileUploader(fileUploadCaptor)

	uploader.UploadFile(slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"}, slackscot.UploadInThreadOption(&slackscot.IncomingMessage{Msg: slack.Msg{}}))

	assert.Len(t, fileUploadCaptor.FileUploads, 1)
	assert.Equal(t, slack.FileUploadParameters{Filename: "imageOfABirdInATree.png", Filetype: "image/png", Title: "Look"}, fileUploadCaptor.FileUploads[0])
}
