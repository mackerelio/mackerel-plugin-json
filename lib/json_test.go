package mpjson

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTraverseMap(t *testing.T) {
	var p JSONPlugin

	p.Prefix = "testprefix"
	p.ExcludeExp = regexp.MustCompile(`^$`)
	p.IncludeExp = regexp.MustCompile(``)

	bytes, _ := ioutil.ReadFile("testdata/jolokia.json")
	var content interface{}
	err := json.Unmarshal(bytes, &content)
	if err != nil {
		panic(err)
	}
	stat, err := p.traverseMap(content, []string{p.Prefix})

	assert.Nil(t, err)

	assert.EqualValues(t, 1073741824, stat[p.Prefix+".value.HeapMemoryUsage.init"])

	// A metric having null (nil) value shouldn't be contained.
	if _, ok := stat[p.Prefix+".value.ObjectPendingFinalizationCount"]; ok {
		fmt.Println(ok)
		t.Fatalf(p.Prefix + ".value.ObjectPendingFinalizationCount shouldn't exist.")
	}

	// Tests for slice handling
	// An object is a slice
	bytes, _ = ioutil.ReadFile("testdata/array.json")
	err = json.Unmarshal(bytes, &content)
	if err != nil {
		panic(err)
	}
	stat, err = p.traverseMap(content, []string{p.Prefix})

	assert.Nil(t, err)

	assert.EqualValues(t, 1, stat[p.Prefix+".0.count1"])
	assert.EqualValues(t, 3, stat[p.Prefix+".1.count1"])

	// Slices with in an object
	bytes, _ = ioutil.ReadFile("testdata/array_within.json")
	err = json.Unmarshal(bytes, &content)
	if err != nil {
		panic(err)
	}
	stat, err = p.traverseMap(content, []string{p.Prefix})

	assert.Nil(t, err)

	assert.EqualValues(t, 10, stat[p.Prefix+".0.count1"])
	assert.EqualValues(t, 30, stat[p.Prefix+".1.count1"])
}

func TestOutputMetric(t *testing.T) {
	var p JSONPlugin
	p.Prefix = "testprefix"

	p.ExcludeExp = regexp.MustCompile(`^$`)
	p.IncludeExp = regexp.MustCompile(``)

	// Should work if the value is float
	path, value := p.outputMetric("hoge.fuga.foo", 12345.67)
	assert.EqualValues(t, "hoge.fuga.foo", path)
	assert.EqualValues(t, 12345.67, value)

	// Should not work if the value is string
	path, value = p.outputMetric("hoge.fuga.foo", "boo")
	assert.EqualValues(t, "", path)
	assert.EqualValues(t, 0, value)

	// The output should be empty if ExcludeExp is specified and matches.
	p.ExcludeExp = regexp.MustCompile(`h??e`)
	path, value = p.outputMetric("hoge.fuga.foo", 12345.67)
	assert.EqualValues(t, "", path)
	assert.EqualValues(t, 0, value)
}

func TestCalcDiff(t *testing.T) {
	dir, err := ioutil.TempDir("", "makerel-plugin-json-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("MACKEREL_PLUGIN_WORKDIR", dir) // to separate tempfile for each test

	var p JSONPlugin
	p.Prefix = "testprefix"
	p.DiffExp = regexp.MustCompile(`foo`)
	p.ExcludeExp = regexp.MustCompile(`^$`)
	p.IncludeExp = regexp.MustCompile(``)

	ts := time.Now().Unix()
	{
		bytes, _ := ioutil.ReadFile("testdata/diff_before.json")
		var content interface{}
		err := json.Unmarshal(bytes, &content)
		if err != nil {
			panic(err)
		}
		stat, err := p.traverseMap(content, []string{p.Prefix})
		assert.Nil(t, err)
		stat, err = p.calcDiff(stat, ts)
		assert.EqualValues(t, 0, stat[p.Prefix+".foo_1"]) // at first
		assert.EqualValues(t, 0, stat[p.Prefix+".foo_2"]) // at first
		assert.EqualValues(t, 300, stat[p.Prefix+".bar"])
	}

	ts += 60
	{
		bytes, _ := ioutil.ReadFile("testdata/diff_after.json")
		var content interface{}
		err := json.Unmarshal(bytes, &content)
		if err != nil {
			panic(err)
		}
		stat, err := p.traverseMap(content, []string{p.Prefix})
		assert.Nil(t, err)
		stat, err = p.calcDiff(stat, ts)
		assert.EqualValues(t, 0, stat[p.Prefix+".foo_1"])   // reset
		assert.EqualValues(t, 100, stat[p.Prefix+".foo_2"]) // diff
		assert.EqualValues(t, 400, stat[p.Prefix+".bar"])
	}
}
