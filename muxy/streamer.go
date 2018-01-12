package muxy

import (
	"os"
	log "github.com/golang/glog"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"strconv"
	"net/url"
	"time"
)

func prepareCacheDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}

	return nil
}

func waitForNextSegment() {
	time.Sleep(9 * time.Second)
}

func startChannelStream(writer http.ResponseWriter, channelPlaylist string) {
	streamID := filepath.Base(channelPlaylist)
	streamID = strings.TrimSuffix(streamID, filepath.Ext(streamID))

	segmentHostUrl, err := url.Parse(channelPlaylist)
	if err != nil {
		log.Error("Could not parse host from " + channelPlaylist)
		sendError(writer)
		return
	}

	segmentHost := segmentHostUrl.Scheme + "://" + segmentHostUrl.Host

	log.Info("Streaming stream " + streamID)

	refetch:

	for true {

		segments, err := FetchStreamSegments(channelPlaylist, streamID)
		if err != nil {
			log.Error("Could not fetch channel playlist: " + err.Error())
			sendError(writer)
			return
		}

		for _, segment := range segments {
			fullSegmentUrl := segmentHost + segment.url

			log.Info("Downloading segment " + fullSegmentUrl)

			client := &http.Client{}

			req, err := http.NewRequest("GET", fullSegmentUrl, nil)
			if err != nil {
				log.Error("Could not request segment: " + err.Error())
				break refetch
			}

			req.Header.Set("User-Agent", "VLC")

			response, err := client.Do(req)

			defer response.Body.Close()

			if err != nil || (response != nil && response.StatusCode != 200) {

				if response == nil {
					log.Error("Could not fetch segment: " + err.Error())
					continue
				}

				if response.Header.Get("Retry-After") != "" {
					// rate limited
					log.Warning("Rate limited, waiting for " + response.Header.Get("Retry-After") + " seconds")
					secWait, _ := strconv.Atoi(response.Header.Get("Retry-After"))
					time.Sleep(time.Second * time.Duration(secWait))
					break refetch
				}
			}

			if response.StatusCode != 200 {
				log.Warning("Status code is " + strconv.Itoa(response.StatusCode))
				continue
			}

			var segmentBytes []byte
			bytesNum, err := response.Body.Read(segmentBytes)
			if err != nil {
				log.Error("Error while fetching segment: " + err.Error())
				break refetch
			}

			log.Info("Downloaded & sending " + strconv.Itoa(bytesNum) + " bytes")
			writer.Write(segmentBytes)

			waitForNextSegment()
		}
	}
}

func FetchStreamSegments(url string, streamID string) ([]Channel, error) {

	tempStreamPath := tempTSDirectory + "/" + streamID
	log.Info("Stream temp dir will be " + tempStreamPath)

	if nil != prepareCacheDir(tempStreamPath) {
		return nil, errors.New("Could not create stream cache dir: " + tempStreamPath)
	}

	tempStreamFilename := tempStreamPath + "/" + streamID

	mediaPlayList, err := parseM3UFile(url, tempStreamFilename)
	if err != nil {
		return nil, errors.New("Could not get channel playlist: " + err.Error())
	}

	var channels []Channel;
	for index, segment := range mediaPlayList.Segments {

		if true == strings.Contains(segment.Title, "▬") {
			continue
		}

		log.Info("Adding Segment{" + "0." + strconv.Itoa(index) + "," + segment.Title + "," + segment.URI + "}")
		channels = append(channels, Channel{"0." + strconv.Itoa(index), segment.Title, segment.URI})
	}

	return channels, nil
}