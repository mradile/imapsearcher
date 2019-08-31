package main

import (
	"fmt"
	"os"

	imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var version string = "dev-snapshot"
var debug bool

var (
	login    string
	password string
	addr     string
)

var (
	searchHostname    string
	searchMailboxName string
	searchEmail       string
)

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = "imap searcher"
	cliApp.Usage = ""
	cliApp.Version = version
	cliApp.Authors = []cli.Author{{Name: "Martin Radile", Email: "martin.radile@gmail.com"}}
	cliApp.Copyright = "Martin Radile 2019"

	cliApp.Flags = []cli.Flag{

		cli.StringFlag{
			Name:        "login",
			Usage:       "login for the email account",
			EnvVar:      "LOGIN",
			Destination: &login,
		},
		cli.StringFlag{
			Name:        "password",
			Usage:       "password for the email account",
			EnvVar:      "PASSWORD",
			Destination: &password,
		},
		cli.StringFlag{
			Name:        "addr",
			Usage:       "imap host & port (imap.example.org:993)",
			EnvVar:      "ADDR",
			Destination: &addr,
		},

		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "debug",
			EnvVar:      "DEBUG",
			Destination: &debug,
		},
	}

	cliApp.Commands = []cli.Command{
		{
			Name:  "search",
			Usage: "search in emails",
			Action: func(context *cli.Context) error {
				if debug {
					log.SetLevel(log.DebugLevel)
				} else {
					log.SetLevel(log.InfoLevel)
				}

				c, err := Login()
				if err != nil {
					return cli.NewExitError("could not connect: "+err.Error(), 1)
				}
				defer c.Logout()

				folders, err := GetIMAPFolder(c)
				if err != nil {
					return cli.NewExitError("could not fetch mailbox names: "+err.Error(), 1)
				}

				emailMap := make(map[string]bool)
				emailReceiver := make(chan string, 10)
				go func() {
					for email := range emailReceiver {
						if _, ok := emailMap[email]; !ok {
							emailMap[email] = true
							log.WithFields(log.Fields{
								"email": email,
							}).Info("added email")
						}
					}
				}()

				for _, folder := range folders {
					err := GetEmails(folder, c, emailReceiver)
					if err != nil {
						log.WithFields(log.Fields{
							"mailbox": folder,
						}).Error("error while fetching emails from mailbox")
					}
				}
				close(emailReceiver)

				log.WithFields(log.Fields{
					"count": len(emailMap),
				}).Info("found emails")
				for email, _ := range emailMap {
					fmt.Println(email)
				}
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "email, e",
					Usage:       "the full email address (me@example.org)",
					EnvVar:      "S_EMAIL",
					Destination: &searchEmail,
				},
				cli.StringFlag{
					Name:        "hostname, ho",
					Usage:       "hostname of email address (the 'example.org' of me@example.org)",
					EnvVar:      "S_HOSTNAME",
					Destination: &searchHostname,
				},
				cli.StringFlag{
					Name:        "mailbox, mb",
					Usage:       "mailbox name of email address (the 'me' of me@example.org)",
					EnvVar:      "S_MAILBOX",
					Destination: &searchMailboxName,
				},
			},
		},
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal("could not initialize app")
	}
}

func Login() (*client.Client, error) {
	log.WithFields(log.Fields{
		"addr": addr,
	}).Debug("trying to connect to imap server")

	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not dial host")
	}

	log.Debug("connected")

	if err := c.Login(login, password); err != nil {
		return nil, errors.Wrap(err, "could not log in")
	}

	log.Debug("logged in")

	return c, err
}

func GetIMAPFolder(c *client.Client) ([]string, error) {
	log.Debug("fetching mailboxes")

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	mailBoxNames := make([]string, 0, 50)
	for m := range mailboxes {
		mailBoxNames = append(mailBoxNames, m.Name)
	}

	if err := <-done; err != nil {
		return nil, errors.Wrap(err, "failed fetching mailboxes")
	}

	return mailBoxNames, nil
}

func GetEmails(imapFolder string, c *client.Client, emailReceiver chan string) error {
	mbox, err := c.Select(imapFolder, false)
	if err != nil {
		return errors.Wrap(err, "could not select mailbox")
	}

	log.WithFields(log.Fields{
		"mailbox":    imapFolder,
		"mail_count": mbox.Messages,
	}).Debug("fetching emails from mailbox")

	if mbox.Messages == 0 {
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(0, mbox.Messages-1)

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	for msg := range messages {
		addEmailsToList(msg.Envelope.From, emailReceiver, imapFolder, "from")
		addEmailsToList(msg.Envelope.To, emailReceiver, imapFolder, "to")
		addEmailsToList(msg.Envelope.Bcc, emailReceiver, imapFolder, "bcc")
		addEmailsToList(msg.Envelope.Cc, emailReceiver, imapFolder, "cc")
		addEmailsToList(msg.Envelope.ReplyTo, emailReceiver, imapFolder, "reply-to")
		addEmailsToList(msg.Envelope.Sender, emailReceiver, imapFolder, "sender")
	}

	if err := <-done; err != nil {
		return errors.Wrap(err, "failed fetching emails")
	}

	return nil
}

func addEmailsToList(source []*imap.Address, emailReceiver chan string, folder, field string) {
	for _, addr := range source {
		email := fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)

		addEmail := false
		if searchEmail != "" && searchEmail == email {
			addEmail = true
		} else if searchHostname != "" && searchHostname == addr.HostName {
			addEmail = true
		} else if searchMailboxName != "" && searchMailboxName == addr.MailboxName {
			addEmail = true
		}

		if addEmail {
			log.WithFields(log.Fields{
				"email":    email,
				"field":    field,
				"folder":   folder,
				"mailbox":  addr.MailboxName,
				"hostname": addr.HostName,
			}).Debug("found email matching pattern")

			emailReceiver <- email
		}
	}
}
