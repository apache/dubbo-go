package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/dubbo/go-for-apache-dubbo/config/support"
	"github.com/montanaflynn/stats"
	"log"
	"sync"
	"sync/atomic"

	"time"
)
import (
	_ "github.com/dubbo/go-for-apache-dubbo/protocol/jsonrpc"
	_ "github.com/dubbo/go-for-apache-dubbo/registry/protocol"

	_ "github.com/dubbo/go-for-apache-dubbo/filter/imp"

	_ "github.com/dubbo/go-for-apache-dubbo/cluster/loadbalance"
	_ "github.com/dubbo/go-for-apache-dubbo/cluster/support"
	_ "github.com/dubbo/go-for-apache-dubbo/registry/zookeeper"
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
var concurrency = flag.Int("c", 1, "concurrency")
var total = flag.Int("n", 1, "total requests for all clients")

func main() {
	flag.Parse()

	conc, tn, err := checkArgs(*concurrency, *total)
	if err != nil {
		log.Printf("err: %v", err)
		return
	}
	n := conc
	m := tn / n

	log.Printf("concurrency: %d\nrequests per client: %d\n\n", n, m)

	var wg sync.WaitGroup
	wg.Add(n * m)

	log.Printf("sent total %d messages, %d message per client", n*m, m)

	conMap, _ := support.Load()
	if conMap == nil {
		panic("conMap is nil")
	}

	time.Sleep(3e9)

	var startWg sync.WaitGroup
	startWg.Add(n)

	var trans uint64
	var transOK uint64

	d := make([][]int64, n, n)

	//it contains warmup time but we can ignore it
	totalT := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		dt := make([]int64, 0, m)
		d = append(d, dt)

		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					log.Print("Recovered in f", r)
				}
			}()

			//warmup
			for j := 0; j < 5; j++ {
				user := &JsonRPCUser{}
				err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser(context.TODO(), []interface{}{"A003"}, user)
				if err != nil {
					fmt.Println(err)
				}
			}

			startWg.Done()
			startWg.Wait()

			for j := 0; j < m; j++ {
				t := time.Now().UnixNano()
				user := &JsonRPCUser{}
				err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser(context.TODO(), []interface{}{"A003"}, user)

				t = time.Now().UnixNano() - t

				d[i] = append(d[i], t)

				if err == nil && user.ID != "" {
					atomic.AddUint64(&transOK, 1)
				}

				if err != nil {
					log.Print(err.Error())
				}

				atomic.AddUint64(&trans, 1)
				wg.Done()
			}
		}(i)

	}

	wg.Wait()

	totalT = time.Now().UnixNano() - totalT
	log.Printf("took %f ms for %d requests\n", float64(totalT)/1000000, n*m)

	totalD := make([]int64, 0, n*m)
	for _, k := range d {
		totalD = append(totalD, k...)
	}
	totalD2 := make([]float64, 0, n*m)
	for _, k := range totalD {
		totalD2 = append(totalD2, float64(k))
	}

	mean, _ := stats.Mean(totalD2)
	median, _ := stats.Median(totalD2)
	max, _ := stats.Max(totalD2)
	min, _ := stats.Min(totalD2)
	p99, _ := stats.Percentile(totalD2, 99.9)

	log.Printf("sent     requests    : %d\n", n*m)
	log.Printf("received requests    : %d\n", atomic.LoadUint64(&trans))
	log.Printf("received requests_OK : %d\n", atomic.LoadUint64(&transOK))
	log.Printf("throughput  (TPS)    : %d\n", int64(n*m)*1000000000/totalT)
	log.Printf("mean: %.f ns, median: %.f ns, max: %.f ns, min: %.f ns, p99.9: %.f ns\n", mean, median, max, min, p99)
	log.Printf("mean: %d ms, median: %d ms, max: %d ms, min: %d ms, p99: %d ms\n", int64(mean/1000000), int64(median/1000000), int64(max/1000000), int64(min/1000000), int64(p99/1000000))
}

// checkArgs check concurrency and total request count.
func checkArgs(c, n int) (int, int, error) {
	if c < 1 {
		log.Printf("c < 1 and reset c = 1")
		c = 1
	}
	if n < 1 {
		log.Printf("n < 1 and reset n = 1")
		n = 1
	}
	if c > n {
		return c, n, errors.New("c must be set <= n")
	}
	return c, n, nil
}
