package slackscot

import (
	"github.com/nlopes/slack"
)

// SlackFileUploader is implemented by any value that has the UploadFile method. slack.Client
// implements it. The main purpose remains is a slight decoupling of the slack.Client in order
// for plugins to be able to write cleaner tests more easily.
type SlackFileUploader interface {
	// UploadFile uploads a file to slack. For more info in this API, check
	// https://godoc.org/github.com/nlopes/slack#Client.UploadFile
	UploadFile(params slack.FileUploadParameters) (file *slack.File, err error)
}

// FileUploader is implemented by any value that has the UploadFile method. slack.Client *almost*
// implements it but requires a thin wrapping to do so to handle UploadOption there for
// added extensibility.
// The main purpose remains is a slight decoupling of the slack.Client in order for plugins to
// be able to write cleaner tests more easily.
type FileUploader interface {
	// UploadFile uploads a file to slack. For more info in this API, check
	// https://godoc.org/github.com/nlopes/slack#Client.UploadFile
	UploadFile(params slack.FileUploadParameters, options ...UploadOption) (file *slack.File, err error)
}

// UploadOption defines an option on a FileUploadParameters (i.e. upload on thread)
type UploadOption func(params *slack.FileUploadParameters)

// UploadInThreadOption sets the file upload thread timestamp to an existing thread timestamp if
// the incoming message triggering this is on an existing thread
func UploadInThreadOption(m *IncomingMessage) func(params *slack.FileUploadParameters) {
	return func(p *slack.FileUploadParameters) {
		if threadTimestamp, inThread := resolveThreadTimestamp(&m.Msg); inThread {
			p.ThreadTimestamp = threadTimestamp
		}
	}
}

// DefaultFileUploader holds a bare-bone SlackFileUploader
type DefaultFileUploader struct {
	slackFileUploader SlackFileUploader
}

// NewFileUploader returns a new DefaultFileUploader wrapping a FileUploader
func NewFileUploader(slackFileUploader SlackFileUploader) (fileUploader *DefaultFileUploader) {
	fileUploader = new(DefaultFileUploader)
	fileUploader.slackFileUploader = slackFileUploader

	return fileUploader
}

// UploadFile uploads a file given the slack.FileUploadParameters with the UploadOptions applied to it
func (fileUploader *DefaultFileUploader) UploadFile(params slack.FileUploadParameters, options ...UploadOption) (file *slack.File, err error) {
	for _, opt := range options {
		opt(&params)
	}

	return fileUploader.slackFileUploader.UploadFile(params)
}
