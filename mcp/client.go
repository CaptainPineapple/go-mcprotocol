package mcp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"
)

type Client interface {
	Read(deviceName string, offset, numPoints int64) ([]byte, error)
	BitRead(deviceName string, offset, numPoints int64) ([]byte, error)
	Write(deviceName string, offset, numPoints int64, writeData []byte) ([]byte, error)
	BitWrite(deviceName string, offset, numPoints int64, writeData []byte) ([]byte, error)
	HealthCheck() error
	ShutDown()
	Reconnect() error
	Connect() error
}

// client3E is 3E frame mcp client
type client3E struct {
	// PLC address
	tcpAddr string //*net.TCPAddr
	// PLC station
	stn *station
	// Connection Handle to PLC
	conn *net.TCPConn
}

func New3EClient(host string, port int, stn *station, keep_alive bool) (Client, error) {
	//tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", host, port))
	// if err != nil {
	// 	return nil, err
	// }
	newClient := client3E{tcpAddr: fmt.Sprintf("%v:%v", host, port), stn: stn}
	err := newClient.Connect()
	if err != nil {
		return nil, err
	}
	//newClient.conn.SetKeepAlive(keep_alive)

	return &newClient, nil
}

// MELSECコミュニケーションプロトコル p180
// 11.4折返しテスト
func (c *client3E) HealthCheck() error {
	requestStr := c.stn.BuildHealthCheckRequest()

	// binary protocol
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return err
	}

	// Send message
	if _, err = c.conn.Write(payload); err != nil {
		return err
	}

	// Receive message
	readBuff := make([]byte, 30)
	readLen, err := c.conn.Read(readBuff)
	if err != nil {
		return err
	}

	resp := readBuff[:readLen]

	if readLen != 18 {
		return errors.New("plc connect test is fail: return length is [" + fmt.Sprintf("%X", resp) + "]")
	}

	// decodeString is 折返しデータ数ヘッダ[1byte]
	if "0500" != fmt.Sprintf("%X", resp[11:13]) {
		return errors.New("plc connect test is fail: return header is [" + fmt.Sprintf("%X", resp[11:13]) + "]")
	}

	//  折返しデータ[5byte]=ABCDE
	if "4142434445" != fmt.Sprintf("%X", resp[13:18]) {
		return errors.New("plc connect test is fail: return body is [" + fmt.Sprintf("%X", resp[13:18]) + "]")
	}

	return nil
}

func (c *client3E) Connect() error {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.Dial("tcp", c.tcpAddr)
	if err != nil {
		return err
	}

	c.conn, _ = conn.(*net.TCPConn)
	return nil
}

func (c *client3E) Reconnect() error {
	c.ShutDown()
	time.Sleep(1 * time.Second)
	return c.Connect()
}

// Read is send read as word command to remote plc by mc protocol
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// numPoints is number of read device points.
func (c *client3E) Read(deviceName string, offset, numPoints int64) ([]byte, error) {
	return c.readHelper(c.stn.BuildReadRequest(deviceName, offset, numPoints), numPoints)
}

// BitRead is send read as bit command to remote plc by mc protocol
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// numPoints is number of read device points.
// results of payload of BitRead will return []byte contains 0, 1, 16 or 17(hex encoded 00, 01, 10, 11)
func (c *client3E) BitRead(deviceName string, offset, numPoints int64) ([]byte, error) {
	return c.readHelper(c.stn.BuildBitReadRequest(deviceName, offset, numPoints), numPoints)
}

func (c *client3E) readHelper(requestStr string, numPoints int64) ([]byte, error) {
	// TODO binary protocol
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return nil, err
	}

	// Send message
	if _, err = c.conn.Write(payload); err != nil {
		return nil, err
	}

	// Receive message
	readBuff := make([]byte, 22+2*numPoints) // 22 is response header size. [sub header + network num + unit i/o num + unit station num + response length + response code]
	readLen, err := c.conn.Read(readBuff)
	if err != nil {
		return nil, err
	}

	return readBuff[:readLen], nil
}

// Write is send write command to remote plc by mc protocol
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// writeData is data to write.
// numPoints is number of write device points.
// writeData is the data to be written. If writeData is larger than 2*numPoints bytes,
// data larger than 2*numPoints bytes is ignored.
func (c *client3E) Write(deviceName string, offset, numPoints int64, writeData []byte) ([]byte, error) {
	return c.writeHelper(c.stn.BuildWriteRequest(deviceName, offset, numPoints, writeData))
}

func (c *client3E) BitWrite(deviceName string, offset, numPoints int64, writeData []byte) ([]byte, error) {
	return c.writeHelper(c.stn.BuildBitWriteRequest(deviceName, offset, numPoints, writeData))
}

func (c *client3E) writeHelper(requestStr string) ([]byte, error) {
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return nil, err
	}
	// Send message
	if _, err = c.conn.Write(payload); err != nil {
		return nil, err
	}

	// Receive message
	readBuff := make([]byte, 22) // 22 is response header size. [sub header + network num + unit i/o num + unit station num + response length + response code]

	readLen, err := c.conn.Read(readBuff)
	if err != nil {
		return nil, err
	}
	return readBuff[:readLen], nil
}

func (c *client3E) ShutDown() {
	c.conn.Close()
}
