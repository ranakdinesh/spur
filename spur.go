package spur

import "fmt"

//const version = "1.0.0"

type Spur struct {
	AppName string
	Debug   bool
	Version string
}

func (s *Spur) New(rootPath string) error {
	pathConfig := initPaths{
		RootPath:    rootPath,
		FolderNames: []string{"adapter", "cmd", "config", "handlers", "migrations", "views", "public", "logs", "tmp", "Model", "", "utils", "views"},
	}
	err := s.Init(pathConfig)

	if err != nil {
		return err
	}

	err = s.checkDotEnv(rootPath)
	if err != nil {
		return err
	}

	return nil
}

func (s *Spur) Init(p initPaths) error {
	root := p.RootPath
	//creating Folders if they do not exist
	for _, path := range p.FolderNames {
		err := s.CreateDirIfNotExist(root + "/" + path)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *Spur) checkDotEnv(rootPath string) error {

	err := s.CreateFileIfNotExists(fmt.Sprintf("%s/.env", rootPath))
	if err != nil {
		return err
	}

	return nil
}
