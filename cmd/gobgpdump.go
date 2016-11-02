package main

import (
	"bufio"
	"fmt"
	mrt "github.com/CSUNetSec/protoparse/protocol/mrt"
	"io"
	"log"
	"os"
	"time"
)

//accepts an error (can be nil which turns the func into a noop
//and variadic arguemnts that are fds to be closed before calling
//os.exit in case the error is not nil
func errx(e error, fds ...io.Closer) {
	if e == nil {
		return
	}
	log.Printf("error: %s\n", e)
	for _, fd := range fds {
		fd.Close()
	}
	os.Exit(-1)
}

func main() {
	if len(os.Args) != 2 {
		log.Println("mrt file not provided")
		os.Exit(-1)
	}
	mrtfd, err := os.Open(os.Args[1])
	errx(err)
	defer mrtfd.Close()
	mrtScanner := bufio.NewScanner(mrtfd)
	scanbuffer := make([]byte, 2<<20) //an internal buffer for the large tokens (1M)
	mrtScanner.Buffer(scanbuffer, cap(scanbuffer))
	mrtScanner.Split(mrt.SplitMrt)
	numentries := 0
	totsz := 0
	t1 := time.Now()
	for mrtScanner.Scan() {
		ret := ""
		numentries++
		data := mrtScanner.Bytes()
		totsz += len(data)
		mrth := mrt.NewMrtHdrBuf(data)
		bgp4h, err := mrth.Parse()
		if err != nil {
			log.Printf("Failed parsing MRT header %d :%s", numentries, err)
		}
		ret += fmt.Sprintf("[%d] MRT Header: %s\n", numentries, mrth)
		bgph, err := bgp4h.Parse()
		if err != nil {
			log.Printf("Failed parsing BGP4MP header %d :%s", numentries, err)
		}
		ret += fmt.Sprintf("BGP4MP Header:%s\n", bgp4h)
		bgpup, err := bgph.Parse()
		if err != nil {
			log.Printf("Failed parsing BGP Header  %d :%s", numentries, err)
		}
		ret += fmt.Sprintf("BGP Header: %s\n", bgph)
		_, err = bgpup.Parse()
		if err != nil {
			log.Printf("Failed parsing BGP Update  %d :%s", numentries, err)
		}
		ret += fmt.Sprintf("BGP Update:%s\n", bgpup)
		fmt.Printf("%s", ret)
	}

	if err := mrtScanner.Err(); err != nil {
		errx(err, mrtfd)
	}
	dt := time.Since(t1)
	log.Printf("Scanned: %d entries, total size: %d bytes in %v\n", numentries, totsz, dt)
}