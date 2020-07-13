package main

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/drkchiloll/arubaos-ssh/arubassh"
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

// AP the properties that exist on AccessPoints
type AP struct {
	MacAddr      string
	Name         string
	Group        string
	Model        string
	Serial       string
	IPAddr       string
	Status       string
	PrimaryWlc   string
	SecondaryWlc string
}

func main() {
	wlc := arubassh.New(host, user, pass, enablePass)
	err := wlc.Client.Connect(10)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer wlc.Client.Close()

	// aps, _ := wlc.GetApDb()
	// for _, ap := range aps {
	// 	fmt.Println(ap)
	// }
	// apIntf := wlc.GetApIntfStats("94:b4:0f:c6:d6:1a")
	// fmt.Println(apIntf)
	apLLDP := wlc.GetApLLDPInfo("ap01Victory.Laundry.unt.tx")
	fmt.Println(apLLDP)
}

func trimWS(text string) string {
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(text, " ")
}
