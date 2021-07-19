package controller

type Controller interface {
	Start(c chan []byte)
	Stop()
}
