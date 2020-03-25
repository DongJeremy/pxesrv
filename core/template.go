package core

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

var templates map[string]*template.Template

const (
	templatePath = "templates"
	targetPath   = "netboot"
)

// PathExists check path exist
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// LoadAndRenderTemplates load tmpl from templates folder
func (s *Service) LoadAndRenderTemplates() (err error) {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	netxServer := fmt.Sprintf("http://%s:%s", s.ServiceIP, s.HTTPPort)
	renderData := map[string]string{
		"NextServer": netxServer,
	}
	templateRoot := filepath.Join(s.DocRoot, templatePath)
	targetRoot := filepath.Join(s.DocRoot, targetPath)
	exist, _ := PathExists(templateRoot)
	if !exist {
		s.Logger.Errorf("template folder %s is not exist", templateRoot)
		return err
	}
	filepath.Walk(templateRoot, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		fileName := filepath.Base(path)
		templatesFileName := template.Must(template.New(fileName).ParseFiles(path))

		destFile := filepath.Join(targetRoot, path[len(templateRoot):])
		destFile = strings.TrimSuffix(destFile, ".tmpl")

		f, err := os.OpenFile(destFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			s.Logger.Errorf("error during template rendering In template %s", destFile)
			return err
		}
		defer f.Close()

		err = templatesFileName.Execute(f, renderData)
		if err != nil {
			return err
		}

		return nil
	})
	s.Logger.Info("[TMPL] templates rendering successful")
	return nil
}
