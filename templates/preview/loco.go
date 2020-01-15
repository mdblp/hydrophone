package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

// Default locale manager when none is used or implemented
type LocoManager struct {
	baseUrl string
	authKey string
}

type Locale struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Source bool   `json:"source"`
}

func NewLocoManager(baseUrl string, authKey string) *LocoManager {
	return &LocoManager{
		baseUrl: baseUrl,
		authKey: authKey,
	}
}

// Just print a message to stdout when it's called
func (l *LocoManager) DownloadLocales(localesPath string) bool {
	log.Println("Download from Loco")
	locales, err := l.getLocales()
	if err != nil {
		fmt.Println(err)
		return false
	}
	for _, v := range locales {
		log.Println("reload " + v.Name)
		l.downloadLocale(v, path.Join(localesPath, v.Code+".yaml"))
	}

	return true

}

func (l *LocoManager) downloadLocale(locale Locale, targetPath string) bool {
	url := fmt.Sprintf("%s/export/locale/%s.yml", l.baseUrl, locale.Code)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		fmt.Printf("error when retriving locales %s", err)
		return false
	}
	req.Header.Add("Authorization", fmt.Sprintf("Loco %s", l.authKey))

	res, err := client.Do(req)
	if err != nil {
		res.Body.Close()
		fmt.Printf("error when retriving locales %s", err)
		return false
	}
	defer res.Body.Close()
	out, err := os.Create(targetPath)
	if err != nil {
		fmt.Printf("error when saving locales %s", err)
		return false
	}
	defer out.Close()
	io.Copy(out, res.Body)
	return true
}

func (l *LocoManager) getLocales() ([]Locale, error) {
	locales := []Locale{}
	url := fmt.Sprintf("%s/locales", l.baseUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Loco %s", l.authKey))

	res, err := client.Do(req)
	if err != nil {
		res.Body.Close()
		return nil, fmt.Errorf("error when retriving locales %s", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(body, &locales)
	return locales, err
}

/*
func main() {
	localizationManager := NewLocoManager(config.LocalizeServiceUrl, config.LocalizeServiceAuthKey)
	localizationManager.DownloadLocales("..")
}*/
