package capture

import (
	"github.com/nlopes/slack"
	"strconv"
	"time"
)

// FileUploadCaptor captures file uploads recorded by
// invocations of UploadFile
type FileUploadCaptor struct {
	FileUploads []slack.FileUploadParameters
	currentID   int
}

// UploadFile tracks a file upload for post-execution validation
func (f *FileUploadCaptor) UploadFile(params slack.FileUploadParameters) (file *slack.File, err error) {
	f.FileUploads = append(f.FileUploads, params)

	file = new(slack.File)
	file.ID = strconv.Itoa(f.currentID)
	file.Name = params.Filename
	file.Filetype = params.Filetype
	file.Title = params.Title
	file.Created = currentJSONTime()

	// Increment id for the next upload
	f.currentID = f.currentID + 1

	return file, nil
}

// NewFileUploader returns a new FileUploadCaptor with an initialized array of FileUploads
func NewFileUploader() (fileUploadCaptor *FileUploadCaptor) {
	fileUploadCaptor = new(FileUploadCaptor)
	fileUploadCaptor.FileUploads = make([]slack.FileUploadParameters, 0)

	return fileUploadCaptor
}

// currentJSONTime creates a JSONTime value with the current time
func currentJSONTime() (now slack.JSONTime) {
	return slack.JSONTime(time.Now().Unix())
}
