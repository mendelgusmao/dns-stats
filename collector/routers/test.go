package routers

type Test struct{}

func (_ Test) name() string {
	return "test"
}

func (_ Test) message() string {
	return `dns-stats (?P<source>.*),(?P<origin>.*),(?P<destination>.*)`
}

func init() {
	register(Test{})
}
