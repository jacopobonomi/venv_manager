package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileSystem interface {
	CreateDir(path string) error
	RemoveAll(path string) error
	Exists(path string) bool
	IsDir(path string) bool
	ReadDir(path string) ([]os.DirEntry, error)
	GetDirSize(path string) (int64, error)
}

type RealFileSystem struct{}

func NewFileSystem() FileSystem {
	return &RealFileSystem{}
}

func (fs *RealFileSystem) CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (fs *RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (fs *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (fs *RealFileSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (fs *RealFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (fs *RealFileSystem) GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	} else if bytes < GB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
}
