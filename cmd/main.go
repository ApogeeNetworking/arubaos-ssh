package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/ApogeeNetworking/arubassh"
	"github.com/subosito/gotenv"
)

var host, user, pass, enablePass string

func init() {
	gotenv.Load()
	host = os.Getenv("SSH_HOST")
	user = os.Getenv("SSH_USER")
	pass = os.Getenv("SSH_PW")
	enablePass = os.Getenv("SSH_ENABLE_PW")
}

func main() {
	wlc := arubassh.New(host, user, pass, enablePass, "8")
	err := wlc.Client.Connect(10)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer wlc.Client.Disconnect()

	ssids := wlc.GetSSIDs()
	for _, ssid := range ssids {
		fmt.Println(ssid)
	}

	// count := wlc.GetClientCountBySSID("MyCampusNet-Legacy-TheNineAtRio")
	// fmt.Println(count)
}

func trimWS(text string) string {
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(text, " ")
}

func normalizeMac(m string) string {
	return strings.ToLower(m[0:2] + ":" + m[2:4] + ":" + m[4:6] +
		":" + m[6:8] + ":" + m[8:10] + ":" + m[10:12])
}
