// Copyright 2022 Security Onion Solutions. All rights reserved.
//
// This program is distributed under the terms of version 2 of the
// GNU General Public License.  See LICENSE for further details.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

package analyze

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/security-onion-solutions/securityonion-soc/model"
	"github.com/stretchr/testify/assert"
)

const TMP_DIR = "/tmp/sensoroni.python"

func cleanup_tmp() {
	os.RemoveAll(TMP_DIR)
}

func init_tmp(tester *testing.T) {
	cleanup_tmp()
	os.MkdirAll(TMP_DIR, 0777)

	entries, err := ioutil.ReadDir(TMP_DIR)
	assert.NoError(tester, err)
	assert.Equal(tester, 0, len(entries))
}

func TestInitAnalyze(tester *testing.T) {
	cfg := make(map[string]interface{})
	sq := NewAnalyze(nil)
	err := sq.Init(cfg)
	assert.NotNil(tester, err)
	assert.Equal(tester, DEFAULT_ANALYZERS_PATH, sq.analyzersPath)
	assert.Equal(tester, DEFAULT_SITE_PACKAGES_PATH, sq.sitePackagesPath)
	assert.Equal(tester, DEFAULT_SOURCE_PACKAGES_PATH, sq.sourcePackagesPath)
	assert.Equal(tester, DEFAULT_ANALYZER_EXECUTABLE, sq.analyzerExecutable)
	assert.Equal(tester, DEFAULT_ANALYZER_INSTALLER, sq.analyzerInstaller)
	assert.Equal(tester, DEFAULT_TIMEOUT_MS, sq.timeoutMs)
	assert.Equal(tester, DEFAULT_PARALLEL_LIMIT, sq.parallelLimit)
	assert.Equal(tester, DEFAULT_SUMMARY_LENGTH, sq.summaryLength)
}

func TestCreateAnalyzer(tester *testing.T) {
	init_tmp(tester)
	defer cleanup_tmp()

	cfg := make(map[string]interface{})
	cfg["analyzersPath"] = "test-resources"
	cfg["sourcePackagesPath"] = "test-source-packages"
	cfg["sitePackagesPath"] = TMP_DIR
	sq := NewAnalyze(nil)
	err := sq.Init(cfg)
	assert.Error(tester, err, "Unable to invoke JobMgr.AddJobProcessor due to nil agent")
	assert.Equal(tester, 1, len(sq.analyzers))

	entries, err := ioutil.ReadDir(TMP_DIR)
	assert.NoError(tester, err)
	assert.Equal(tester, 15, len(entries))
}

func TestInit(tester *testing.T) {
	cfg := make(map[string]interface{})
	sq := NewAnalyze(nil)
	err := sq.Init(cfg)
	assert.NotNil(tester, err)
}

func TestJobKindMissing(tester *testing.T) {
	cfg := make(map[string]interface{})
	sq := NewAnalyze(nil)
	sq.Init(cfg)

	// Job kind is not set to analyze, so nothing should execute
	job := model.NewJob()
	reader, err := sq.ProcessJob(job, nil)
	assert.Nil(tester, reader)
	assert.Nil(tester, err)
	assert.Empty(tester, job.Results)
}

func TestJobFilterMissing(tester *testing.T) {
	cfg := make(map[string]interface{})
	sq := NewAnalyze(nil)
	sq.Init(cfg)

	// Proper job kind, but no filter set yet
	job := model.NewJob()
	job.Kind = "analyze"
	reader, err := sq.ProcessJob(job, nil)
	assert.Nil(tester, reader)
	assert.Nil(tester, err)
	assert.Empty(tester, job.Results)
}

func TestAnalyzersMissing(tester *testing.T) {
	cfg := make(map[string]interface{})
	sq := NewAnalyze(nil)
	sq.Init(cfg)

	// Job kind and filter parameters specified but still no analyzers
	job := model.NewJob()
	job.Kind = "analyze"
	job.Filter.Parameters["foo"] = "bar"
	reader, err := sq.ProcessJob(job, nil)
	assert.Nil(tester, reader)
	assert.Error(tester, err, "No analyzers processed successfully")
	assert.Empty(tester, job.Results)
}

func TestAnalyzersExecuted(tester *testing.T) {
	init_tmp(tester)
	defer cleanup_tmp()

	cfg := make(map[string]interface{})
	cfg["analyzersPath"] = "test-resources"
	cfg["sourcePackagesPath"] = "test-source-packages"
	cfg["sitePackagesPath"] = TMP_DIR
	sq := NewAnalyze(nil)
	sq.Init(cfg)

	job := model.NewJob()
	job.Kind = "analyze"
	job.Filter.Parameters["foo"] = "bar"
	reader, err := sq.ProcessJob(job, nil)
	assert.Nil(tester, reader)
	assert.Nil(tester, err)
	assert.Len(tester, job.Results, 1)
	assert.Equal(tester, "whois", job.Results[0].Id)
	assert.Equal(tester, "something here that is so long it will need to be ...", job.Results[0].Summary)
}

func TestCreateResult(tester *testing.T) {
	cfg := make(map[string]interface{})
	analyzer := model.NewAnalyzer("test", true)
	sq := NewAnalyze(nil)
	sq.Init(cfg)
	result := sq.createJobResult(analyzer, "myinput", []byte(`{"foo":"bar"}`), nil)
	assert.Equal(tester, `{"foo":"bar"}`, result.Summary)

	result = sq.createJobResult(analyzer, "myinput", []byte(`{"foo":"bar", "status": "threat", "data":"this is a long piece of data"}`), nil)
	assert.Equal(tester, `{"foo":"bar", "status": "threat", "data":"this is ...`, result.Summary)

	result = sq.createJobResult(analyzer, "myinput", []byte(`{"foo":"bar", "status": "threat", "summary":"this is a long piece of data"}`), nil)
	assert.Equal(tester, `this is a long piece of data`, result.Summary)

	// Normal exit
	err := exec.ExitError{}
	result = sq.createJobResult(analyzer, "myinput", []byte(`{"foo":"bar"}`), &err)
	assert.Equal(tester, `internal_failure`, result.Summary)
}
