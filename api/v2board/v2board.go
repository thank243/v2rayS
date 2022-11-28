package v2board

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/go-resty/resty/v2"

	"github.com/thank243/v2rayS/api"
)

// New create an api instance
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
			newError(v.Err).AtError().WriteToLog()
		}
	})

	client.SetBaseURL(apiConfig.APIHost)
	// Create Key for each requests
	client.SetQueryParams(map[string]string{
		"node_id": strconv.Itoa(apiConfig.NodeID),
		"token":   apiConfig.Key,
	})
	// Read local rule list
	localRuleList := readLocalRuleList(apiConfig.RuleListPath)
	apiClient := &APIClient{
		client:        client,
		NodeID:        apiConfig.NodeID,
		Key:           apiConfig.Key,
		APIHost:       apiConfig.APIHost,
		NodeType:      apiConfig.NodeType,
		SpeedLimit:    apiConfig.SpeedLimit,
		DeviceLimit:   apiConfig.DeviceLimit,
		LocalRuleList: localRuleList,
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

// Describe return a description of the client
func (c *APIClient) Describe() api.ClientInfo {
	return api.ClientInfo{APIHost: c.APIHost, NodeID: c.NodeID, Key: c.Key, NodeType: c.NodeType}
}

// Debug set the client debug for client
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

	if res.StatusCode() > 400 {
		body := res.Body()
		return nil, fmt.Errorf("request %s failed: %s, %s", c.assembleURL(path), string(body), err)
	}
	rtn, err := simplejson.NewJson(res.Body())
	if err != nil {
		return nil, fmt.Errorf("return %s invalid", res.String())
	}
	return rtn, nil
}

// GetNodeInfo will pull NodeInfo Config from v2board
func (c *APIClient) GetNodeInfo() (nodeInfo *api.NodeInfo, err error) {
	var path string

	switch c.NodeType {
	case "V2ray":
		path = apiSuffix + "/Deepbwork/config"
	case "Trojan":
		path = apiSuffix + "/TrojanTidalab/config"
	case "Shadowsocks":
		if nodeInfo, err = c.ParseSSNodeResponse(); err == nil {
			return nodeInfo, nil
		} else {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported Node type: %s", c.NodeType)
	}
	res, err := c.client.R().
		SetQueryParam("local_port", "1").
		ForceContentType("application/json").
		Get(path)

	response, err := c.parseResponse(res, path, err)
	c.access.Lock()
	defer c.access.Unlock()
	c.ConfigResp = response
	if err != nil {
		return nil, err
	}

	switch c.NodeType {
	case "V2ray":
		nodeInfo, err = c.ParseV2rayNodeResponse(response)
	case "Trojan":
		nodeInfo, err = c.ParseTrojanNodeResponse(response)
	case "Shadowsocks":
		nodeInfo, err = c.ParseSSNodeResponse()
	default:
		return nil, fmt.Errorf("unsupported Node type: %s", c.NodeType)
	}

	if err != nil {
		res, _ := response.MarshalJSON()
		return nil, fmt.Errorf("Parse node info failed: %s, \nError: %s", string(res), err)
	}

	return nodeInfo, nil
}

// GetUserList will pull user form v2board
func (c *APIClient) GetUserList() (UserList *[]api.UserInfo, err error) {
	var path string

	switch c.NodeType {
	case "V2ray":
		path = apiSuffix + "/Deepbwork/user"
	case "Trojan":
		path = apiSuffix + "/TrojanTidalab/user"
	case "Shadowsocks":
		path = apiSuffix + "/ShadowsocksTidalab/user"
	default:
		return nil, fmt.Errorf("unsupported Node type: %s", c.NodeType)
	}
	res, err := c.client.R().
		ForceContentType("application/json").
		Get(path)

	response, err := c.parseResponse(res, path, err)
	if err != nil {
		return nil, err
	}
	numOfUsers := len(response.Get("data").MustArray())
	userList := make([]api.UserInfo, numOfUsers)
	for i := 0; i < numOfUsers; i++ {
		user := api.UserInfo{}
		user.UID = response.Get("data").GetIndex(i).Get("id").MustInt()
		user.SpeedLimit = uint64(c.SpeedLimit * 1000000 / 8)
		user.DeviceLimit = c.DeviceLimit
		switch c.NodeType {
		case "Shadowsocks":
			user.Email = response.Get("data").GetIndex(i).Get("secret").MustString()
			user.Passwd = response.Get("data").GetIndex(i).Get("secret").MustString()
			user.Method = response.Get("data").GetIndex(i).Get("cipher").MustString()
			user.Port = uint32(response.Get("data").GetIndex(i).Get("port").MustUint64())
		case "Trojan":
			user.UUID = response.Get("data").GetIndex(i).Get("trojan_user").Get("password").MustString()
			user.Email = response.Get("data").GetIndex(i).Get("trojan_user").Get("password").MustString()
		case "V2ray":
			user.UUID = response.Get("data").GetIndex(i).Get("v2ray_user").Get("uuid").MustString()
			user.Email = response.Get("data").GetIndex(i).Get("v2ray_user").Get("email").MustString()
			user.AlterID = uint16(response.Get("data").GetIndex(i).Get("v2ray_user").Get("alter_id").MustUint64())
		}
		userList[i] = user
	}

	return &userList, nil
}

// ReportUserTraffic reports the user traffic
func (c *APIClient) ReportUserTraffic(userTraffic *[]api.UserTraffic) error {
	var path string
	switch c.NodeType {
	case "V2ray":
		path = apiSuffix + "/Deepbwork/submit"
	case "Trojan":
		path = apiSuffix + "/TrojanTidalab/submit"
	case "Shadowsocks":
		path = apiSuffix + "/ShadowsocksTidalab/submit"
	}

	data := make([]UserTraffic, len(*userTraffic))
	for i, traffic := range *userTraffic {
		data[i] = UserTraffic{
			UID:      traffic.UID,
			Upload:   traffic.Upload,
			Download: traffic.Download}
	}

	res, err := c.client.R().
		SetQueryParam("node_id", strconv.Itoa(c.NodeID)).
		SetBody(data).
		ForceContentType("application/json").
		Post(path)
	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}
	return nil
}

