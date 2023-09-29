package main

import (
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type Envelope struct {
	date      time.Time
	to        []string
	from      []string
	subject   string
	message   string
	filenames []string
	htmlType  bool
}

type Mail struct {
	login    string
	passwd   string
	server   string
	Folder   string
	localDir string
	debug    bool
}

func (rec *Mail) Init() {
	rec.login = os.Getenv("EMAIL_ADDRS")
	rec.passwd = os.Getenv("EMAIL_PASSW")
	rec.server = os.Getenv("EMAIL_SERVR")
	rec.Folder = os.Getenv("EMAIL_FOLDR")
	rec.localDir = os.Getenv("FILEDIR")
	if os.Getenv("EMAIL_DEBUG") == "true" {
		rec.debug = true
	} else {
		rec.debug = false
	}
}

func (rec *Mail) Receiver() []Envelope {
	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS(rec.server, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(rec.login, rec.passwd); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// Select a mailbox
	if _, err := c.Select(rec.Folder, false); err != nil {
		log.Fatal(err)
	}

	// Set search criteria
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("IDs found:", ids)
	if len(ids) == 0 {
		log.Println("No Ids")
		return []Envelope{}
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(ids...)

	var section imap.BodySectionName
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	var envlps []Envelope
	for msg := range messages {
		var envlp Envelope

		if msg == nil {
			log.Fatal("Server didn't returned message")
		}

		r := msg.GetBody(&section)
		if r == nil {
			log.Fatal("Server didn't returned message body")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
		}

		// Print some info about the message
		header := mr.Header
		if date, err := header.Date(); err == nil {
			envlp.date = date
		}
		if from, err := header.AddressList("From"); err == nil {
			for _, fr := range from {
				envlp.from = append(envlp.from, fr.Name+" | "+fr.Address)
			}
		}
		if to, err := header.AddressList("To"); err == nil {
			for _, t := range to {
				envlp.to = append(envlp.to, t.Name+" | "+t.Address)
			}
		}
		if subject, err := header.Subject(); err == nil {
			envlp.subject = subject
		}

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				b, _ := io.ReadAll(p.Body)
				if strings.Contains(p.Header.Get("Content-Type"), "html") {
					envlp.htmlType = true
				} else {
					envlp.htmlType = false
				}
				envlp.message = string(b)

			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Printf("Got attachment: %v\n", filename)
				// Create file with attachment name
				tempDir := strconv.FormatInt(time.Now().UnixMilli(), 10)
				os.MkdirAll(rec.localDir+tempDir, os.ModePerm)
				if err != nil {
					log.Panic(err)
				}
				path := rec.localDir + tempDir + "/" + filename
				file, err := os.Create(path)
				if err != nil {
					log.Fatal(err)
				}
				// using io.Copy instead of io.ReadAll to avoid insufficient memory issues
				_, err = io.Copy(file, p.Body)
				if err != nil {
					log.Fatal(err)
				}
				file.Close()
				envlp.filenames = append(envlp.filenames, path)
			}
		}
		envlps = append(envlps, envlp)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")

	return envlps
}
