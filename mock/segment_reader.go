package main

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"os"
)

type SegmentReader interface {
	Read(fileName string) ([]MockSegment, error)
}

type fileSegmentReader struct {
}

func NewFileSegmentReader() SegmentReader {
	return &fileSegmentReader{}
}

func (f *fileSegmentReader) Read(fileName string) ([]MockSegment, error) {
	var segments []MockSegment
	s, err := f.readJSONFile(fileName)
	if err != nil {
		return nil, err
	}
	segments = append(segments, s...)

	return segments, nil
}

func (f *fileSegmentReader) readJSONFile(fileName string) ([]MockSegment, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to close file")
		}
	}(file)

	var events []MockSegment
	if err := json.NewDecoder(file).Decode(&events); err != nil {
		log.Err(err).Msg("Failed to decode JSON")
		return nil, err
	}

	return events, nil
}