// GetNodeRule implements the API interface
func (c *APIClient) GetNodeRule() (*[]api.DetectRule, error) {
	ruleList := c.LocalRuleList
	if c.NodeType != "V2ray" {
		return &ruleList, nil
	}

	// V2board only support the rule for v2ray
	// fix: reuse config response
	c.access.Lock()
	defer c.access.Unlock()
	ruleListResponse := c.ConfigResp.Get("routing").Get("rules").GetIndex(1).Get("domain").MustStringArray()
	for i, rule := range ruleListResponse {
		rule = strings.TrimPrefix(rule, "regexp:")
		ruleListItem := api.DetectRule{
			ID:      i,
			Pattern: regexp.MustCompile(rule),
		}
		ruleList = append(ruleList, ruleListItem)
	}
	return &ruleList, nil
}

// ReportNodeStatus implements the API interface
func (c *APIClient) ReportNodeStatus(*api.NodeStatus) (err error) {
	return nil
}

// ReportNodeOnlineUsers implements the API interface
func (c *APIClient) ReportNodeOnlineUsers(*[]api.OnlineUser) error {
	return nil
}

// ReportIllegal implements the API interface
func (c *APIClient) ReportIllegal(*[]api.DetectResult) error {
	return nil
}

// ParseTrojanNodeResponse parse the response for the given nodeInfo format
func (c *APIClient) ParseTrojanNodeResponse(nodeInfoResponse *simplejson.Json) (*api.NodeInfo, error) {
	var TLSType = "tls"

	port := uint32(nodeInfoResponse.Get("local_port").MustUint64())
	host := nodeInfoResponse.Get("ssl").Get("sni").MustString()

	// Create GeneralNodeInfo
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		TransportProtocol: "tcp",
		EnableTLS:         true,
		TLSType:           TLSType,
		Host:              host,
	}
	return nodeInfo, nil
}

// ParseSSNodeResponse parse the response for the given nodeInfo format
func (c *APIClient) ParseSSNodeResponse() (*api.NodeInfo, error) {
	var port uint32
	var method string
	userInfo, err := c.GetUserList()
	if err != nil {
		return nil, err
	}
	if len(*userInfo) > 0 {
		port = (*userInfo)[0].Port
		method = (*userInfo)[0].Method
	}

	// Create GeneralNodeInfo
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		TransportProtocol: "tcp",
		CypherMethod:      method,
	}

	return nodeInfo, nil
}

// ParseV2rayNodeResponse parse the response for the given nodeInfo format
func (c *APIClient) ParseV2rayNodeResponse(nodeInfoResponse *simplejson.Json) (*api.NodeInfo, error) {
	var TLSType = "tls"
	var path, host, serviceName string
	var header json.RawMessage
	var enableTLS bool
	var alterID uint16 = 0

	inboundInfo := simplejson.New()
	if buf, ok := nodeInfoResponse.CheckGet("inbound"); ok {
		inboundInfo = buf
		// Compatible with v2board 1.5.5-dev
	} else if buf, ok = nodeInfoResponse.CheckGet("inbounds"); ok {
		marshalByte, _ := json.Marshal(buf.MustArray()[0].(map[string]interface{}))
		inboundInfo, _ = simplejson.NewJson(marshalByte)
	} else {
		return nil, fmt.Errorf("unable to find inbound(s) in the nodeInfo")
	}

	port := uint32(inboundInfo.Get("port").MustUint64())
	transportProtocol := inboundInfo.Get("streamSettings").Get("network").MustString()

	switch transportProtocol {
	case "ws":
		path = inboundInfo.Get("streamSettings").Get("wsSettings").Get("path").MustString()
		host = inboundInfo.Get("streamSettings").Get("wsSettings").Get("headers").Get("Host").MustString()
	case "grpc":
		if data, ok := inboundInfo.Get("streamSettings").Get("grpcSettings").CheckGet("serviceName"); ok {
			serviceName = data.MustString()
		}
	case "tcp":
		if data, ok := inboundInfo.Get("streamSettings").Get("tcpSettings").CheckGet("header"); ok {
			if httpHeader, err := data.MarshalJSON(); err != nil {
				return nil, err
			} else {
				header = httpHeader
			}
		}

	}
	if inboundInfo.Get("streamSettings").Get("security").MustString() == "tls" {
		enableTLS = true
	} else {
		enableTLS = false
	}

	// Create GeneralNodeInfo
	// AlterID will be updated after next sync
	nodeInfo := &api.NodeInfo{
		NodeType:          c.NodeType,
		NodeID:            c.NodeID,
		Port:              port,
		AlterID:           alterID,
		TransportProtocol: transportProtocol,
		EnableTLS:         enableTLS,
		TLSType:           TLSType,
		Path:              path,
		Host:              host,
		ServiceName:       serviceName,
		Header:            header,
	}
	return nodeInfo, nil
}
