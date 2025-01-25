package utils

import (
	"os"
)

type FileSystem interface {
	CreateDir(path string) error
	RemoveAll(path string) error
	Exists(path string) bool
	IsDir(path string) bool
	ReadDir(path string) ([]os.DirEntry, error)
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
