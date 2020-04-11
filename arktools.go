package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/james4k/rcon"
	"github.com/oov/ns"
)

func main() {
	flag.Parse()
	var err error
	switch flag.Arg(0) {
	case "islisten":
		err = isListen()
	case "send":
		err = sendRCON()
	case "watchdog":
		err = startWatchDog()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func hasState(proto string, state, port int) (bool, error) {
	f, err := os.Open("/proc/net/" + proto)
	if err != nil {
		return false, fmt.Errorf("cannot open /proc/net/%s: %w", proto, err)
	}
	defer f.Close()
	entries, err := ns.Parse(f)
	if err != nil {
		return false, fmt.Errorf("failed to parse /proc/net/%s: %w", proto, err)
	}
	for _, e := range entries {
		if e.State == state && e.LocalPort == port {
			return true, nil
		}
	}
	return false, nil
}

func isListen() error {
	proto := flag.Arg(1)
	var state int
	switch proto {
	case "tcp":
		state = 0x0a
	case "udp":
		state = 0x07
	default:
		return fmt.Errorf("unknown protocol: %s", proto)
	}
	port, err := strconv.Atoi(flag.Arg(2))
	if err != nil {
		return fmt.Errorf("unexpected port number: %w", err)
	}
	listen, err := hasState(proto, state, port)
	if err != nil {
		return fmt.Errorf("cannot get state: %w", err)
	}
	if !listen {
		return fmt.Errorf("is not listen")
	}
	return nil
}

func requestRCON(d *rcon.RemoteConsole, command string) (string, error) {
	sid, err := d.Write(command)
	if err != nil {
		return "", fmt.Errorf("could not write to RCON server: %w", err)
	}
	res, rid, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("could not read from RCON server: %w", err)
	}
	if sid != rid {
		return "", fmt.Errorf("ICON ID mismatched")
	}
	return res, nil
}

func sendRCON() error {
	host := flag.Arg(1)
	port, err := strconv.Atoi(flag.Arg(2))
	if err != nil {
		return fmt.Errorf("unexpected port number: %w", err)
	}
	password := flag.Arg(3)
	d, err := rcon.Dial(fmt.Sprintf("%s:%d", host, port), password)
	if err != nil {
		return fmt.Errorf("could not connect to RCON server: %w", err)
	}
	defer d.Close()

	command := flag.Arg(4)
	res, err := requestRCON(d, command)
	if err != nil {
		return err
	}
	fmt.Print(res)
	return nil
}

func getActivePlayers(addr string, password string) ([]string, error) {
	d, err := rcon.Dial(addr, password)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RCON server: %w", err)
	}
	res, err := requestRCON(d, "listplayers")
	if err != nil {
		return nil, err
	}
	players := strings.Split(strings.TrimSpace(res), "\n")
	if len(players) == 1 && players[0] == "No Players Connected" {
		return []string{}, nil
	}
	return players, nil
}

func writeResult(w http.ResponseWriter, typ string, r string) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(struct {
		Type   string `json:"type"`
		Result string `json:"result"`
	}{
		Type:   typ,
		Result: r,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func writeError(w http.ResponseWriter, e error) {
	writeResult(w, "error", e.Error())
}

func startWatchDog() error {
	listenAddr := flag.Arg(1)
	rconAddr := flag.Arg(2)
	rconPassword := flag.Arg(3)
	logFilePath := flag.Arg(4)

	var activePlayers []string
	var lastError error
	var m sync.RWMutex
	go func() {
		for {
			players, err := getActivePlayers(rconAddr, rconPassword)
			m.Lock()
			activePlayers = players
			lastError = err
			m.Unlock()
			if err != nil {
				time.Sleep(10 * time.Second)
			} else {
				time.Sleep(5 * time.Minute)
			}
		}
	}()
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		if logFilePath == "" {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(logFilePath)
		if err != nil {
			http.Error(w, fmt.Sprintf("ERROR: %v", err), 500)
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			http.Error(w, fmt.Sprintf("ERROR: %v", err), 500)
			return
		}
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Content-Type", "text/plain")
		http.ServeContent(w, r, "", fi.ModTime(), f)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.RLock()
		if lastError != nil {
			writeError(w, lastError)
		} else {
			writeResult(w, "running", fmt.Sprintf("online: %d player(s)", len(activePlayers)))
		}
		m.RUnlock()
	})
	server := &http.Server{
		Addr:              listenAddr,
		ReadHeaderTimeout: 20 * time.Second,
	}
	return server.ListenAndServe()
}
