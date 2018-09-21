package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/arkady-emelyanov/rollover/config"
	"github.com/ghodss/yaml"
	"gopkg.in/olivere/elastic.v6"
)

type rolloverFn func(s *rolloverSt) rolloverFn

type rolloverSt struct {
	client *elastic.Client

	rule config.RolloverAlias
	ctx  context.Context
	err  error

	newIndexName string // index receives writes right now
	oldIndexName string // rolled index, read-only
}

var configFile string

// init and parse flags.
func init() {
	flag.StringVar(&configFile, "config", "", "/path/to/config.yml")
	flag.Parse()
}

// FSM method: Make index read-only, by updating settings and
// setting up `index.blocks.write` property to true.
func doMakeReadOnly(st *rolloverSt) rolloverFn {
	log.Printf("Index: %s, making read-only", st.oldIndexName)
	_, err := st.client.IndexPutSettings(st.oldIndexName).
		BodyString(`{"index.blocks.write": true}`).Do(st.ctx)

	if err != nil {
		st.err = err
		return nil
	}

	log.Printf("Index: %s, is now read-only", st.oldIndexName)
	return doFlush
}

// FSM method: Flush all possible `refresh_interval` buffers, by
// forcing flush operation.
func doFlush(st *rolloverSt) rolloverFn {
	log.Printf("Index: %s, requesting flush", st.oldIndexName)
	_, err := st.client.Flush(st.oldIndexName).
		Force(true).Do(st.ctx)

	if err != nil {
		st.err = err
		return nil
	}

	log.Printf("Index: %s, flushed", st.oldIndexName)
	return doForceMerge
}

// FSM method: Perform force merge operation. Merging, reduces number
// of segments of read-only index.
func doForceMerge(st *rolloverSt) rolloverFn {
	n := st.rule.Optimize.MaxSegments
	if n < 1 {
		n = 1
	}

	tt := time.Now()
	log.Printf("Index: %s, forcemerge, max_num_segments=%d",
		st.oldIndexName, n)

	_, err := st.client.Forcemerge(st.oldIndexName).MaxNumSegments(n).Do(st.ctx)
	if err != nil {
		st.err = err
		return nil
	}

	log.Printf("Index: %s, forcemerged, operation took: %s",
		st.oldIndexName, time.Now().Sub(tt))
	return nil
}

// FSM method: Request rollover and check response, if
// rolled over, proceed with optimization steps.
func doRollover(st *rolloverSt) rolloverFn {
	log.Printf("Alias: %s, checking...", st.rule.Alias)

	dt := time.Now()
	ri := st.rule.NewIndexName(dt)
	log.Printf("Alias: %s, performing rollover to index: '%s'",
		st.rule.Alias, ri)

	ris := st.client.RolloverIndex(st.rule.Alias).
		NewIndex(ri)

	hasCondition := false
	if st.rule.Conditions.MaxDocs > 0 {
		log.Printf(
			"Alias: %s, adding rule condition: max_docs=%d",
			st.rule.Alias, st.rule.Conditions.MaxDocs)

		ris = ris.AddCondition("max_docs", st.rule.Conditions.MaxDocs)
		hasCondition = true
	}
	if st.rule.Conditions.MaxAge != "" {
		log.Printf(
			"Alias: %s, adding rule condition: max_age=%s",
			st.rule.Alias, st.rule.Conditions.MaxAge)

		ris = ris.AddCondition("max_age", st.rule.Conditions.MaxAge)
		hasCondition = true
	}

	if !hasCondition {
		log.Printf("Alias: %s, no rule condition defined, skipping...",
			st.rule.Alias)
		return nil
	}

	res, err := ris.Do(st.ctx)
	if err != nil {
		log.Printf(
			"Alias: %s, error during rule request: %#v",
			st.rule.Alias,
			err,
		)
		return nil
	}

	if res.RolledOver {
		st.newIndexName = res.NewIndex
		st.oldIndexName = res.OldIndex
		log.Printf("Alias: %s, rolled over => new index: %s, old index: %s",
			st.rule.Alias, st.oldIndexName, st.newIndexName)
		return doMakeReadOnly

	}

	log.Printf("Alias: %s, no condition matched, skipping...", st.rule.Alias)
	return nil
}

// Load and parse configuration.
func loadConfig(f string) (*config.Config, error) {
	h, err := os.Open(f)
	if err != nil {
		return nil, err
	}

	defer h.Close()
	b, err := ioutil.ReadAll(h)
	if err != nil {
		return nil, err
	}

	cfg := &config.Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// rollover fsm
func rollover(r config.RolloverAlias, client *elastic.Client) {
	log.Printf("Alias: %s, checking rollover", r.Alias)

	st := &rolloverSt{
		client: client,
		ctx:    context.Background(),
		rule:   r,
	}

	// perform fsm propagation
	for handler := doRollover; handler != nil; {
		handler = handler(st)
	}

	if st.err != nil {
		log.Printf("Index: %s, errored: %#v", st.oldIndexName, st.err)
		return
	}
}

func main() {
	if configFile == "" {
		log.Fatalf("no configuration file provided, -config=/path/to/config.yml")
	}

	cfg, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to parse configuration: %#v", err)
	}
	if len(cfg.ElasticSearch.Endpoints) == 0 {
		log.Fatal("No ElasticSearch endpoints configured, quiting...")
	}

	client, err := elastic.NewClient(
		elastic.SetURL(cfg.ElasticSearch.Endpoints...),
		elastic.SetSniff(false), // disable discovery, use only provided endpoints
	)
	if err != nil {
		log.Fatalf("ElasticSearch connection error: %#v", err)
	}

	defer client.Stop()
	for {
		log.Println("main: iteration start")
		for _, r := range cfg.Rollover {
			go rollover(r, client)
		}

		log.Println("main: iteration done, sleeping...")
		<-time.After(time.Minute * 5)
	}
}
