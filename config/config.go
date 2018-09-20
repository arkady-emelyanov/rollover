package config

import (
	"fmt"
	"strings"
	"time"
)

type ElasticSearch struct {
	Endpoints []string `json:"endpoints"`
}

type RolloverConditions struct {
	MaxDocs int    `json:"max_docs"`
	MaxAge  string `json:"max_age"`
}

type RolloverOptimize struct {
	MaxSegments int `json:"max_segments"`
}

type RolloverAlias struct {
	Alias      string             `json:"alias"`
	NewName    string             `json:"new_name"`
	Conditions RolloverConditions `json:"conditions"`
	Optimize   RolloverOptimize   `json:"optimize"`
}

// formatting new index name
func (r RolloverAlias) NewIndexName(dt time.Time) string {
	rp := strings.NewReplacer(
		`%Y`, fmt.Sprintf("%02d", dt.Year()),
		`%m`, fmt.Sprintf("%02d", dt.Month()),
		`%d`, fmt.Sprintf("%02d", dt.Day()),
		`%H`, fmt.Sprintf("%02d", dt.Hour()),
		`%M`, fmt.Sprintf("%02d", dt.Minute()),
		`%s`, fmt.Sprintf("%02d", dt.Second()),
	)
	return rp.Replace(r.NewName)
}

type Config struct {
	ElasticSearch ElasticSearch   `json:"elasticsearch"`
	Rollover      []RolloverAlias `json:"rollover"`
}
