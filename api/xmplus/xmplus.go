package xmplus

import (
    "bufio"
	"encoding/json"
	"fmt"
	"errors"
	"log"
	"regexp"
	"sync/atomic"
	"time"
	"sync"
	"os"
	"reflect"

	"github.com/bitly/go-simplejson"
	"github.com/go-resty/resty/v2"

	"github.com/XMPlusDev/XMPlus-NoRelay/api"
)

type APIClient struct {
	client        *resty.Client
	APIHost       string
	NodeID        int
	Key           string
	resp          atomic.Value
	eTags          map[string]string
	LastReportOnline   map[int]int
	access        sync.Mutex
	LocalRuleList []api.DetectRule
}

func New(apiConfig *api.Config) *APIClient {
	client := resty.New()
	client.SetRetryCount(3)
	if apiConfig.Timeout > 0 {
		client.SetTimeout(time.Duration(apiConfig.Timeout) * time.Second)
	} else {
		client.SetTimeout(5 * time.Second)
	}
	client.OnError(func(req *resty.Request, err error) {
		if v, ok := err.(*resty.ResponseError); ok {
			// v.Response contains the last response from the server
			// v.Err contains the original error
			log.Print(v.Err)
		}
	})
	client.SetBaseURL(apiConfig.APIHost)
	
	client.SetQueryParam("key", apiConfig.Key)
	
	localRuleList := readLocalRuleList(apiConfig.RuleListPath)
	
	apiClient := &APIClient{
		client:        client,
		NodeID:        apiConfig.NodeID,
		Key:           apiConfig.Key,
		APIHost:       apiConfig.APIHost,
		LastReportOnline:    make(map[int]int),
		eTags:         make(map[string]string),
		LocalRuleList:  localRuleList,
	}
	
	return apiClient
}

// readLocalRuleList reads the local rule list file
func readLocalRuleList(path string) (LocalRuleList []api.DetectRule) {

	LocalRuleList = make([]api.DetectRule, 0)
	if path != "" {
		// open the file
		file, err := os.Open(path)

		// handle errors while opening
		if err != nil {
			log.Printf("Error when opening file: %s", err)
			return LocalRuleList
		}

		fileScanner := bufio.NewScanner(file)

		// read line by line
		for fileScanner.Scan() {
			LocalRuleList = append(LocalRuleList, api.DetectRule{
				ID:      -1,
				Pattern: regexp.MustCompile(fileScanner.Text()),
			})
		}
		// handle first encountered error while reading
		if err := fileScanner.Err(); err != nil {
			log.Fatalf("Error while reading file: %s", err)
			return
		}

		file.Close()
	}

	return LocalRuleList
}

func (c *APIClient) Describe() api.ClientInfo {
	return api.ClientInfo{APIHost: c.APIHost, NodeID: c.NodeID, Key: c.Key}
}

func (c *APIClient) Debug() {
	c.client.SetDebug(true)
}

func (c *APIClient) assembleURL(path string) string {
	return c.APIHost + path
}

func (c *APIClient) parseResponse(res *resty.Response, path string, err error) (*simplejson.Json, error) {
	if err != nil {
		return nil, fmt.Errorf("request %s failed: %s", c.assembleURL(path), err)
	}

	if res.StatusCode() > 399 {
		body := res.Body()
		return nil, fmt.Errorf("request %s failed: %s, %s", c.assembleURL(path), string(body), err)
	}
	rtn, err := simplejson.NewJson(res.Body())
	if err != nil {
		return nil, fmt.Errorf("%s", res.String())
	}
	return rtn, nil
}

func (c *APIClient) GetNodeInfo() (nodeInfo *api.NodeInfo, err error) {
	server := new(serverConfig)
	path := fmt.Sprintf("/api/backend/server/%d", c.NodeID)
	res, err := c.client.R().
		ForceContentType("application/json").
		SetHeader("If-None-Match", c.eTags["server"]).
		Get(path)

	if res.StatusCode() == 304 {
		return nil, errors.New(api.NodeNotModified)
	}
	// update etag
	if res.Header().Get("Etag") != "" && res.Header().Get("Etag") != c.eTags["server"] {
		c.eTags["server"] = res.Header().Get("Etag")
	}
	
	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}
	
	b, _ := response.Encode()
	json.Unmarshal(b, server)

	if server.Port == 0 {
		return nil, errors.New("server port must > 0")
	}
	
	if server.Type == "" {
		return nil, fmt.Errorf("server Type cannot be %s", server.Type)
	}
	
	c.resp.Store(server)
	
	//version := c.resp.Load().(*serverConfig).version
	//if(version < "v20231001"){
	//	return nil, errors.New("Update your XMPlus v2 panel to latest version")
	//}
	
	nodeInfo, err = c.parseNodeResponse(server)
	if err != nil {
		return nil, fmt.Errorf("Parse node info failed: %s, \nError: %v", res.String(), err)
	}
	return nodeInfo, nil
}


