package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type Service struct {
	State bool
	Mail
	TeleBot
	EnvelopeCh chan Envelope
	addr       string
	cert       string
	key        string
}

func (s *Service) Init() {
	s.State = true
	s.TeleBot = TeleBot{}
	s.Mail = Mail{}
	s.TeleBot.Init()
	s.Mail.Init()
	s.EnvelopeCh = make(chan Envelope)
	s.addr = os.Getenv("SERV_ADDR")
	s.cert = os.Getenv("SERV_CERT")
	s.key = os.Getenv("SERV_KEY")
}

func (s *Service) Stop() {
	s.State = false
}

func (s *Service) Start() {
	s.State = true
}

func (s *Service) RestartReceiver() {
	s.State = false
	s.Mail.Init()
	s.State = true
}

func (s *Service) Run() {
	go func(e chan Envelope, m *Mail) {
		ticker := time.NewTicker(time.Minute)
		for ; true; <-ticker.C {
			if s.State {
				for _, envelope := range m.Receiver() {
					e <- envelope
				}
			}
		}
	}(s.EnvelopeCh, &s.Mail)
	go s.RunServer()
	s.RunBot(s.EnvelopeCh)

}

func (s *Service) RunServer2() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.URL.Path[1:])
		if err != nil {
			log.Panic(err)
		}
		html, err := s.MsgStore.get(id)
		if err != nil {
			log.Panic(err)
		}
		fmt.Fprint(w, html)
	})

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         s.addr,
	}
	log.Println(s.addr, s.cert, s.key)
	err := srv.ListenAndServeTLS(s.cert, s.key)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (s *Service) RunServer() {
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("88.210.9.244.sslip.io"), //Your domain here
		Cache:      autocert.DirCache("certs"),                      //Folder for storing certificates
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.URL.Path[1:])
		if err != nil {
			log.Println(err)
			return
		}
		html, err := s.MsgStore.get(id)
		if err != nil {
			log.Panic(err)
		}
		fmt.Fprint(w, html)
	})

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello Murad")
	})

	server := &http.Server{
		Addr: ":https",
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
			MinVersion:     tls.VersionTLS12, // improves cert reputation score at https://www.ssllabs.com/ssltest/
		},
	}

	go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

	log.Fatal(server.ListenAndServeTLS("", "")) //Key and cert are coming from Let's Encrypt

}
