# IMAPSearcher
IMAPSearcher is a small cli tool for searching in the address fields of emails in imap folders.  


I made this tool because i wanted to get an overview of the different domains in my imap folders. I use my own domain for emails and use different emails for all kinde of accounts (e.g. amazon@my-domain.net). The main purpose for this tool is to see which addresses are currently used.  

IMAPSearcher connects to a an imap server and iterates over all IMAP folders and every email found in these folders. In every email, the fields from, to, cc, bcc, reply-to and sender are searched for a given term. 

Currently only IMAP servers with TLS are supported.
 
## Usage
The cli app has three different search options. You can search by the whole email address, the part before the @ or the part after the @, aka hostname. 

The main output is a list of found email addresses which are printed to stdout, one by line. All other output is printed to stderr so you should be able to redirect the output of IMAPSearcher and process its output. 

### General options
For every search you need to provide information to which IMAP server and as which user you want to search. The parameters are login (--login), password (--password) and an address (--addr). The address consists of the hostname, a colon and the port. 

### Search for a full email address
Search for given email address:

```bash
imapsearch --login me --password secret --host imap.example.org:993 search --email me@example.org
```

### Search for a full hostname
Search for given hostname:

```bash
imapsearch --login me --password secret --host imap.example.org:993 search --hostname example.org
```

### Search for a mailbox
Search for given mailbox:

```bash
imapsearch --login me --password secret --host imap.example.org:993 search --mailbox me
```
