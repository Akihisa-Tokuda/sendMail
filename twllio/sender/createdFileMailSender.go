package sender

import (
	"io"

	log "github.com/sirupsen/logrus"

	"path/filepath"

	"github.com/spf13/viper"

	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/fsnotify/fsnotify"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func initConfig() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	log.Print(dir)
	viper.SetDefault("targetDir", dir)
	viper.SetDefault("fileType", "png")
	viper.SetConfigType("toml")
	viper.AddConfigPath(dir)
	err := viper.ReadInConfig()
	if err != nil {
		log.Panicf("設定ファイル読み込みエラー: %s \n", err)
	}
}

func StartObserve() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	logfile, err := os.OpenFile(dir+"/sendMail.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic("cannnot open log:" + err.Error())
	}
	defer logfile.Close()

	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	initConfig()

	log.Printf("start observing folder")
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

	log.Printf("file type is matched. Try to send mail :%v", path)

	message := mail.NewV3Mail()
	from := mail.NewEmail("", viper.GetString("fromMailAdress"))
	message.SetFrom(from)

	p := mail.NewPersonalization()
	for _, ad := range viper.GetStringSlice("mailAddresses") {
		to := mail.NewEmail("", ad)
		p.AddTos(to)
	}
	message.AddPersonalizations(p)

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
		log.Println(response.StatusCode)
		log.Println(response.Body)
		log.Println(response.Headers)
	}
}
