package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Default locale manager when none is used or implemented
type LokaliseManager struct {
	baseUrl string
	authKey string
}

type LocalesDef struct {
	Project_id string `json:"project_id"`
	Url        string `json:"bundle_url"`
}

func NewLokaliseManager(baseUrl string, authKey string) *LokaliseManager {
	return &LokaliseManager{
		baseUrl: baseUrl,
		authKey: authKey,
	}
}

// Just print a message to stdout when it's called
func (l *LokaliseManager) DownloadLocales(localesPath string) bool {
	archivePath, _ := filepath.Abs(filepath.Join(localesPath, "lokalise.zip"))
	dstPath, _ := filepath.Abs(localesPath)
	log.Println("1. Build package and get url from Lokalise.com")
	localesUrl, err := l.getFileUrl()
	if err != nil {
		fmt.Println(err)
		return false
	}
	client := &http.Client{}
	log.Printf("2. Download file from %s", localesUrl)
	req, err := http.NewRequest("GET", localesUrl, nil)

	if err != nil {
		fmt.Printf("error when retriving locales %s", err)
		return false
	}
	res, err := client.Do(req)
	if err != nil {
		res.Body.Close()
		fmt.Printf("error when retriving locales %s", err)
		return false
	}
	defer res.Body.Close()
	out, err := os.Create(archivePath)
	defer os.Remove(archivePath)
	if err != nil {
		fmt.Printf("error when saving locales %s", err)
		return false
	}
	defer out.Close()
	io.Copy(out, res.Body)
	// Unzip files under locales
	log.Printf("3. Unzip locales")
	_, err = Unzip(archivePath, dstPath)
	if err != nil {
		fmt.Printf("error when saving locales %s", err)
		return false
	}
	return true
}

func (l *LokaliseManager) getFileUrl() (string, error) {
	url := fmt.Sprintf("%s/files/download", l.baseUrl)
	client := &http.Client{}
	payload := strings.NewReader("{\"format\": \"yaml\",\"original_filenames\": false,\"bundle_structure\": \"%LANG_ISO%.yaml\"}")
	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		return "", err
	}

	req.Header.Add("x-api-token", l.authKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		res.Body.Close()
		return "", fmt.Errorf("error when retriving file path %s", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	localesPath := LocalesDef{}
	err = json.Unmarshal(body, &localesPath)
	return localesPath.Url, err
}

func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)) {
			return filenames, fmt.Errorf("%s: illegal file path (potential ZipSlip)", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

/*
func main() {
	LokaliseManager := NewLokaliseManager("https://api.lokalise.com/api2/projects/164383755e2a92a7917465.42378216", "579988c710c246dcb08a96bd13c4a70211fca21b")
	LokaliseManager.DownloadLocales("..")
}
*/
