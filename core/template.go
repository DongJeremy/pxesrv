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
	menuPath     = "netboot"
	ksPath       = "netboot/linux/ks"
)

// LoadTemplates load tmpl from templates folder
func (s *Service) LoadTemplates() (err error) {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templateLayoutPath := filepath.Join(s.DocRoot, templatePath+"/*.tmpl")
	includeFiles, err := filepath.Glob(templateLayoutPath)
	if err != nil {
		return err
	}
	for _, file := range includeFiles {
		fileName := filepath.Base(file)
		templates[fileName] = template.Must(template.New(fileName).ParseFiles(file))
	}
	log.Info("templates loading successful")
	return nil
}

// RenderFile replace variable from config
func (s *Service) RenderFile() (err error) {
	netxServer := fmt.Sprintf("http://%s:%s", s.ServiceIP, s.HTTPPort)
	renderData := map[string]string{
		"NextServer": netxServer,
	}
	var destFile string
	var f *os.File
	for filename, template := range templates {
		destFileName := strings.TrimSuffix(filename, ".tmpl")
		if strings.Contains(destFileName, "menu") {
			destFile = filepath.Join(s.DocRoot, menuPath, destFileName)
		} else {
			destFile = filepath.Join(s.DocRoot, ksPath, destFileName)
		}
		f, err = os.OpenFile(destFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Errorf("error during template rendering In template %s", destFile)
			return
		}
		defer f.Close()

		err = template.Execute(f, renderData)
		if err != nil {
			return
		}
	}
	log.Info("templates rendering successful")
	return nil
}
