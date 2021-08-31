package router

type bits uint8

const (
	agent  bits = 1 << iota // agent flag markes route added by this app
	active                  // active show that this route is curretly enabled
)

func (flags bits) set(b bits) {
	flags |= b
}

func (flags bits) clear(b bits) {
	flags &= ^b
}

func (flags bits) has(b bits) bool {
	return flags&b == b
}
