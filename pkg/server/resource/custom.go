package resource

import (
	"github.com/sst/sst/v3/pkg/project"
)

type Custom struct {
	project *project.Project
}

type CustomDownloadInput struct {
	Hash string `json:"hash"`
}

type CustomDownloadOutput struct {
	File string `json:"file"`
}

func (r *Custom) Download(input *CustomDownloadInput, output *CustomDownloadOutput) error {
	file, err := r.project.DownloadResource(input.Hash)
	if err != nil {
		return err
	}
	output.File = file
	return nil
}
