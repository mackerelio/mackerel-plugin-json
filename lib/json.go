package mpjson

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mackerelio/golib/pluginutil"
)

// JSONPlugin plugin for JSON
type JSONPlugin struct {
	Target             string
	Tempfile           string
	URL                string
	Prefix             string
	InsecureSkipVerify bool
	ShowOnlyNum        bool
	Stdin              bool
	ExcludeExp         *regexp.Regexp
	IncludeExp         *regexp.Regexp
	DiffExp            *regexp.Regexp
}

type Values struct {
	LastTimestamp int64              `json:"last_timestamp"`
	Metrics       map[string]float64 `json:"metrics"`
}

func (p JSONPlugin) traverseMap(content interface{}, path []string) (map[string]float64, error) {
	stat := make(map[string]float64)
	if reflect.TypeOf(content).Kind() == reflect.Slice {
		for i, c := range content.([]interface{}) {
			ts, _ := p.traverseMap(c, append(path, strconv.Itoa(i)))
			for tk, tv := range ts {
				stat[tk] = tv
			}
		}
	} else {
		for k, v := range content.(map[string]interface{}) {
			if v != nil {
				if reflect.TypeOf(v).Kind() == reflect.Map {
					ts, _ := p.traverseMap(v, append(path, k))
					for tk, tv := range ts {
						stat[tk] = tv
					}
				} else if reflect.TypeOf(v).Kind() == reflect.Slice {
					for i, c := range v.([]interface{}) {
						ts, _ := p.traverseMap(c, append(path, strconv.Itoa(i)))
						for tk, tv := range ts {
							stat[tk] = tv
						}
					}
				} else {
					tk, tv := p.outputMetric(strings.Join(append(path, k), "."), v)
					if tk != "" {
						stat[tk] = tv
					}
				}
			}
		}
	}

	return stat, nil
}

func (p JSONPlugin) outputMetric(path string, value interface{}) (string, float64) {
	if p.IncludeExp.MatchString(path) && !p.ExcludeExp.MatchString(path) {
		if reflect.TypeOf(value).Kind() == reflect.Float64 {
			return path, value.(float64)
		}
	}

	return "", 0
}

// FetchMetrics interface for mackerel-plugin
func (p JSONPlugin) FetchMetrics() (map[string]float64, error) {

	var bytes []byte
	var err error

	if p.URL != "" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: p.InsecureSkipVerify},
		}
		client := &http.Client{Transport: tr}
		response, err := client.Get(p.URL)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		bytes, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
	}

	if p.Stdin {
		bytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
	}

	var content interface{}
	if err := json.Unmarshal(bytes, &content); err != nil {
		return nil, err
	}

	return p.traverseMap(content, []string{p.Prefix})
}

func (p JSONPlugin) calcDiff(metrics map[string]float64, ts int64) (map[string]float64, error) {
	lastValues := p.fetchLastValues()
	diffTime := float64(ts - lastValues.LastTimestamp)
	if lastValues.LastTimestamp != 0 && diffTime > 600 {
		log.Println("Too long duration")
		diffTime = 0 // do not calc diff
	}

	err := p.saveValues(Values{
		LastTimestamp: ts,
		Metrics:       metrics,
	})
	if err != nil {
		log.Printf("Couldn't save values: %s", err)
	}

	for path, value := range metrics {
		if !p.DiffExp.MatchString(path) {
			continue
		}
		if lastValue, ok := lastValues.Metrics[path]; ok && diffTime > 0 {
			diff := (value - lastValue) * 60 / diffTime
			if diff >= 0 {
				metrics[path] = diff
			} else {
				log.Printf("Counter %s seems to be reset.", path)
				metrics[path] = 0
			}
		} else {
			metrics[path] = 0 // cannot calc diff
		}
	}

	return metrics, nil
}

func (p JSONPlugin) tempfilename() string {
	// tempfiles have an unique name for each configuration.
	filename := fmt.Sprintf("mackerel-plugin-json.%x", sha1.Sum([]byte(p.Prefix+p.URL)))
	return filepath.Join(pluginutil.PluginWorkDir(), filename)
}

func (p JSONPlugin) fetchLastValues() Values {
	v := Values{
		LastTimestamp: 0,
		Metrics:       make(map[string]float64),
	}
	filename := p.tempfilename()
	f, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Couldn't fetch last values: %s", err)
		}
		return v
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&v); err != nil {
		// invalid file format. try to delete
		os.Remove(filename)
	}
	return v
}

func (p JSONPlugin) saveValues(v Values) error {
	filename := p.tempfilename()
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(v)
}

// Do do doo
func Do() {
	url := flag.String("url", "", "URL to get a JSON")
	stdin := flag.Bool("stdin", false, "Receive JSON from STDIN")
	prefix := flag.String("prefix", "custom", "Prefix for metric names")
	insecure := flag.Bool("insecure", false, "Skip certificate verifications")
	exclude := flag.String("exclude", `^$`, "Exclude metrics that matches the expression")
	include := flag.String("include", ``, "Output metrics that matches the expression")
	diff := flag.String("diff", ``, "Calculate difference of metrics that matches the expression")
	flag.Parse()

	if (*url == "") && (*stdin == false) {
		fmt.Println("-url or -stdin are mandatory")
		os.Exit(1)
	}

	if (*url != "") && (*stdin == true) {
		fmt.Println("-url and -stdin are exclusive")
		os.Exit(1)
	}

	var jsonplugin JSONPlugin

	jsonplugin.URL = *url
	jsonplugin.Stdin = *stdin
	jsonplugin.Prefix = *prefix
	jsonplugin.InsecureSkipVerify = *insecure
	var err error
	jsonplugin.ExcludeExp, err = regexp.Compile(*exclude)
	if err != nil {
		fmt.Printf("exclude expression is invalid: %s", err)
		os.Exit(1)
	}
	jsonplugin.IncludeExp, err = regexp.Compile(*include)
	if err != nil {
		fmt.Printf("include expression is invalid: %s", err)
		os.Exit(1)
	}
	if *diff != "" {
		jsonplugin.DiffExp, err = regexp.Compile(*diff)
		if err != nil {
			fmt.Printf("diff expression is invalid: %s", err)
			os.Exit(1)
		}
	}

	metrics, _ := jsonplugin.FetchMetrics()

	ts := time.Now().Unix()
	if jsonplugin.DiffExp != nil {
		metrics, _ = jsonplugin.calcDiff(metrics, ts)
	}
	for k, v := range metrics {
		fmt.Printf("%s\t%f\t%d\n", k, v, ts)
	}
}
