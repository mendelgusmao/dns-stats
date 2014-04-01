package routers

type Thomson_DWG850_4B struct{}

func (_ Thomson_DWG850_4B) name() string {
	return "thomson-dwg850-4b"
}

func (_ Thomson_DWG850_4B) message() string {
	return `\[Host (?P<source>[^\[\]]+) (UD|TC)P (?P<origin>.*),.* --> .* ALLOW: Outbound access request \[DNS query for (?P<destination>[^\[\]]+)`
}

func init() {
	register(Thomson_DWG850_4B{})
}
