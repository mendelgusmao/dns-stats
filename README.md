This is a data collector for DNS queries intercepted by modems/routers.
It has only been tested with the cable modem Thomson DWG850-4B.
I don't know if it will work with other devices but there's a big chance if the 
device has the ability of sending events to remote syslog servers. Some tweak is
needed depending on the message format from where the useful data is extracted.

Configuration
-------------

Your machine should be running some kind of syslog daemon (syslog-ng, rsyslog, etc).
You'll need to create a rule to forward syslog messages from the router to &lt;machine running dns-stats&gt;:1514.

rsyslog example:

```
:hostname,isequal,"2013" @127.0.0.1:1514 
& ~
```

*"2013" should be something like "192.168.0.1" but my router is buggy and sends the current year instead of its address. Maybe it's not your case.*

- Download the latest version of Go &lt;http://code.google.com/p/go/downloads/list&gt;
- *if it's your first time using Go, there are some environment details that should be clarified with some research*
- At a terminal, clone this repo and execute *go build && ./dns-stats -source &lt;router ip&gt;:&lt;router name&gt;* 
- Configure your router to send syslog messages to the address of your machine. Be sure that port 514 is open to your internal network.
- Leave it running for a while
- Execute ./dns-stats -report or go to http://&lt;server ip&gt;:8514/dns

If nothing happens and you only see "tick messages", double check the configuration.
If OK, maybe the syslog messages sent from your router are in another format than the used by DWG850-4B and you should fork this repo and add your router definition to collector/routers, following the Router interface:

```
package routers

type ROUTER_NAME struct{}

func (_ ROUTER_NAME) name() string {
	return "router-name"
}

func (_ ROUTER_NAME) message() string {
	return `regular expression that must have two named captures: "origin" and "destination"`
}

func init() {
	register(ROUTER_NAME{})
}

```
