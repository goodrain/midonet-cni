package ipam

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"time"

	"github.com/Sirupsen/logrus"
	"github.com/goodrain/midonet-cni/pkg/types"
)

//RegionAPI regin api
type RegionAPI struct {
	ReginNetAPI string
	Token       string //5ca196801173be06c7e6ce41d5f7b3b8071e680a
	HTTPTimeOut time.Duration
}

//GetNewIP 申请新ip
func (r *RegionAPI) GetNewIP(info types.ReginNewIP, namespace string) (string, error) {
	var err error
	var ip = ""
	var url = r.ReginNetAPI + "midolnet/" + namespace + "/bindings"
	var jsonStr, _ = json.Marshal(info)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logrus.Error("Create RegionAPI request error.", err.Error())
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+r.Token)
	client := &http.Client{
		Timeout: r.HTTPTimeOut,
	}
	errSize := 0
	logrus.Debug("Start try get ip from RegionAPI")
	for {
		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			logrus.Errorf("ReginAPI (%s) Get new ip error %s,", url, err.Error())
			if errSize < 3 {
				errSize++
				continue
			}
			break
		} else {
			if resp.StatusCode != 201 {
				logrus.Error("RegionAPI error response:", resp.Status, resp.Body)
				if errSize < 3 {
					errSize++
					continue
				}
				break
			} else {
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return "", err
				}
				logrus.Info("PostDockerIdForIPNew:====" + url + "====" + string(jsonStr) + "====" + string(body))
				dataMap := parseJSON(body)
				ip = dataMap["ip"]
				break
			}
		}
	}
	return ip, err
}

func parseJSON(body []byte) map[string]string {
	j2 := make(map[string]interface{})
	json.Unmarshal(body, &j2)
	dataMap := map[string]string{}
	for k, v := range j2 {
		switch vv := v.(type) {
		case string:
			dataMap[k] = vv
		case int:
			dataMap[k] = strconv.Itoa(vv)
		case int8:
			dataMap[k] = strconv.Itoa(int(vv))
		case int16:
			dataMap[k] = strconv.Itoa(int(vv))
		case int32:
			dataMap[k] = strconv.Itoa(int(vv))
		case int64:
			dataMap[k] = strconv.Itoa(int(vv))
		case float32:
			dataMap[k] = strconv.Itoa(int(vv))
		case float64:
			dataMap[k] = strconv.Itoa(int(vv))
		default:
		}
	}
	return dataMap
}
