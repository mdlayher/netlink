// Command nlstress continuously stresses the netlink package.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"

	"github.com/mdlayher/genetlink"
)

func main() {
	var (
		jFlag = flag.Int("j", runtime.NumCPU(), "number of jobs")
	)
	flag.Parse()

	go func() {
		// For pprof support.
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	var wg sync.WaitGroup
	wg.Add(*jFlag)

	for i := 0; i < *jFlag; i++ {
		go func() {
			defer wg.Done()

			for {
				work()
			}
		}()
	}

	wg.Wait()
}

func work() {
	c, err := genetlink.Dial(nil)
	if err != nil {
		panicf("failed to dial: %v", err)
	}
	defer c.Close()

	for i := 0; i < 10; i++ {
		if _, err := c.GetFamily("nlctrl"); err != nil {
			panicf("failed to get nlctrl: %v", err)
		}
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
