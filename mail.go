package main

import (
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aarzilli/sandblast"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
)

type Envelope struct {
	date        time.Time
	to          []string
	from        []string
	subject     string
	message     string
	attachments []io.Reader
}

func mailReceiver() []Envelope {
	// log.SetOutput(io.Discard)
	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS("imap.yandex.com:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login("sh0ma04@yandex.ru", "htglhbaigtvybtcn"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// Select a mailbox
	if _, err := c.Select("INBOX", false); err != nil {
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
			// log.Println("Date:", date)
			envlp.date = date
		}
		if from, err := header.AddressList("From"); err == nil {
			// log.Println("From:", from)
			for _, fr := range from {
				envlp.from = append(envlp.from, fr.Name+" | "+fr.Address)
			}
		}
		if to, err := header.AddressList("To"); err == nil {
			// log.Println("To:", to)
			for _, t := range to {
				envlp.to = append(envlp.to, t.Name+" | "+t.Address)
			}
		}
		if subject, err := header.Subject(); err == nil {
			// log.Println("Subject:", subject)
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
				// log.Printf("Got text: %v\n%v\n", p.Header.Get("Content-Type"), p.Header.Get("Content-Transfer-Encoding"))
				if strings.Contains(p.Header.Get("Content-Type"), "html") {
					node, err := html.Parse(strings.NewReader(string(b)))
					if err != nil {
						log.Fatal("Parsing error: ", err)
					}
					_, envlp.message, _ = sandblast.Extract(node, sandblast.KeepLinks)
				} else {
					envlp.message = string(b)
				}

			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Printf("Got attachment: %v\n", filename)
				// Create file with attachment name
				file, err := os.Create(filename)
				if err != nil {
					log.Fatal(err)
				}
				// using io.Copy instead of io.ReadAll to avoid insufficient memory issues
				_, err = io.Copy(file, p.Body)
				if err != nil {
					log.Fatal(err)
				}
				file.Close()
				envlp.attachments = append(envlp.attachments, p.Body)
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
