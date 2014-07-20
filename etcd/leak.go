// +build ignore

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/coreos/etcd/config"
	"github.com/coreos/etcd/etcd"
)

func main() {
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

	defer func() {
		f, err := os.Create("heapdump")
		if err != nil {
			log.Fatal(err)
		}
		debug.WriteHeapDump(f.Fd())
	}()

	path, _ := ioutil.TempDir("", "etcd-")
	defer os.RemoveAll(path)

	config := config.New()
	config.Name = "ETCDSERVEBENCH"
	config.DataDir = path
	config.Addr = "localhost:1110"
	config.Peer.Addr = "localhost:1120"
	config.Snapshot = true
	config.SnapshotCount = 100
	config.VeryVeryVerbose = true

	ed := etcd.New(config)
	go ed.Run()
	<-ed.ReadyNotify()
	defer ed.Stop()

	fmt.Println("running", os.Args[0])
	cmd := exec.Command(os.Args[0], "-test.run=XXXX", "-test.bench=BenchmarkServer")
	cmd.Env = append([]string{
		fmt.Sprintf("TEST_BENCH_CLIENT_N=%d", int(1e8)),
		fmt.Sprintf("TEST_BENCH_SERVER_URL=%s", config.Addr+"/v2/keys/foo"),
	}, os.Environ()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Test failure: %v, with output: %s", err, out)
	}
}
