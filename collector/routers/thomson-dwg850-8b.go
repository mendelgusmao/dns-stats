package routers

type Thomson_DWG850_8B struct{}

func (_ Thomson_DWG850_8B) name() string {
	return "thomson-dwg850-8b"
}

func (_ Thomson_DWG850_8B) message() string {
	return `(UD|TC)P (?P<origin>.*),.* --> .* ALLOW: Outbound access request \[DNS query for (?P<destination>[^\[\]]+)`
}

func init() {
	register(Thomson_DWG850_8B{})
}
