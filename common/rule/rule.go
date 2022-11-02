// Package rule is to control the audit rule behaviors
package rule

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/deckarep/golang-set"

	"github.com/thank243/v2rayS/api"
)

type Manager struct {
	InboundRule         *sync.Map // Key: Tag, Value: []api.DetectRule
	InboundDetectResult *sync.Map // key: Tag, Value: mapset.NewSet []api.DetectResult
}

func New() *Manager {
	return &Manager{
		InboundRule:         new(sync.Map),
		InboundDetectResult: new(sync.Map),
	}
}

func (m *Manager) UpdateRule(tag string, newRuleList []api.DetectRule) error {
	if value, ok := m.InboundRule.LoadOrStore(tag, newRuleList); ok {
		oldRuleList := value.([]api.DetectRule)
		if !reflect.DeepEqual(oldRuleList, newRuleList) {
			m.InboundRule.Store(tag, newRuleList)
		}
	}
	return nil
}

func (m *Manager) GetDetectResult(tag string) (*[]api.DetectResult, error) {
	detectResult := make([]api.DetectResult, 0)
	if value, ok := m.InboundDetectResult.LoadAndDelete(tag); ok {
		resultSet := value.(mapset.Set)
		it := resultSet.Iterator()
		for result := range it.C {
			detectResult = append(detectResult, result.(api.DetectResult))
		}
	}
	return &detectResult, nil
}

func (m *Manager) Detect(tag string, destination string, email string) (reject bool) {
	reject = false
	var hitRuleID = -1
	// If we have some rule for this inbound
	if value, ok := m.InboundRule.Load(tag); ok {
		ruleList := value.([]api.DetectRule)
		for _, r := range ruleList {
			if r.Pattern.Match([]byte(destination)) {
				hitRuleID = r.ID
				reject = true
				break
			}
		}
		// If we hit some rule
		if reject && hitRuleID != -1 {
			l := strings.Split(email, "|")
			uid, err := strconv.Atoi(l[len(l)-1])
			if err != nil {
				newError(fmt.Sprintf("Record illegal behavior failed! Cannot find user's uid: %s", email)).AtDebug().WriteToLog()
				return reject
			}
			newSet := mapset.NewSetWith(api.DetectResult{UID: uid, RuleID: hitRuleID})
			// If there are any hit history
			if v, ok := m.InboundDetectResult.LoadOrStore(tag, newSet); ok {
				resultSet := v.(mapset.Set)
				// If this is a new record
				if resultSet.Add(api.DetectResult{UID: uid, RuleID: hitRuleID}) {
					m.InboundDetectResult.Store(tag, resultSet)
				}
			}
		}
	}
	return reject
}
