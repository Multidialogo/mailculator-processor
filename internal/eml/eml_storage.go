package eml

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type EMLStorage struct {
	basePath string
}

func NewEMLStorage(basePath string) *EMLStorage {
	return &EMLStorage{basePath}
}

func (s *EMLStorage) Store(emlData EML) (string, error) {
	year, month, _ := time.Now().Date()
	dirPath := filepath.Join(s.basePath, fmt.Sprintf("%v/%v", year, month))
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	path := filepath.Join(dirPath, fmt.Sprintf("%s.EML", emlData.MessageId))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	emlWriter := new(Writer)
	if err := emlWriter.Write(file, emlData); err != nil {
		return "", err
	}

	return path, nil
}
