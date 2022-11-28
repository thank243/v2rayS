package v2board

import (
	"sync"

	"github.com/bitly/go-simplejson"
	"github.com/go-resty/resty/v2"

	"github.com/thank243/v2rayS/api"
)

const apiSuffix = "/api/v1/server"

// APIClient create an api client to the panel.
type APIClient struct {
	client        *resty.Client
	APIHost       string
	NodeID        int
	Key           string
	NodeType      string
	SpeedLimit    float64
	DeviceLimit   int
	LocalRuleList []api.DetectRule
	ConfigResp    *simplejson.Json
	access        sync.Mutex
}

type UserTraffic struct {
	UID      int   `json:"user_id"`
	Upload   int64 `json:"u"`
	Download int64 `json:"d"`
}
