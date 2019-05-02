// Iterates over the existing and new netns (via inotify) given a NS_PREFIX
// and sets a LOG rule for some specific CHAINS on the nat table.
package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/niedbalski/go-iptables/iptables"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

const (
	NS_FILEPATH = "/var/run/netns"
	NS_PREFIX   = "qrouter"
)

var CHAINS_TO_LOG = []string{
	"neutron-l3-agent-POSTROUTING",
}

func NewNamespaces(errorChannel *chan error, nsChannel *chan string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()
	err = watcher.Add(NS_FILEPATH)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				*errorChannel <- fmt.Errorf("Cannot get watcher events")
				return
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				*nsChannel <- event.Name
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				*errorChannel <- fmt.Errorf("Cannot get watcher events")
				return
			}
			log.Println("error:", err)
		}
	}
}

func CurrentNamespaces(errorChannel *chan error, nsChannel *chan string) {
	nameSpaces, err := filepath.Glob(NS_FILEPATH + "/**")
	if err != nil {
		*errorChannel <- err
		return
	}

	for _, namespace := range nameSpaces {
		if strings.HasPrefix(filepath.Base(namespace), NS_PREFIX) {
			*nsChannel <- namespace
		}
	}
}

func AddIptablesLogToNamespace(nameSpace string) error {
	iptables, err := iptables.New(nameSpace)
	if err != nil {
		return err
	}

	for _, chain := range CHAINS_TO_LOG {
		log.Infof("Adding logging rules to namespace: %s - chain: %s", nameSpace, chain)
		exists, _ := iptables.Exists("nat", chain)
		if exists {
			err = iptables.Insert("nat", chain, 1, "-j", "LOG", "--log-prefix", chain)
			if err != nil {
				return fmt.Errorf("cannot insert log rule on chain: %s", err)
			}
		}
	}
	return nil
}

func main() {
	var errorChannel chan error
	var nsChannel chan string

	errorChannel = make(chan error)
	nsChannel = make(chan string)

	go NewNamespaces(&errorChannel, &nsChannel)
	go CurrentNamespaces(&errorChannel, &nsChannel)
	go func(nsChannel *chan string) {
		for {
			select {
			case nameSpace := <-*nsChannel:
				{
					err := AddIptablesLogToNamespace(filepath.Base(nameSpace))
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}(&nsChannel)

	for {
		select {
		case err := <-errorChannel:
			{
				log.Panicf("Error processing namespaces: %s", err)
			}
		}
	}
}
