package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

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

func main() {
	d, _ := ioutil.ReadFile("aplist.txt")
	payload := string(d)

	lines := strings.Split(payload, "\n")
	wlc := arubassh.New(host, user, pass, enablePass)
	err := wlc.Client.Connect(10)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer wlc.Client.Close()
	for _, line := range lines {
		// apLLDP := wlc.GetApLLDPInfo(line)
		// fmt.Println(apLLDP)
		apIntf := wlc.GetApIntfStats(line)
		fmt.Println(apIntf)
	}
	// out, _ := wlc.Client.SendCmd("show loginsessions")
	// ipRE := regexp.MustCompile(`(\d+\.){3}\d+`)
	// var logins int
	// lines := strings.Split(out, "\n")
	// for _, line := range lines {
	// 	if ipRE.MatchString(line) {
	// 		logins++
	// 	}
	// }
	// fmt.Println(logins)
}

func normalizeMac(m string) string {
	return strings.ToLower(m[0:2] + ":" + m[2:4] + ":" + m[4:6] +
		":" + m[6:8] + ":" + m[8:10] + ":" + m[10:12])
}

func p() {
	wlc := arubassh.New(host, user, pass, enablePass)
	err := wlc.Client.Connect(10)
	if err != nil {
		log.Fatalf("%v", err)
	}

	aps, _ := wlc.GetApDb()
	wlc.Client.Close()

	conns := spawn()
	defer release(conns)
	var wg sync.WaitGroup
	var mut sync.Mutex
	sem := make(chan struct{}, 4)
	var apStatus []ApStatus
	for _, ap := range aps {
		wg.Add(1)
		sem <- struct{}{}
		go func(ap arubassh.AP) {
			defer wg.Done()
			apStat := make(chan ApStatus, 1)
			worker := &ConnPoolWorker{
				Conns:    conns,
				Sem:      sem,
				Ap:       &ap,
				ApStatus: apStat,
			}
			go work(worker)
			stat := <-worker.ApStatus
			mut.Lock()
			apStatus = append(apStatus, stat)
			mut.Unlock()
		}(ap)
	}
	wg.Wait()
	for _, s := range apStatus {
		fmt.Println(s)
	}
}

func work(worker *ConnPoolWorker) {
WorkLoop:
	for {
		for i, con := range worker.Conns {
			if !con.InUse {
				worker.Conns[i].InUse = true
				apIntf := con.Awlc.GetApIntfStats(worker.Ap.MacAddr)
				apLLDP := con.Awlc.GetApLLDPInfo(worker.Ap.Name)
				worker.ApStatus <- ApStatus{
					Status:     apIntf.Speed + "-" + strings.ToUpper(apIntf.Duplex),
					RemoteSw:   apLLDP.RemoteSw,
					RemoteIntf: apLLDP.RemoteIntf,
				}
				<-worker.Sem
				worker.Conns[i].InUse = false
				break WorkLoop
			}
		}
	}
}

// ApStatus ...
type ApStatus struct {
	Status     string
	RemoteSw   string
	RemoteIntf string
}

// ConnPoolWorker ...
type ConnPoolWorker struct {
	Conns    [4]*ConnPool
	Ap       *arubassh.AP
	Sem      chan struct{}
	ApStatus chan ApStatus
}

// ConnPool ...
type ConnPool struct {
	Awlc  *arubassh.Wlc
	InUse bool
}

func spawn() [4]*ConnPool {
	var conns [4]*ConnPool
	for i := 0; i < 4; i++ {
		w := arubassh.New(host, user, pass, enablePass)
		err := w.Client.Connect(5)
		if err != nil {
			log.Fatalf("%v", err)
		}
		conns[i] = &ConnPool{Awlc: w}
	}
	return conns
}

func release(conns [4]*ConnPool) {
	for _, c := range conns {
		c.Awlc.Client.Close()
	}
}
