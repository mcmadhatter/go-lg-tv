package lgtv

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	// For converting stuff to and from hex
	// For networking stuff
)

type TV struct {
	ApplicationID   string
	ApplicationName string
	Id              string
	Name            string
	Ip              net.IP
	Found           bool
	Pin             string
}

type LG_TV_COMMAND int

const (
	_            = iota
	TV_CMD_POWER = 1 + iota
	TV_CMD_NUMBER_0
	TV_CMD_NUMBER_1
	TV_CMD_NUMBER_2
	TV_CMD_NUMBER_3
	TV_CMD_NUMBER_4
	TV_CMD_NUMBER_5
	TV_CMD_NUMBER_6
	TV_CMD_NUMBER_7
	TV_CMD_NUMBER_8
	TV_CMD_NUMBER_9
	TV_CMD_UP
	TV_CMD_DOWN
	TV_CMD_LEFT
	TV_CMD_RIGHT
	TV_CMD_OK = 5 + iota
	TV_CMD_HOME_MENU
	TV_CMD_BACK
	TV_CMD_VOLUME_UP
	TV_CMD_VOLUME_DOWN
	TV_CMD_MUTE_TOGGLE
	TV_CMD_CHANNEL_UP
	TV_CMD_CHANNEL_DOWN
	TV_CMD_BLUE
	TV_CMD_GREEN
	TV_CMD_RED
	TV_CMD_YELLOW
	TV_CMD_PLAY
	TV_CMD_PAUSE
	TV_CMD_STOP
	TV_CMD_FAST_FORWARD
	TV_CMD_REWIND
	TV_CMD_SKIP_FORWARD
	TV_CMD_SKIP_BACKWARD
	TV_CMD_RECORD
	TV_CMD_RECORDING_LIST
	TV_CMD_REPEAT
	TV_CMD_LIVE_TV
	TV_CMD_EPG
	TV_CMD_PROGRAM_INFORMATION
	TV_CMD_ASPECT_RATIO
	TV_CMD_EXTERNAL_INPUT
	TV_CMD_PIP_SECONDARY_VIDEO
	TV_CMD_SHOW_SUBTITLE
	TV_CMD_PROGRAM_LIST
	TV_CMD_TELE_TEXT
	TV_CMD_MARK
	TV_CMD_3D_VIDEO = 48 + iota
	TV_CMD_3D_LR
	TV_CMD_DASH
	TV_CMD_PREVIOUS_CHANNEL
	TV_CMD_FAVORITE_CHANNEL
	TV_CMD_QUICK_MENU
	TV_CMD_TEXT_OPTION
	TV_CMD_AUDIO_DESCRIPTION
	TV_CMD_ENERGY_SAVING
	TV_CMD_AV_MODE
	TV_CMD_SIMPLINK
	TV_CMD_EXIT
	TV_CMD_RESERVATION_PROGRAM_LIST
	TV_CMD_PIP_CHANNEL_UP
	TV_CMD_PIP_CHANNEL_DOWN
	TV_CMD_SWITCH_VIDEO
	TV_CMD_APPS
)

var socketsUp = false

var conn *net.UDPConn // UDP Connection

// SendHttpReqToLGTV sends an http request to the LG smart tv specified by *TV
func (tv *TV) SendHttpReqToLGTV(msgType string, message string) (int, error, string) {
	buf := []byte(message)
	body := bytes.NewBuffer(buf)
	client := &http.Client{}
	r, _ := http.NewRequest("POST", "http://"+tv.Ip.String()+":8080"+msgType, body)

	r.Header.Add("Content-Type", "text/xml; charset=utf-8")
	r.Header.Add("Content-Length", strconv.Itoa(len(buf)))
	r.Header.Add("Connection", "Close")
	r.Header.Add("User-Agent", "Linux/2.6.18 UDAP/2.0 NinjaSphere/0.1")

	resp, err := client.Do(r)

	if err != nil {

		fmt.Println(err)
		return 0, err, ""

	} else if resp.StatusCode == 200 { // OK

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return resp.StatusCode, nil, string(bodyBytes)
	} else {
		return resp.StatusCode, nil, ""
	}
}

//GetTVToShowPin  ask the LG smart tv to show it's pin number on the screen so as to faclitate pairing
func (tv *TV) GetTVToShowPin() {

	if socketsUp == false {
		prepareSockets()
	}
	broadcastMessage("1990", "B-SEARCH * HTTP/1.1\r\nHOST: 255.255.255.255:1990\r\nMAN: \"ssdp:discover\"\r\nMX: 3\r\nST: urn:schemas-udap:service:smartText:1\r\nUSER-AGENT: linux UDAP/2.0 ninjasphere\r\n\r\n")
	for tv.Found == false {
		tv.checkForMessages()
	}

}

