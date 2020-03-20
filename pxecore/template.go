package pxecore

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

var templates map[string]*template.Template

const (
	templatePath = "templates"
	menuPath     = "netboot/pxelinux.cfg"
	ksPath       = "netboot/linux/ks"
)

// LoadTemplates load tmpl from templates folder
func (s *Server) LoadTemplates() (err error) {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templateLayoutPath := filepath.Join(s.Config.Common.RootPath, templatePath+"/*.tmpl")
	includeFiles, err := filepath.Glob(templateLayoutPath)
	if err != nil {
		return err
	}
	for _, file := range includeFiles {
		fileName := filepath.Base(file)
		templates[fileName] = template.Must(template.New(fileName).ParseFiles(file))
	}
	log.Println("templates loading successful")
	return nil
}

// RenderFile replace variable from config
func (s *Server) RenderFile() (err error) {
	renderData := map[string]string{
		"NextServer": s.Config.Common.NextServer,
	}
	var destFile string
	var f *os.File
	for filename, template := range templates {
		destFileName := strings.TrimSuffix(filename, ".tmpl")
		if strings.Contains(destFileName, "default") {
			destFile = filepath.Join(s.Config.Common.RootPath, menuPath, destFileName)
		} else {
			destFile = filepath.Join(s.Config.Common.RootPath, ksPath, destFileName)
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
	log.Println("templates rendering successful")
	return nil
}
