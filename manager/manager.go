package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"code.google.com/p/gcfg"
)

type Param struct {
	Manager struct {
		Command string
	}
	Main struct {
		Data_Path     string
		Queue_Size    int64
		Raw_Log_Level string `gcfg:"log-level"`
		Log_Level     int    `gcfg:"raw-log-level"`
		Report_Url    string `gcfg:"report-url"`
	}
	Client struct {
		Name              string `gcfg:"name"`
		Instance_Id       string `gcfg:"instance-id"`
		Raw_Nodes         string `gcfg:"msgfiber-nodes"`
		Nodes             []string
		Psk               string
		Raw_Ping_Interval string `gcfg:"heartbeat-interval"`
		Ping_Interval     time.Duration
	}
	Channel struct {
		Include string
	}
}

func getConfigPath() string {
	configPath := flag.String("c", "/etc/msgclient/msgclient.conf", "Path of config file or URI")
	flag.Parse()
	return *configPath
}

func main() {
	var err error
	param := new(Param)
	err = gcfg.ReadFileInto(param, getConfigPath())

	if err != nil {
		log.Fatalln(err.Error())
	}

	if param.Manager.Command == "" {
		log.Fatal("No [manager] - command specificed in config file")
	}

	cmdstr := strings.Split(param.Manager.Command, " ")
	cmd := exec.Command(cmdstr[0], cmdstr[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(2 * time.Second):
		os.Exit(0)
	case err := <-done:
		if err != nil {
			log.Fatalf("")
		}
	}

}