//PairWithPin pair with the specified pin, using the pin in tv config
func (tv *TV) PairWithPin() {

	fmt.Println("Pairing with TV: " + tv.Name + " using Pin: " + tv.Pin)

	tv.SendHttpReqToLGTV("/udap/api/pairing", "<?xml version=\"1.0\" encoding=\"utf-8\"?><envelope><api type=\"pairing\"><name>hello</name><value>"+tv.Pin+"</value><port>8080</port></api></envelope>")
}

//SendCommandCode sends a command to the tv
func (tv *TV) SendCommandCode(cmd int) bool {
	fmt.Println("Sending command " + strconv.Itoa(cmd) + " to an LG tv called " + tv.Name)

	success, _, _ := tv.SendHttpReqToLGTV("/udap/api/command", "<?xml version=\"1.0\" encoding=\"utf-8\"?><envelope><api type=\"command\"><name>HandleKeyInput</name><value>"+strconv.Itoa(cmd)+"</value></api></envelope>")

	/* it is necessary to pair everytime the LG tv is turned on , if a command fails here try repairing */
	if success != 200 {
		tv.PairWithPin()
		success, _, _ = tv.SendHttpReqToLGTV("/udap/api/command", "<?xml version=\"1.0\" encoding=\"utf-8\"?><envelope><api type=\"command\"><name>HandleKeyInput</name><value>"+strconv.Itoa(cmd)+"</value></api></envelope>")

	}

	return success == 200
}

func broadcastMessage(portAddr string, msg string) {
	fmt.Println("Broadcasting message:", msg, "to", net.IPv4bcast.String()+":"+portAddr)
	udpAddr, err := net.ResolveUDPAddr("udp4", net.IPv4bcast.String()+":"+portAddr)
	if err != nil {
		fmt.Println("ERROR!:", err)
		os.Exit(1)
	}
	//	buf, _ := hex.DecodeString(msg)
	buf := []byte(msg)
	// If we'e got an error
	if err != nil {
		fmt.Println("ERROR!:", err)
		os.Exit(1)
	}

	_, _ = conn.WriteToUDP(buf, udpAddr)
	return
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Oops: " + err.Error() + "\n")
		return ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPAddr:
				return v.IP.String()
			}

		}
	}

	return ""
}

func prepareSockets() {
	if getLocalIP() == "" {
		fmt.Println("Error: Can't determine local IP address. Exiting!")
		os.Exit(1)
	} else {
		fmt.Println("Local IP is:", getLocalIP())
	}

	udpAddr, err := net.ResolveUDPAddr("udp4", ":1990") // Get our address ready for listening
	if err != nil {
		fmt.Println("Resolve:", err)
		os.Exit(1)
	}
	conn, err = net.ListenUDP("udp", udpAddr) // Now we listen on the address we just resolved
	if err != nil {
		fmt.Println("Listen:", err)
		os.Exit(1)
	}
	socketsUp = true
}

func (tv *TV) checkForMessages() (bool, error) { // Now we're checking for messages

	//	var msg string     // Holds the incoming message
	var buf [1024]byte // We want to get 1024 bytes of messages (is this enough? Need to check!)

	var success bool
	var err error

	n, addr, _ := conn.ReadFromUDP(buf[0:]) // Read 1024 bytes from the buffer
	ip := getLocalIP()                      // Get our local IP
	if n > 0 && addr.IP.String() != ip {    // If we've got more than 0 bytes and it's not from us

		msg := string(buf[0:n])                    // n is how many bytes we grabbed from UDP
		success, err = tv.handleMessage(msg, addr) // Hand it off to our handleMessage func. We pass on the message and the address (for replying to messages)

	}

	return success, err
}

// handleMessage parses a message found by CheckForMessages
func (tv *TV) handleMessage(message string, addr *net.UDPAddr) (bool, error) {

	if len(message) == 0 { // Blank message? Don't try and parse it!
		return false, errors.New("Blank message")
	}

	if addr.Port == 1990 {
		/* this was a braodcast response */
		rTV, err := regexp.Compile(`SERVER: [\w//.]* [\w//.]* ([\w-]*)`)
		if err != nil {
			fmt.Println("Could not find an LG TV")
		} else {

			tvNames := rTV.FindStringSubmatch(message)

			tv.Name = tvNames[1]
			tv.Found = true
			tv.Ip = addr.IP
			fmt.Println("TV Name is " + tv.Name + " IP address is " + tv.Ip.String())
			tv.pairingRequestPin()

		}
	} else if (addr.IP.String() == tv.Ip.String()) && (tv.Found == true) {

		fmt.Println("Message from", addr.IP, "message is", message)
	} else {
		/* Ignore */
	}

	return true, nil
}

func (tv *TV) pairingRequestPin() {
	code, err, data := tv.SendHttpReqToLGTV("/udap/api/pairing", "<?xml version=\"1.0\" encoding=\"utf-8\"?><envelope><api type=\"pairing\"><name>showKey</name></api></envelope>")

	if err != nil {
		fmt.Println("Pin Request failed")
	} else if code != 200 {
		fmt.Println("Pin Request failed: %s", data)
	} else {
		fmt.Println("Pin Request ok")
	}
}
