package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var server string
var lock *api.Lock
var stopCh chan struct{}
var id string
var wait bool

func emit(lock *api.Lock, lc <-chan struct{}, sc chan bool) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	log.Infoln("Established receiver connection")
	defer conn.Close()
	defer lock.Unlock()

	counter := 1
	for {
		select {
		case _, ok := <-lc:
			if !ok {
				log.Warnln("Lock lost!")
				sc <- true
				return
			}
		default:
			// emit counter
			fmt.Fprintf(conn, "%s %d\n", id, counter)
			time.Sleep(1 * time.Second)
			counter++
		}
	}
}

func main() {

	// cli flags
	flag.BoolVar(&wait, "wait", true, "use consul session locking")
	flag.StringVar(&id, "id", "unknown", "id of emitter")
	flag.StringVar(&server, "server", "127.0.0.1:9000", "tcp server")
	flag.Parse()

	// logging
	level, err := log.ParseLevel("DEBUG")
	if err != nil {
		panic(err)
	}
	log.SetLevel(level)

	// catch sigint to clean up
	c := make(chan os.Signal, 2)
	signal.Notify(c,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-c
		err := lock.Unlock()
		if err != nil {
			log.Warnln("Could clean up!")
			log.Warnln(err)
		}
		log.Print("Exiting")
		os.Exit(1)
	}()

	// create consul client
	log.Infoln("Setting up consul connection")
	client, err := api.NewClient(&api.Config{Address: "127.0.0.1:8500"})
	if err != nil {
		panic(err)
	}

	// set session lock options
	opts := &api.LockOptions{
		Key:        "tcp_receiver/lock",
		Value:      []byte("set by tcp emitter " + id),
		SessionTTL: "10s",
		SessionOpts: &api.SessionEntry{
			Behavior: "release",
		},
	}

	// create session lock
	log.Infoln("Creating session lock")
	lock, err = client.LockOpts(opts)
	if err != nil {
		panic(err)
	}

	// consul lock acquision stop channel
	stopCh = make(chan struct{})
	// consul lock status channel
	lockCh := make(<-chan struct{})
	// emit running status channel
	emitCh := make(chan bool)

	// session lock loop
	for {
		if wait {
			// cancel blocking lock acquision
			go func() {
				time.Sleep(1 * time.Second)
				stopCh <- struct{}{}
			}()

			log.Infoln("Attemping lock acquision")
			// this is a blocking operation
			lockCh, err = lock.Lock(stopCh)
			if err != nil {
				log.Errorln(err)
			}
			log.Infoln("Lock aquired")
		}

		go emit(lock, lockCh, emitCh)

		// wait until emitter stops
		<-emitCh
	}
}
