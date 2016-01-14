package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/0xAX/notificator"
	"github.com/everdev/mack"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	_ToDownload = "toDownload"
	_Downloaded = "downloaded"
)

type Result struct {
	Url        string
	Downloaded bool
}

type DataBase struct {
	session *mgo.Session
}

func main() {
	var database DataBase
	var notify *notificator.Notificator
	notify = notificator.New(notificator.Options{
		DefaultIcon: "icon/default.png",
		AppName:     "Youtube",
	})

	database.session, _ = mgo.Dial("localhost:27017")
	defer database.session.Close()
	fmt.Println("session connected to mongodb")
	c := database.session.DB("youtube").C("toDownload")
	var R *Result
	//fmt.Println(*R)
	iter := c.Find(nil).Sort("$natural").Tail(time.Second * 5)
	for {
		for iter.Next(&R) {
			Res := R
			fmt.Println("Added result : " + Res.Url)
			notify.Push("Youtube Video", Res.Url, "", notificator.UR_NORMAL)
			mack.Notify(Res.Url, "Youtube")
			if !R.isDownloaded(&database) {
				go func(Res *Result) {
					err := R.download()
					if err == nil {
						Res.updateDB(&database)
						fmt.Println("waiting for : " + Res.Url)
						if err != nil {
							fmt.Println("Error while downloading video : " + Res.Url)
							fmt.Println(err)
						}
					}
				}(Res)
			} else {
				log.Printf("Video is already downloaded ", R.Url)
			}
		}
	}
}

func (R *Result) updateDB(database *DataBase) {
	R.Downloaded = true
	c := database.session.DB("youtube").C("toDownload")
	c.Update(bson.M{"url": R.Url}, &R)
}

func (R *Result) isDownloaded(database *DataBase) bool {
	c := database.session.DB("youtube").C("toDownload")
	err := c.Find(bson.M{"url": R.Url}).One(&R)
	if err != nil {
		return false
	}
	if R.Downloaded {
		return true
	}
	return false
}

func (R *Result) download() error {
	// docker build current directory
	cmdName := "youtube-dl"
	cmdArgs := []string{"-o", "/Users/akash/youtube_video/", "-c", "-f", "22", R.Url}

	cmd := exec.Command(cmdName, cmdArgs...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		return errors.New("Failed")
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			fmt.Printf("docker build out | %s\n", scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		return errors.New("Failed")
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		return errors.New("Failed")
	}
	return errors.New("Failed")
}
