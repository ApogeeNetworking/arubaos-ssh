package arubassh

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ApogeeNetworking/gonetssh"
	"github.com/ApogeeNetworking/gonetssh/universal"
)

var contains = strings.Contains

// Wlc ...
type Wlc struct {
	Client universal.Device
}

// New ...
func New(host, user, pass, enablePass string) *Wlc {
	cl, _ := gonetssh.NewDevice(host, user, pass, enablePass, gonetssh.DType.Aruba)
	return &Wlc{Client: cl}
}

// SetApName ...
func (w *Wlc) SetApName(wiredMac, newName string) {
	// Move the Config Terminal Mode
	cmd := fmt.Sprintf("ap-rename wired-mac %s %s", wiredMac, newName)
	// Set the AP Name using the Wire MAC Address of the AP
	w.Client.SendCmd(cmd)
}

// SetApGroup ...
func (w *Wlc) SetApGroup(wiredMac, newGroup string) {
	// Move the Config Terminal Mode
	cmd := fmt.Sprintf("ap-regroup wired-mac %s %s", wiredMac, newGroup)
	// Set the AP Name using the Wire MAC Address of the AP
	w.Client.SendCmd(cmd)
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
	apNameRe := regexp.MustCompile(`^ap\d+\S+|^(\w+:){5}\w+`)
	serialRe := regexp.MustCompile(`(\w+){7,15}`)
	macRe := regexp.MustCompile(`(\w+:){5}\w+`)
	wlcIPRe := regexp.MustCompile(`(\d+\.){3}\d+`)
	var apLines []string
	for _, line := range lines {
		if apNameRe.MatchString(line) {
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
		ap.MacAddr = macRe.FindString(line)
		serialStr := strings.Join(apList[7:], " ")
		priWlcStr := strings.Join(apList[5:8], " ")
		ap.Serial = serialRe.FindString(serialStr)
		ap.PrimaryWlc = wlcIPRe.FindString(priWlcStr)
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

// ApIntfParams ...
type ApIntfParams struct {
	MacAddr string
	ApName  string
}

// GetApIntf ...
func (w *Wlc) GetApIntf(p ApIntfParams) ApIntf {
	var apIntf ApIntf
	var cmd string
	switch {
	case p.MacAddr != "":
		cmd = fmt.Sprintf("show ap port status wired-mac %s", p.MacAddr)
	case p.ApName != "":
		cmd = fmt.Sprintf("show ap port status ap-name %s", p.ApName)
	}
	out, _ := w.Client.SendCmd(cmd)
	if p.MacAddr != "" {
		res := fmt.Sprintf("AP with MAC address %s not found.", p.MacAddr)
		if contains(out, res) {
			return apIntf
		}
	}
	if contains(out, "No information available for this AP") {
		return apIntf
	}
	// MAC Address Regular Expression
	re := regexp.MustCompile(`(\w+:){5}\w+`)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = trimWS(line)
		if re.MatchString(line) &&
			!contains(line, "show") &&
			!contains(line, "wired-mac") &&
			!contains(line, "down") {
			intfSplit := strings.Split(line, " ")
			if len(intfSplit) < 18 {
				continue
			}
			apIntf = ApIntf{
				Status: intfSplit[5],
				Speed:  intfSplit[6] + " " + intfSplit[7],
				Duplex: intfSplit[8],
				Tx:     intfSplit[15],
				Rcv:    intfSplit[17],
			}
			break
		}
	}
	return apIntf
}

// APLldp the properties of a Neighbor Connected to the AP
type APLldp struct {
	RemoteSw   string
	RemoteIntf string
}

// GetApLLDPInfo ...
func (w *Wlc) GetApLLDPInfo(apName string) APLldp {
	var apLLDP APLldp
	re := regexp.MustCompile(`^ap\d+\S+`)
	cmd := fmt.Sprintf("show ap lldp neighbors ap-nam %s", apName)
	out, _ := w.Client.SendCmd(cmd)
	if strings.Contains(out, "AP is down") {
		return apLLDP
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if re.MatchString(line) {
			line = trimWS(line)
			lldpSplit := strings.Split(line, " ")
			if len(lldpSplit) < 5 {
				return apLLDP
			}
			apLLDP = APLldp{
				RemoteSw:   lldpSplit[3],
				RemoteIntf: lldpSplit[4],
			}
		}
	}
	return apLLDP
}

// WirelessClient ...
type WirelessClient struct {
	IPAddr     string
	ApName     string
	Auth       string
	BSSID      string
	SSID       string
	MacAddr    string
	DeviceType string
	Channel    int
	TxBytes    int64
	RcvBytes   int64
}

// GetWirelessClients ...
func (w *Wlc) GetWirelessClients() []WirelessClient {
	var clients []WirelessClient
	out, _ := w.Client.SendCmd("show user-table")
	ipRe := regexp.MustCompile(`(\d+\.){3}\d+`)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if ipRe.MatchString(line) {
			ip := ipRe.FindString(line)
			line = trimWS(line)
			clSplit := strings.Split(line, " ")
			client := WirelessClient{
				ApName:  clSplit[4],
				IPAddr:  ip,
				MacAddr: clSplit[1],
			}
			// Front [SSID/... to DeviceType]
			endArr := clSplit[6:]
			endJoin := strings.Join(endArr, " ")
			ssidRe := regexp.MustCompile(`(.*/){2}\S+`)
			// WhiteSpace in SSID
			ssidWsRe := regexp.MustCompile(`(\w+\s\w+)/`)
			ssidStr := ssidRe.FindString(endJoin)
			if ssidStr != "" {
				ssidSplit := strings.Split(ssidStr, "/")
				client.SSID = ssidSplit[0]
				client.BSSID = ssidSplit[1]
			}
			switch {
			// DeviceTypes do not have Spaces within them
			case len(endArr) == 5 && !ssidWsRe.MatchString(endJoin):
				client.DeviceType = endArr[3]
			// DeviceType's Are [OS X | Window x] (multi-spaced DeviceType)
			case len(endArr) == 6 && !ssidWsRe.MatchString(endJoin):
				client.DeviceType = endArr[3] + " " + endArr[4]
			case len(endArr) == 6 && ssidWsRe.MatchString(endJoin):
				client.DeviceType = endArr[5]
			// DeviceType's are OS X | Windows x && SSID with WhiteSpace
			case len(endArr) == 7 && ssidWsRe.MatchString(endJoin):
				client.DeviceType = endArr[4] + " " + endArr[5]
			}
			if client.DeviceType == "" {
				client.DeviceType = "Unknown"
			}
			clients = append(clients, client)
		}
	}
	return clients
}

// GetClientDetails ...
func (w *Wlc) GetClientDetails(client *WirelessClient) WirelessClient {
	var channel int
	var tx, rcv int64
	cmd := fmt.Sprintf("sh ap association client-mac %s | beg Parameter", client.MacAddr)
	out, _ := w.Client.SendCmd(cmd)
	chanRe := regexp.MustCompile(`Channel\s+(\d+)`)
	txRe := regexp.MustCompile(`Client\sTx\sBytes\s+(\d+)`)
	rcvRe := regexp.MustCompile(`Client\sRx\sBytes\s+(\d+)`)
	if chanRe.MatchString(out) {
		chMatch := chanRe.FindStringSubmatch(out)
		if len(chMatch) == 2 {
			channel, _ = strconv.Atoi(chMatch[1])
			client.Channel = channel
		}
	}
	if txRe.MatchString(out) {
		txMatch := txRe.FindStringSubmatch(out)
		if len(txMatch) == 2 {
			tx, _ = strconv.ParseInt(txMatch[1], 10, 64)
			client.TxBytes = tx
		}
	}
	if rcvRe.MatchString(out) {
		rxMatch := rcvRe.FindStringSubmatch(out)
		if len(rxMatch) == 2 {
			rcv, _ = strconv.ParseInt(rxMatch[1], 10, 64)
			client.RcvBytes = rcv
		}
	}
	return *client
}

func trimWS(text string) string {
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(text, " ")
}
