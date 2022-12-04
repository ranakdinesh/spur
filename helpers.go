package spur

import "os"

// CreateDirIfNotExist function CreateDirStructure will create directory structure of our project
func (s *Spur) CreateDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, mode)
		if err != nil {
			return err
		}

	}
	return nil
}

// CreateFileIfNotExists function CreateFileIfNotExists will create file if it does not exist
func (s *Spur) CreateFileIfNotExists(path string) error {
	var _, err = os.Stat(path)
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			_ = file.Close()

		}(file)

	}

	return nil
}
