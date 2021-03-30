package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/viper"

	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/fsnotify/fsnotify"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func loadConfig() {
	viper.SetDefault("targetDir", "./")
	viper.SetDefault("fileType", "png")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("設定ファイル読み込みエラー: %s \n", err))
	}
}

func main() {
	log.Printf("start")
	loadConfig()

	checkDir()

}

func checkDir() {
	dirname := viper.GetString("targetDir")
	log.Printf("チェック対象ディレクトリ: %v", dirname)
	log.Printf("送信先アドレス：%v", viper.GetStringSlice("mailAddresses"))

	watcher, err := fsnotify.NewWatcher() // (1)
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool) // (2)
	go func() {             // (3)
		for {
			select {
			case event := <-watcher.Events:
				//log.Println("event: ", event)
				switch {
				// case event.Op&fsnotify.Write == fsnotify.Write:
				// 	log.Println("Modified file: ", event.Name)
				case event.Op&fsnotify.Create == fsnotify.Create:
					log.Println("Created file: ", event.Name)
					handleFileCreated(event.Name)
					// case event.Op&fsnotify.Remove == fsnotify.Remove:
					// 	log.Println("Removed file: ", event.Name)
					// case event.Op&fsnotify.Rename == fsnotify.Rename:
					// 	log.Println("Renamed file: ", event.Name)
					// case event.Op&fsnotify.Chmod == fsnotify.Chmod:
					// 	log.Println("File changed permission: ", event.Name)
					//
				}
			case err := <-watcher.Errors:
				log.Println("error: ", err)
				done <- true // (4)
			}
		}
	}()

	err = watcher.Add(dirname) // (5)
	if err != nil {
		log.Fatal(err)
	}

	<-done // (6)
}

func handleFileCreated(path string) {
	if "."+viper.GetString("fileType") != filepath.Ext(path) {
		log.Printf("type is not matched :%v", filepath.Ext(path))
		return
	}

	log.Print(path)
	message := mail.NewV3Mail()
	from := mail.NewEmail("", viper.GetString("fromMailAdress"))
	message.SetFrom(from)

	p := mail.NewPersonalization()
	for _, ad := range viper.GetStringSlice("mailAddresses") {
		to := mail.NewEmail("", ad)
		p.AddTos(to)
		message.AddPersonalizations(p)
	}

	message.Subject = viper.GetString("subject")

	c := mail.NewContent("text/plain", viper.GetString("text"))
	message.AddContent(c)

	// 画像ファイルを添付
	a := mail.NewAttachment()
	file, _ := os.OpenFile(path, os.O_RDONLY, 0600)
	defer file.Close()
	data, _ := ioutil.ReadAll(file)
	data_enc := base64.StdEncoding.EncodeToString(data)
	a.SetContent(data_enc)
	a.SetType("image/" + viper.GetString("fileType"))
	a.SetFilename(file.Name())
	a.SetDisposition("attachment")
	message.AddAttachment(a)

	client := sendgrid.NewSendClient(viper.GetString("sedGridApiKey"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
