package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/miekg/dns"
)

func main() {
	httpPortText := os.Getenv("HTTP_PORT")
	if httpPortText == "" {
		httpPortText = "8088"
	}
	httpPort, err := strconv.Atoi(httpPortText)
	if err != nil {
		panic(fmt.Errorf("failed to parse HTTP port: %w", err))
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: http.HandlerFunc(httpHandler),
	}

	errs := make(chan error)

	go func() {
		fmt.Printf("HTTP server listening on port %d\n", httpPort)
		errs <- httpServer.ListenAndServe()
	}()

	dnsPortText := os.Getenv("DNS_PORT")
	if dnsPortText == "" {
		dnsPortText = "8053"
	}
	dnsPort, err := strconv.Atoi(dnsPortText)
	if err != nil {
		panic(fmt.Errorf("failed to parse dns port: %w", err))
	}

	dnsServer := dns.Server{
		Addr:    fmt.Sprintf(":%d", dnsPort),
		Handler: dns.HandlerFunc(handleDNSRequest),
		Net:     "udp",
	}

	go func() {
		fmt.Printf("DNS server listening on port %d\n", dnsPort)
		errs <- dnsServer.ListenAndServe()
	}()

	err = <-errs
	if err != nil {
		panic(err)
	}
}

var records = map[string]string{
	"test.service.": "192.168.0.2",
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	fmt.Printf("Handling DNS request for %s\n", r.Question[0].Name)

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range m.Question {
			switch q.Qtype {
			case dns.TypeA:
				fmt.Printf("Query for %s\n", q.Name)
				ip := records[q.Name]
				if ip != "" {
					rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
					if err == nil {
						m.Answer = append(m.Answer, rr)
					}
				}
			}
		}
	}

	if err := w.WriteMsg(m); err != nil {
		panic(err)
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")

	body := fmt.Sprintf(`
request-uri: %s
method: %s
host: %s
remote-addr: %s
`, r.RequestURI, r.Method, r.Host, r.RemoteAddr)

	if _, err := w.Write([]byte(body)); err != nil {
		panic(err)
	}
}
