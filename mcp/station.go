package mcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	SUB_HEADER = "5000" // 3Eフレームでは固定

	HEALTH_CHECK_COMMAND    = "1906" // binary mode expression. if ascii mode then 0619
	HEALTH_CHECK_SUBCOMMAND = "0000"

	READ_COMMAND         = "0104" // binary mode expression. if ascii mode then 0401
	READ_SUB_COMMAND     = "0000"
	BIT_READ_SUB_COMMAND = "0100"

	WRITE_COMMAND         = "0114" // binary mode expression. if ascii mode then 1401
	WRITE_SUB_COMMAND     = "0000"
	BIT_WRITE_SUB_COMMAND = "0100"

	MONITORING_TIMER = "1000" // 3[sec]
)

// DeviceCodes is device name and hex value map
var DeviceCodes = map[string]string{
	"X": "9C",
	"Y": "9D",
	"M": "90",
	"L": "92",
	"F": "93",
	"V": "94",
	"B": "A0",
	"W": "B4",
	"D": "A8",
}

type Station interface {
	BuildHealthCheckRequest() string
	BuildBitReadRequest(deviceName string, offset, numPoints int64) string
	BuildReadRequest(deviceName string, offset, numPoints int64) string
	//BuildBatchReadRequest(deviceName string, offset, numPoints int64) string
	//BuildRandomReadRequest(...)
	BuildBitWriteRequest(deviceName string, offset, numPoints int64, writeData []byte) string
	BuildWriteRequest(deviceName string, offset, numPoints int64, writeData []byte) string
}

// Each single PLC that is connected on MELSECNET and CC-Link IE is called a station.
type station3E struct {
	// PLC Network number - not used in 1E Frame
	networkNum string
	// PC Number
	pcNum string
	// PLC stn Unit I/O Number - not used in 1E Frame
	unitIONum string
	// PLC stn Unit Station Number - not used in 1E Frame
	unitStationNum string
}

func NewStation(networkNum, pcNum, unitIONum, unitStationNum string, frameVersion FrameVersion) (Station, error) {
	switch frameVersion {
	case Frame1E:
		return &station1E{
			pcNum: pcNum,
		}, nil
	case Frame3E:
		return &station3E{
			networkNum:     networkNum,
			pcNum:          pcNum,
			unitIONum:      unitIONum,
			unitStationNum: unitStationNum,
		}, nil
	}
	return nil, fmt.Errorf("Cannot create station for unhandled frameVersion %s", frameVersion)
}

func (h *station3E) BuildHealthCheckRequest() string {

	returnDataNum := "0500"    // 5 device. if ascii mode then 0005
	returnData := "4142434445" // value is "ABCDE".

	requestStr := HEALTH_CHECK_COMMAND + HEALTH_CHECK_SUBCOMMAND + returnDataNum + returnData

	// data length
	requestCharLen := len(MONITORING_TIMER+requestStr) / 2 // 1byte=2char
	dataLenBuff := new(bytes.Buffer)
	_ = binary.Write(dataLenBuff, binary.LittleEndian, int64(requestCharLen))
	dataLen := fmt.Sprintf("%X", dataLenBuff.Bytes()[0:2]) // 2byte固定

	return SUB_HEADER +
		h.networkNum +
		h.pcNum +
		h.unitIONum +
		h.unitStationNum +
		dataLen +
		MONITORING_TIMER +
		requestStr
}

// BuildReadRequest represents MCP read as word command.
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// numPoints is number of read device points.
func (h *station3E) BuildReadRequest(deviceName string, offset, numPoints int64) string {
	return h.buildReadRequestHelper(deviceName, offset, numPoints, READ_SUB_COMMAND)
}

// BuildReadRequest represents MCP read as bit command.
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// numPoints is number of read device points.
func (h *station3E) BuildBitReadRequest(deviceName string, offset, numPoints int64) string {
	return h.buildReadRequestHelper(deviceName, offset, numPoints, BIT_READ_SUB_COMMAND)
}