func (c *APIClient) GetServiceList() (ServiceList *[]api.ServiceInfo, err error) {
	path := fmt.Sprintf("/api/backend/service/%d", c.NodeID)
	res, err := c.client.R().
		SetHeader("If-None-Match", c.eTags["services"]).
		ForceContentType("application/json").
		Get(path)
	
	if res.StatusCode() == 304 {
		return nil, errors.New(api.ServiceNotModified)
	}
	// update etag
	if res.Header().Get("Etag") != "" && res.Header().Get("Etag") != c.eTags["services"] {
		c.eTags["services"] = res.Header().Get("Etag")
	}

	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}
	
	services := new([]Service)
	
	b, _ := response.Get("services").Encode()
	json.Unmarshal(b, services)
	
	if err := json.Unmarshal(b, services); err != nil {
		return nil, fmt.Errorf("Unmarshal %s failed: %s", reflect.TypeOf(services), err)
	}
	
	serviceList, err := c.ParseUserListResponse(services)
	if err != nil {
		res, _ := json.Marshal(services)
		return nil, fmt.Errorf("Parse user list failed: %s", string(res))
	}

	return serviceList, nil
}

func (c *APIClient) ParseUserListResponse(serviceResponse *[]Service) (*[]api.ServiceInfo, error) {
	c.access.Lock()
	defer func() {
		c.LastReportOnline = make(map[int]int)
		c.access.Unlock()
	}()	
	
	var deviceLimit, onlineipcount, ipcount int = 0, 0, 0
	
	serviceList := []api.ServiceInfo{}
	
	for _, service := range *serviceResponse {
		deviceLimit = service.Iplimit
		ipcount = service.Ipcount
		
		if deviceLimit > 0 && ipcount > 0 {
			lastOnline := 0
			if v, ok := c.LastReportOnline[service.Id]; ok {
				lastOnline = v
			}
			if onlineipcount = deviceLimit - ipcount + lastOnline; onlineipcount > 0 {
				deviceLimit = onlineipcount
			} else if lastOnline > 0 {
				deviceLimit = lastOnline
			} else {
				continue
			}
		}

		serviceList = append(serviceList, api.ServiceInfo{
			UID:  service.Id,
			UUID: service.Uuid,
			Email: service.Email,
			Passwd: service.Uuid,
			DeviceLimit: deviceLimit,
			SpeedLimit:  uint64(service.Speedlimit * 1000000 / 8),
		})
	}

	return &serviceList, nil
}


func (c *APIClient) GetNodeRule() (*[]api.DetectRule, error) {
	ruleList := c.LocalRuleList
	
	routes := c.resp.Load().(*serverConfig).Routes
	
	//Rules := len(routes)
	//detects := make([]api.DetectRule, Rules)
	
	for i := range routes {
		ruleListItem := api.DetectRule{
			ID:      routes[i].Id,
			Pattern: regexp.MustCompile(routes[i].Regex),
		}
		//detects[i] = ruleListItem
		ruleList = append(ruleList, ruleListItem)
	}

	//return &detects, nil
	return &ruleList, nil
}


func (c *APIClient) ReportServiceTraffic(serviceTraffic *[]api.ServiceTraffic) error {
	path := fmt.Sprintf("/api/backend/service/traffic/%d", c.NodeID)

	data := make([]ServiceTraffic, len(*serviceTraffic))	
	for i, traffic := range *serviceTraffic {
		data[i] = ServiceTraffic{
			UID:      traffic.UID,
			Upload:   traffic.Upload,
			Download: traffic.Download,
		}
	}
	postData := &PostData{Data: data}
	res, err := c.client.R().
		SetBody(postData).
		ForceContentType("application/json").
		Post(path)
	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}

	return nil
}

func (c *APIClient) ReportNodeOnlineIPs(onlineServiceList *[]api.OnlineIP) error {
	c.access.Lock()
	defer c.access.Unlock()

	reportOnline := make(map[int]int)
	data := make([]OnlineIP, len(*onlineServiceList))
	for i, service := range *onlineServiceList {
		data[i] = OnlineIP{UID: service.UID, IP: service.IP}
		if _, ok := reportOnline[service.UID]; ok {
			reportOnline[service.UID]++
		} else {
			reportOnline[service.UID] = 1
		}
	}
	c.LastReportOnline = reportOnline // Update LastReportOnline

	postData := &PostData{Data: data}
	path := fmt.Sprintf("/api/backend/service/online/%d", c.NodeID)
	res, err := c.client.R().
		SetBody(postData).
		SetResult(&Response{}).
		ForceContentType("application/json").
		Post(path)

	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}

	return nil
}

