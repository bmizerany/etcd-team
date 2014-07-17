/*
Copyright 2013 CoreOS Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/etcd/config"
)

func TestRunStop(t *testing.T) {
	path, _ := ioutil.TempDir("", "etcd-")
	defer os.RemoveAll(path)

	config := config.New()
	config.Name = "ETCDTEST"
	config.DataDir = path
	config.Addr = "localhost:0"
	config.Peer.Addr = "localhost:0"

	etcd := New(config)
	go etcd.Run()
	<-etcd.ReadyNotify()
	etcd.Stop()
}

// A benchmark for profiling the server without the HTTP client code.
// The client code runs in a subprocess.
//
// Borrowed and modified from Go net/http
// For use like:
//   $ go test -c
//   $ ./http.test -test.run=XX -test.bench=BenchmarkServer -test.benchtime=15s -test.cpuprofile=http.prof
//   $ go tool pprof http.test http.prof
//   (pprof) web
func BenchmarkServer(b *testing.B) {
	b.ReportAllocs()
	// Child process mode;
	if uurl := os.Getenv("TEST_BENCH_SERVER_URL"); uurl != "" {
		n, err := strconv.Atoi(os.Getenv("TEST_BENCH_CLIENT_N"))
		if err != nil {
			panic(err)
		}
		for i := 0; i < n; i++ {
			req, err := http.NewRequest("PUT", uurl, strings.NewReader("value=bar&ttl=10"))
			if err != nil {
				log.Panicf("Get: %v", err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Panicf("Put: %v", err)
			}
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
			if res.StatusCode/100 != 2 {
				log.Panicf("StateCode: %d", res.StatusCode)
			}
			res, err = http.Get(uurl)
			if err != nil {
				log.Panicf("Get: %v", err)
			}
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}
		os.Exit(0)
		return
	}

	path, _ := ioutil.TempDir("", "etcd-")
	defer os.RemoveAll(path)

	config := config.New()
	config.Name = "ETCDSERVEBENCH"
	config.DataDir = path
	config.Addr = "localhost:1110"
	config.Peer.Addr = "localhost:1120"
	config.Snapshot = true
	config.SnapshotCount = 100

	etcd := New(config)
	go etcd.Run()
	<-etcd.ReadyNotify()
	defer etcd.Stop()

	b.StartTimer()

	cmd := exec.Command(os.Args[0], "-test.run=XXXX", "-test.bench=BenchmarkServer")
	cmd.Env = append([]string{
		fmt.Sprintf("TEST_BENCH_CLIENT_N=%d", b.N),
		fmt.Sprintf("TEST_BENCH_SERVER_URL=%s", config.Addr+"/v2/keys/foo"),
	}, os.Environ()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		b.Errorf("Test failure: %v, with output: %s", err, out)
	}
}