func (h *station3E) buildReadRequestHelper(deviceName string, offset, numPoints int64, subCommand string) string {
	// get device symbol hex layout
	deviceCode := DeviceCodes[deviceName]

	// offset convert to little endian layout
	// MELSECコミュニケーションプロトコル リファレンス(p67) MELSEC-Q/L: 3[byte], MELSEC iQ-R: 4[byte]
	offsetBuff := new(bytes.Buffer)
	_ = binary.Write(offsetBuff, binary.LittleEndian, offset)
	offsetHex := fmt.Sprintf("%X", offsetBuff.Bytes()[0:3]) // 仮にQシリーズとするので3byte trim

	// read points
	pointsBuff := new(bytes.Buffer)
	_ = binary.Write(pointsBuff, binary.LittleEndian, numPoints)
	points := fmt.Sprintf("%X", pointsBuff.Bytes()[0:2]) // 2byte固定

	// data length
	requestCharLen := len(MONITORING_TIMER+READ_COMMAND+READ_SUB_COMMAND+deviceCode+offsetHex+points) / 2 // 1byte=2char
	dataLenBuff := new(bytes.Buffer)
	_ = binary.Write(dataLenBuff, binary.LittleEndian, int64(requestCharLen))
	dataLen := fmt.Sprintf("%X", dataLenBuff.Bytes()[0:2]) // 2byte固定

	return SUB_HEADER +
		h.networkNum +
		h.pcNum +
		h.unitIONum +
		h.unitStationNum +
		dataLen +
		MONITORING_TIMER +
		READ_COMMAND +
		subCommand +
		offsetHex +
		deviceCode +
		points
}

func (h *station3E) BuildWriteRequest(deviceName string, offset, numPoints int64, writeData []byte) string {
	return h.buildWriteRequestHelper(deviceName, offset, numPoints, writeData, WRITE_SUB_COMMAND)
}

func (h *station3E) BuildBitWriteRequest(deviceName string, offset, numPoints int64, writeData []byte) string {
	return h.buildWriteRequestHelper(deviceName, offset, numPoints, writeData, BIT_WRITE_SUB_COMMAND)
}

// BuildWriteRequest represents MCP write command.
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// writeData is data to write.
// numPoints is number of write device points.
// writeData is the data to be written. If writeData is larger than 2*numPoints bytes,
// data larger than 2*numPoints bytes is ignored.
func (h *station3E) buildWriteRequestHelper(deviceName string, offset, numPoints int64, writeData []byte, subCommand string) string {
	// get device symbol hex layout
	deviceCode := DeviceCodes[deviceName]

	// offset convert to little endian layout
	// MELSECコミュニケーションプロトコル リファレンス(p67) MELSEC-Q/L: 3[byte], MELSEC iQ-R: 4[byte]
	offsetBuff := new(bytes.Buffer)
	_ = binary.Write(offsetBuff, binary.LittleEndian, offset)
	offsetHex := fmt.Sprintf("%X", offsetBuff.Bytes()[0:3]) // 仮にQシリーズとするので3byte trim

	// convert write data to little endian word
	writeBuff := new(bytes.Buffer)
	_ = binary.Write(writeBuff, binary.LittleEndian, writeData)
	writeHex := fmt.Sprintf("%X", writeBuff.Bytes()[0:2*numPoints]) // 2 byte per 1 device point

	// write points
	pointsBuff := new(bytes.Buffer)
	_ = binary.Write(pointsBuff, binary.LittleEndian, numPoints)
	points := fmt.Sprintf("%X", pointsBuff.Bytes()[0:2]) // 2byte固定

	// data length
	requestCharLen := len(MONITORING_TIMER+WRITE_COMMAND+WRITE_SUB_COMMAND+deviceCode+offsetHex+points+writeHex) / 2 // 1byte=2char
	dataLenBuff := new(bytes.Buffer)
	_ = binary.Write(dataLenBuff, binary.LittleEndian, int64(requestCharLen))
	dataLen := fmt.Sprintf("%X", dataLenBuff.Bytes()[0:2]) // 2byte固定
	return SUB_HEADER +
		h.networkNum +
		h.pcNum +
		h.unitIONum +
		h.unitStationNum +
		dataLen +
		MONITORING_TIMER +
		WRITE_COMMAND +
		subCommand +
		offsetHex +
		deviceCode +
		points +
		writeHex
}

type station1E struct {
	// PC Number
	pcNum string
}

func (h *station1E) BuildWriteRequest(deviceName string, offset, numPoints int64, writeData []byte) string {
	return h.buildWriteRequestHelper(deviceName, offset, numPoints, writeData, WRITE_SUB_COMMAND)
}

func (h *station1E) BuildBitWriteRequest(deviceName string, offset, numPoints int64, writeData []byte) string {
	return h.buildWriteRequestHelper(deviceName, offset, numPoints, writeData, BIT_WRITE_SUB_COMMAND)
}