func (c *APIClient) parseNodeResponse(s *serverConfig) (*api.NodeInfo, error) {
	var (
		TLSType  = "none"
		path, host, quic_security, quic_key, serviceName, seed, htype, Alpn, Dest, PrivateKey, MinClientVer, MaxClientVer string
		header                  json.RawMessage
		enableTLS, congestion ,RejectUnknownSni, AllowInsecure, Show  bool
		alterID                 uint16 = 0
		MaxTimeDiff,ProxyProtocol  uint64 = 0, 0	
		ServerNames,  ShortIds []string
	)
	
	NodeType := s.Type

	if s.SecuritySettings.Alpn != "" {
		Alpn = s.SecuritySettings.Alpn
	}
	
	Flow := "xtls-rprx-direct"
	
	if s.NetworkSettings.Flow == "xtls-rprx-vision" || s.NetworkSettings.Flow == "xtls-rprx-vision-udp443"{
		Flow = s.NetworkSettings.Flow
	}
	
	TLSType = s.Security
	
	if TLSType == "tls" {
		enableTLS = true
		RejectUnknownSni = s.SecuritySettings.RejectUnknownSni
        AllowInsecure = s.SecuritySettings.AllowInsecure

		if s.SecuritySettings.ServerName == "" {
			return nil, fmt.Errorf("TLS certificate domain (ServerName) is empty: %s",  s.SecuritySettings.ServerName)
		}
	}
	
	if TLSType == "reality" {
		Dest = s.SecuritySettings.Dest
		Show = s.SecuritySettings.Show
		PrivateKey = s.SecuritySettings.PrivateKey
		MinClientVer = s.SecuritySettings.MinClientVer
		MaxClientVer = s.SecuritySettings.MaxClientVer
		MaxTimeDiff = uint64(s.SecuritySettings.MaxTimeDiff)
		ShortIds = s.SecuritySettings.ShortIds
		ServerNames = s.SecuritySettings.ServerNames
		ProxyProtocol = uint64(s.SecuritySettings.ProxyProtocol)
	}

	transportProtocol := s.NetworkSettings.Transport

	switch transportProtocol {
		case "ws":
			path = s.NetworkSettings.Path
			if headerHost, err := s.NetworkSettings.Headers.MarshalJSON(); err != nil {
					return nil, err
			} else {
				w, _ := simplejson.NewJson(headerHost)
				host = w.Get("Host").MustString()
			}
		case "h2":
			path = s.NetworkSettings.Path
			host = s.NetworkSettings.Host
		case "grpc":
			serviceName = s.NetworkSettings.ServiceName
		case "tcp":
			if httpHeader, err := s.NetworkSettings.Header.MarshalJSON(); err != nil {
					return nil, err
			} else {
				t, _ := simplejson.NewJson(httpHeader)
				htype := t.Get("type").MustString()
				if htype == "http" {
					path = t.Get("request").Get("path").MustString()
					header, _ = json.Marshal(map[string]any{
						"type": "http",
						"request": map[string]any{
							"path": path,
						}})
				}else{
					header, _ = json.Marshal(map[string]any{
						"type": "none",
						})
				}
			}
		case "quic":
			quic_key = s.NetworkSettings.Quickey
			quic_security = s.NetworkSettings.QuicSecurity
			if headerType, err := s.NetworkSettings.Header.MarshalJSON(); err != nil {
					return nil, err
			} else {
				h, _ := simplejson.NewJson(headerType)
				htype = h.Get("type").MustString()
			}
			header, _ = json.Marshal(map[string]any{
					"type": htype,
				})
		case "kcp":
			seed = s.NetworkSettings.Seed
			congestion = s.NetworkSettings.Congestion
			if headerType, err := s.NetworkSettings.Header.MarshalJSON(); err != nil {
					return nil, err
			} else {
				k, _ := simplejson.NewJson(headerType)
				htype = k.Get("type").MustString()
			}
			header, _ = json.Marshal(map[string]any{
					"type": htype,
				})		
	}
	
	if NodeType == "Shadowsocks"  && (transportProtocol == "ws" || transportProtocol == "grpc" || transportProtocol == "quic") {
		NodeType = "Shadowsocks-Plugin"
	}
	
	nodeInfo := &api.NodeInfo{
		NodeType:          NodeType,
		NodeID:            c.NodeID,
		Port:              uint32(s.Port),
		Transport:         transportProtocol,
		EnableTLS:         enableTLS,
		TLSType:           TLSType,
		Path:              path,
		Host:              host,
		ServiceName:       serviceName,
		Flow:              Flow,
		Header:            header,
		AlterID:           alterID,
		Seed:              seed,
		Congestion:        congestion,
		Sniffing:          s.Sniffing,
		RejectUnknownSNI:  RejectUnknownSni,
		Fingerprint:       s.SecuritySettings.Fingerprint, 
		Quic_security:     quic_security,
		Alpn:              Alpn,
		Quic_key:          quic_key,
		CypherMethod:      s.Cipher,
		Address:           s.Address, 
		AllowInsecure:     AllowInsecure,
		ListenIP:          s.Listenip, 
		ProxyProtocol:     s.NetworkSettings.ProxyProtocol,
		CertMode:          s.Certmode,
		CertDomain:        s.SecuritySettings.ServerName,
		ServerKey:         s.ServerKey,
		SpeedLimit:        uint64(s.Speedlimit * 1000000 / 8),
		SendIP:            s.SendThrough,
		Dest:              Dest,
		Show:              Show,
		ServerNames:       ServerNames,  
		PrivateKey:        PrivateKey,
		ShortIds:          ShortIds,
		MinClientVer:      MinClientVer,
		MaxClientVer:      MaxClientVer,
		MaxTimeDiff:       MaxTimeDiff,
		Xver:              ProxyProtocol,
	}
	return nodeInfo, nil
}
