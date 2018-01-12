package muxy

import (
	"net/http"
	"encoding/json"
	"strconv"
	"github.com/gorilla/mux"
	log "github.com/golang/glog"
	"encoding/base64"
)

func sendError(w http.ResponseWriter) {
	log.Info("Sending errorcode 500")
	w.WriteHeader(500)
}

func sendJson(w http.ResponseWriter, data interface{}) {
	var dataBytes []byte

	if data != "" {
		var jsonData, err= json.Marshal(data)

		if err != nil {
			log.Error("Could not encode to JSON: " + err.Error())
			sendError(w)
			return
		}

		dataBytes = jsonData
	} else {
		dataBytes = []byte(data.(string))
	}

	w.Header().Set("Content-Type", "application/json")

	log.Info("Sending: " + string(dataBytes))

	w.Write(dataBytes)
}

func streamChannel(w http.ResponseWriter, r *http.Request) {
	encodedStreamURI := mux.Vars(r)["link"]

	decodedStreamURI, err := base64.StdEncoding.DecodeString(encodedStreamURI)
	if err != nil {
		log.Error("Could not decode stream URI: " + encodedStreamURI)
		sendError(w)
		return
	}

	startChannelStream(w, string(decodedStreamURI))
}

func getLineupStatus(w http.ResponseWriter, r *http.Request) {
	sendJson(w, map[string]interface{}{
		"ScanInProgress": 1,
		"ScanPossible": 1,
		"Source": "Cable",
		"SourceList": []string{"Cable"},
	})
}

func getLineup(w http.ResponseWriter, r *http.Request) {
	channels, err := getChannelPlaylist(m3ufile)

	if err != nil {
		log.Error("Could not get channels: " + err.Error())
		sendError(w)
		return
	}

	var lineup []map[string]string

	for _, channel := range channels {
		lineup = append(lineup, map[string]string{
			"GuideNumber": channel.number,
			"GuideName": channel.name,
			"URL": channel.url,
		})
	}

	sendJson(w, lineup)
}

func getDeviceInfo(w http.ResponseWriter, r *http.Request) {
	sendJson(w, deviceInfo)
}

func doNothing(w http.ResponseWriter, r *http.Request) {
	sendJson(w, nil)
}

func getDeviceXmlInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	str := `<root xmlns="urn:schemas-upnp-org:device-1-0">
    <specVersion>
        <major>1</major>
        <minor>0</minor>
    </specVersion>
    <URLBase>` + listenUrl + `</URLBase>
    <device>
        <deviceType>urn:schemas-upnp-org:device:MediaServer:1</deviceType>
        <friendlyName>muxy</friendlyName>
        <manufacturer>Silicondust</manufacturer>
        <modelName>HDTC-2US</modelName>
        <modelNumber>HDTC-2US</modelNumber>
        <serialNumber></serialNumber>
        <UDN>uuid:12345678</UDN>
    </device>
</root>`
	w.Write([]byte(str))
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Info(r.RemoteAddr + " " + r.Method + " " + r.URL.Path)

		if r.Method == "POST" {
			r.ParseForm()
			log.Info("Body: " + r.Form.Encode())
		}

		handler.ServeHTTP(w, r)
	})
}

func SetM3UFile(path string) {
	m3ufile = path
}

func SetMaxStreams(num int) {
	tunerCount = num
}

func SetListenHost(host string) {
	listenHost = host
}

func SetListenPort(port int) {
	listenPort = port
}

func RunListener() {
	router := mux.NewRouter()

	router.HandleFunc("/device.json", getDeviceInfo).Methods("GET")
	router.HandleFunc("/discover.json", getDeviceInfo).Methods("GET")

	router.HandleFunc("/lineup_status.json", getLineupStatus).Methods("GET")
	router.HandleFunc("/lineup.json", getLineup).Methods("GET")

	router.HandleFunc("/lineup.post", doNothing).Methods("GET", "POST")

	router.HandleFunc("/stream/{link:.*}", streamChannel).Methods("GET")

	router.HandleFunc("/", getDeviceXmlInfo).Methods("GET")
	router.HandleFunc("/device.xml", getDeviceXmlInfo).Methods("GET")

	err := http.ListenAndServe(
		listenHost + ":" + strconv.Itoa(listenPort),
		logRequest(router),
	)

	if err != nil {
		log.Error("Could not start listener: " + err.Error())
	}
}
