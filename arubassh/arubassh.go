package arubassh

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/drkchiloll/gonet"
)

// Wlc ...
type Wlc struct {
	Client *gonet.Gonet
}

// New ...
func New(host, user, pass, enablePass string) *Wlc {
	return &Wlc{
		Client: &gonet.Gonet{
			IP:       host,
			Username: user,
			Password: pass,
			Vendor:   "aruba",
			Enable:   enablePass,
		},
	}
}

// SetApName ...
func (w *Wlc) SetApName(wiredMac, newName string) {
	// Move the Config Terminal Mode
	w.Client.SendConfig("configure terminal")
	cmd := fmt.Sprintf("ap-rename wired-mac %s %s", wiredMac, newName)
	// Set the AP Name using the Wire MAC Address of the AP
	w.Client.SendConfig(cmd)
	// Exit Config Mode
	w.Client.SendConfig("exit")
}

// SetApGroup ...
func (w *Wlc) SetApGroup(wiredMac, newGroup string) {
	// Move the Config Terminal Mode
	w.Client.SendConfig("configure terminal")
	cmd := fmt.Sprintf("ap-regroup wired-mac %s %s", wiredMac, newGroup)
	// Set the AP Name using the Wire MAC Address of the AP
	w.Client.SendConfig(cmd)
	// Exit Config Mode
	w.Client.SendConfig("exit")
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

// GetApDb ...
func (w *Wlc) GetApDb() ([]AP, error) {
	var aps []AP
	out, err := w.Client.SendCmd("show ap database long")
	if err != nil {
		return aps, err
	}
	lines := strings.Split(out, "\n")
	re := regexp.MustCompile(`^ap\d+\S+|^(\w+:){5}\w+`)
	var apLines []string
	for _, line := range lines {
		if re.MatchString(line) {
			line = trimWS(line)
			apLines = append(apLines, line)
		}
	}
	for _, line := range apLines {
		apList := strings.Split(line, " ")
		ap := AP{
			Name:   apList[0],
			Group:  apList[1],
			Model:  apList[2],
			Status: strings.ToLower(apList[4]),
		}

		switch ap.Status {
		case "up":
			ap.PrimaryWlc = apList[7]
			ap.MacAddr = apList[9]
			ap.Serial = apList[10]
		case "down":
			ap.PrimaryWlc = apList[6]
			ap.MacAddr = apList[8]
			ap.Serial = apList[9]
		}
		aps = append(aps, ap)
	}
	return aps, nil
}

// ApIntf ...
type ApIntf struct {
	Status string
	Speed  string
	Duplex string
	Tx     string
	Rcv    string
}

// GetApIntfStats ...
func (w *Wlc) GetApIntfStats(wiredMac string) ApIntf {
	cmd := fmt.Sprintf("show ap port status wired-mac %s", wiredMac)
	out, _ := w.Client.SendCmd(cmd)
	re := regexp.MustCompile(`(\w+:){5}\w+`)
	lines := strings.Split(out, "\n")
	contains := strings.Contains
	for _, line := range lines {
		line = trimWS(line)
		if re.MatchString(line) && !contains(line, cmd) && !contains(line, "down") {
			intfSplit := strings.Split(line, " ")
			apIntf := ApIntf{
				Status: intfSplit[5],
				Speed:  intfSplit[6] + " " + intfSplit[7],
				Duplex: intfSplit[8],
				Tx:     intfSplit[15],
				Rcv:    intfSplit[17],
			}
			return apIntf
		}
	}
	return ApIntf{}
}

// APLldp the properties of a Neighbor Connected to the AP
type APLldp struct {
	RemoteSw   string
	RemoteIntf string
}

// GetApLLDPInfo ...
func (w *Wlc) GetApLLDPInfo(apName string) APLldp {
	re := regexp.MustCompile(`^ap\d+\S+`)
	cmd := fmt.Sprintf("show ap lldp neighbors ap-name %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	lines := strings.Split(out, "\n")
	var apLLDP APLldp
	for _, line := range lines {
		if re.MatchString(line) {
			line = trimWS(line)
			lldpSplit := strings.Split(line, " ")
			apLLDP = APLldp{
				RemoteSw:   lldpSplit[3],
				RemoteIntf: lldpSplit[4],
			}
		}
	}
	return apLLDP
}

func trimWS(text string) string {
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(text, " ")
}
