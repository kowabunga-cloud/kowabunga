/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kiwi

import (
	"fmt"
	"sync"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/miekg/dns"
)

// DnsServer represents an instance of a server, which serves DNS requests at a particular address (host and port).
// A server is capable of serving numerous zones on the same address and the listener may be stopped for
// graceful termination (POSIX only).
type DnsServer struct {
	Port      int
	Recursors []string

	srv     *dns.Server
	records map[string]string // formatted as "example.com.": "a.b.c.d"
	m       sync.Mutex        // protects the servers
}

func NewDnsServer(port int, recursors []string) (*DnsServer, error) {
	return &DnsServer{
		Port:      port,
		Recursors: recursors,
		records:   map[string]string{},
	}, nil
}

func (s *DnsServer) Update(records map[string]string) {
	s.m.Lock()
	s.records = records
	s.m.Unlock()
}

func (s *DnsServer) Start(port int) error {
	addr := fmt.Sprintf(":%d", s.Port)

	s.srv = &dns.Server{
		Addr: addr, // Listen on all local interfaces
		Net:  "udp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			s.ServeDNS(w, r)
		}),
	}

	// Run the server in its own goroutine so we can log start‑up failures cleanly.
	go func() {
		err := s.srv.ListenAndServe()
		if err != nil {
			klog.Errorf("Failed to start DNS server: %v", err)
		}
	}()

	klog.Infof("Starting DNS server on udp:%d", s.Port)
	return nil
}

func (s *DnsServer) Stop() error {
	return s.srv.Shutdown()
}

// ServeDNS is called for every incoming DNS request.
func (s *DnsServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	// Build a fresh reply that mirrors the request ID and flags.
	resp := new(dns.Msg)
	resp.SetReply(r)
	resp.Authoritative = true

	// Guard against empty queries.
	if len(r.Question) == 0 {
		_ = w.WriteMsg(resp)
		return
	}
	q := r.Question[0]

	// Try to answer from the local zone (only A records for now)
	if q.Qtype == dns.TypeA {
		s.m.Lock()
		ip, ok := s.records[q.Name]
		s.m.Unlock()
		if ok {
			rr, err := dns.NewRR(q.Name + " IN A " + ip)
			if err != nil {
				klog.Debugf("[dns] Failed to build RR: %v", err)
			} else {
				resp.Answer = []dns.RR{rr}
				_ = w.WriteMsg(resp)
				return
			}
		}
		// If the name isn’t in local records we fall through to forwarding.
	}

	// Forward the whole request to recursors
	c := new(dns.Client)
	c.Net = "udp"
	c.Timeout = 5 * time.Second // reasonable timeout for a network hop

	var errRec error
	var fwdResp *dns.Msg
	for _, rec := range s.Recursors {
		// Forward the exact request we received.
		recursor := fmt.Sprintf("%s:53", rec)
		fwdResp, _, errRec = c.Exchange(r, recursor)
		if errRec == nil {
			break // first recursor replied
		}
	}

	// If all forwarding fail, reply with SERVFAIL so the client knows something went wrong.
	if errRec != nil {
		klog.Errorf("[dns] Forwarding error: %v", errRec)
		resp.Rcode = dns.RcodeServerFailure
		_ = w.WriteMsg(resp)
		return
	}

	// Preserve the original transaction ID (already done by Exchange) and send back recursor’s answer.
	_ = w.WriteMsg(fwdResp)
}
