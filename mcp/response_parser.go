package mcp

import (
	"errors"
	"fmt"
)

type Parser interface {
	Process(resp []byte) (*Response, error)
}

type parser_3E struct {
}

type parser_1E struct {
}

func NewParser(frameVersion FrameVersion) Parser {
	switch frameVersion {
	case Frame1E:
		return &parser_1E{}
	case Frame3E:
		return &parser_3E{}
	default:
		return nil
	}
}

// Response represents mcp response
type Response struct {
	// Sub header
	SubHeader string
	// network number
	NetworkNum string
	// PC number
	PCNum string
	// Request Unit I/O number
	UnitIONum string
	// Request Unit station number
	UnitStationNum string
	// Response data length
	DataLen string
	// Response data code
	EndCode string
	// Response data
	Payload []byte
	// error data
	ErrInfo []byte
}

func (p *parser_3E) Process(resp []byte) (*Response, error) {
	if len(resp) < 22 {
		return nil, errors.New("length must be larger than 22 byte")
	}

	subHeaderB := resp[0:2]
	networkNumB := resp[2:3]
	pcNumB := resp[3:4]
	unitIONumB := resp[4:6]
	unitStationNumB := resp[6:7]
	dataLenB := resp[7:9]
	endCodeB := resp[9:11]
	payloadB := resp[11:]

	return &Response{
		SubHeader:      fmt.Sprintf("%X", subHeaderB),
		NetworkNum:     fmt.Sprintf("%X", networkNumB),
		PCNum:          fmt.Sprintf("%X", pcNumB),
		UnitIONum:      fmt.Sprintf("%X", unitIONumB),
		UnitStationNum: fmt.Sprintf("%X", unitStationNumB),
		DataLen:        fmt.Sprintf("%X", dataLenB),
		EndCode:        fmt.Sprintf("%X", endCodeB),
		Payload:        payloadB,
	}, nil
}

//Processes the raw response with the 1E Frame Format.
//here we only have
func (p *parser_1E) Process(resp []byte) (*Response, error) {
	if len(resp) < 2 {
		return nil, errors.New("length must be larger than 2 bytes")
	}

	if len(resp) == 2 {
		return nil, fmt.Errorf("PLC returned an error code: %X", resp[0:2])
	}

	return &Response{
		SubHeader: fmt.Sprintf("%X", resp[0]),
		EndCode:   fmt.Sprintf("%X", resp[1]),
		Payload:   resp[2:],
	}, nil
}
