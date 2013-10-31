This is a data collector for DNS queries intercepted by modems/routers.
It has only been tested with the cable modem Thomson DWG850-4B.
I don't know if it will work with other devices but there's a big chance if the 
device has the ability of sending events to remote syslog servers. Some tweak is
needed depending on the message format from where the useful data is extracted.

Configuration
-------------

Your machine should be running some kind of syslog daemon (syslog-ng, rsyslog, etc).
(I tested only with rsyslog.)
You'll need to create a rule to forward syslog messages from the router to &lt;machine running dns-stats&gt;:1514.

rsyslog example:

```
:hostname,isequal,"2013" @127.0.0.1:1514 & ~
```

"2013" should be something like "192.168.0.1" but my router is buggy and sends the current year instead of its address. Maybe it's not your case.

- Download the latest version of Go &lt;http://code.google.com/p/go/downloads/list&gt;
- *if it's your first time using Go, there are some details of installation that should be clarified with some research*
- At a terminal, clone this repo and execute *go build && ./dns-stats*
- Go to https://&lt;router IP&gt;/RgFirewallRL.asp
- Enable Permitted Connections and Apply
- Leave it for some hours
- Execute ./dns-stats -report or go to http://&lt;server ip&gt;:1514

If nothing happens and you only see "tick messages" maybe the syslog message sent
from your router has another format than I'm using for DWG850-4B and you should
contact me, providing a sample of the message and the model of your router.
