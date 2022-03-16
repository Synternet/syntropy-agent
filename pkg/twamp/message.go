package twamp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
)

func sendMessage(conn net.Conn, msg interface{}) error {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, msg)
	if err != nil {
		return err
	}

	size := buf.Len()
	n, err := conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	if size != n {
		return errors.New("could not send message")
	}

	return nil
}

func receiveMessage(conn net.Conn, msg interface{}) error {
	buf := make([]byte, binary.Size(msg))
	_, err := conn.Read(buf)
	if err != nil {
		return err
	}

	reader := bytes.NewBuffer(buf)
	err = binary.Read(reader, binary.BigEndian, msg)
	if err != nil {
		return err
	}

	return nil
}
