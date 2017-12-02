package machine

import (
	"model"
	"configuration"
	"github.com/google/gopacket/pcap"
	"time"
	"github.com/google/gopacket"
	"net"
	"encoding/binary"
	"bytes"
)

// Initial capacity of the frames buffer.
const STARTING_MAP_SIZE uint = 8000
const BUFFER_MAX_SIZE uint = 250000
const FILTER_TZSP = "udp port 37008"
const TAG_TYPE_PADDING = byte(0x00)
const TAG_TYPE_END = byte(0x01)
const ETHER_TYPE_IPV4 = uint16(2048)
const ETHER_TYPE_IPV6 = uint16(34525)
const PROTOCOL_UDP = uint8(17)
const PROTOCOL_TCP = uint8(6)
const PORT_TZSP = uint16(37008)

// Attribute routerMacAddress *([]byte) - MAC address of monitored router's interface.
// Attribute conf model.NetworkConfiguration - network configuration settings. See model.NetworkConfiguration.
// Attribute statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
// Attribute handler *pcap.Handle - incoming frames handler. See pcap.Handle.
type FramesParser struct {
	routerMacAddress		*([]byte)
	networkConfiguration 	*model.NetworkConfiguration
	statisticalData 		*model.StatisticalData
	handler					*pcap.Handle
}

// Creating instance of the FramesParser.
// Parameter conf model.NetworkConfiguration - network configuration settings. See model.NetworkConfiguration.
// Parameter statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
// Returning *FramesParser - FramesParser object.
func NewFramesParser(conf *model.NetworkConfiguration, statisticalData *model.StatisticalData) *FramesParser {
	framesParser := FramesParser {
		networkConfiguration: conf,
		statisticalData: statisticalData,
	}
	return &framesParser
}

// Starting of the frames capturing under selected network configuration.
func (FramesParser *FramesParser) StartCapturing() {
	FramesParser.readRouterMacAddress()
	FramesParser.openNetworkAdapter()
	FramesParser.processFrames()
}

// Converting of string to MAC address (byte array format).
func (FramesParser *FramesParser) readRouterMacAddress() {
	macAddress := FramesParser.networkConfiguration.RouterMacAddress
	hw, err := net.ParseMAC(macAddress)
	if err != nil {
		configuration.Error.Panicf("Error reading of router's MAC address %s: %v", macAddress, err)
	}
	array := []byte(hw)
	FramesParser.routerMacAddress = &array
}

// Opening of the network adapter and setting of TZSP filter.
func (FramesParser *FramesParser) openNetworkAdapter() {
	configuration.Info.Println("Opening of the network adapter.")
	readTimeout := time.Duration(FramesParser.networkConfiguration.ReadTimeout) * time.Millisecond
	handler, err01 := pcap.OpenLive(FramesParser.networkConfiguration.AdapterName,
		int32(FramesParser.networkConfiguration.MaximumFrameSize),
		false, readTimeout)
	if err01 != nil {
		configuration.Error.Panicf("Error opening device %s: %v",
			FramesParser.networkConfiguration.AdapterName, err01)
	}
	configuration.Info.Println("Network adapter is open.")

	configuration.Info.Println("Setting of TZSP filter.")
	err02 := handler.SetBPFFilter(FILTER_TZSP)
	if err02 != nil {
		configuration.Error.Panicf("Error applying of TZSP filter: %v", err02)
	}
	configuration.Info.Println("TZSP filter is applied.")
	FramesParser.handler = handler
}

// Sequential processing of frames.
func (FramesParser *FramesParser) processFrames() {
	configuration.Info.Println("Starting of frames processing.")
	go func() {
		framesRing := make([](*[]byte), BUFFER_MAX_SIZE)
		actualRingSize := uint(0)
		tickChannel := time.Tick(time.Millisecond * time.Duration(FramesParser.networkConfiguration.DataBuffer))
		handler := FramesParser.handler
		defer handler.Close()
		framesSource := gopacket.NewPacketSource(handler, handler.LinkType())
		framesSource.Lazy = true
		for frame := range framesSource.Packets() {
			frameData := frame.Data()
			framesRing[actualRingSize] = &frameData
			select {
			case <- tickChannel:
				go FramesParser.processFramesBucket(framesRing, actualRingSize)
				actualRingSize = uint(0)
			default:
				actualRingSize++
			}
		}
		configuration.Info.Println("Frames processing finished.")
	}()
}

// Processing of frames bucket by using aggregation on bytes over same raw data types.
// Parameter buffer []*gopacket.Packet - buffered network frames.
func (FramesParser *FramesParser) processFramesBucket(frames [](*[]byte), size uint) {
	repository := make(map[model.RawDataType](*model.RawData), STARTING_MAP_SIZE)
	for i:=uint(0); i<size+1; i++ {
		// template
		rawDataType := model.RawDataType{
			NetworkProtocol:   0,
			TransportProtocol: 0,
			SrcPort:           0,
			DstPort:           0,
			Direction:         0,
		}
		startIndex := uint(0)
		// ethernet 2
		originalFrame := unwrapTzsp(frames[i])
		originalFrameX := *originalFrame
		length := len(originalFrameX)
		if length >= 14 {
			ethertype := []byte{originalFrameX[startIndex + 12], originalFrameX[startIndex + 13]}
			ethertypeU := binary.BigEndian.Uint16(ethertype)
			sourceAddress := originalFrameX[startIndex + 6 : startIndex + 12]
			if bytes.Equal(sourceAddress, *(FramesParser.routerMacAddress)) {
				rawDataType.Direction = 1
			}
			startIndex = uint(14)
			rawDataType.NetworkProtocol = uint(ethertypeU)
			// ipv4
			if length >= 34 && ethertypeU == ETHER_TYPE_IPV4 {
				ihl := originalFrameX[startIndex] & 0x0f
				ihlU := uint8(ihl) * 4
				protocol := originalFrameX[startIndex + 9]
				protocolU := uint8(protocol)
				startIndex += uint(ihlU)
				rawDataType.TransportProtocol = uint(protocolU)
				// tcp or udp
				if (length >= 42 && protocolU == PROTOCOL_UDP) || (length >= 54 && protocolU == PROTOCOL_TCP) {
					sourcePort := []byte{originalFrameX[startIndex], originalFrameX[startIndex + 1]}
					destinationPort := []byte{originalFrameX[startIndex + 2], originalFrameX[startIndex + 3]}
					sourcePortU := binary.BigEndian.Uint16(sourcePort)
					destinationPortU := binary.BigEndian.Uint16(destinationPort)
					rawDataType.SrcPort = uint(sourcePortU)
					rawDataType.DstPort = uint(destinationPortU)
				}
				// ipv6
			} else if length >= 54 && ethertypeU == ETHER_TYPE_IPV6 {
				nextHeader := originalFrameX[startIndex + 6]
				nextHeaderU := uint8(nextHeader)
				startIndex += uint(40)
				rawDataType.TransportProtocol = uint(nextHeaderU)
				// tcp or udp
				if (length >= 62 && nextHeaderU == PROTOCOL_UDP) || (length >= 74 && nextHeaderU == PROTOCOL_TCP) {
					sourcePort := []byte{originalFrameX[startIndex], originalFrameX[startIndex + 1]}
					destinationPort := []byte{originalFrameX[startIndex + 2], originalFrameX[startIndex + 3]}
					sourcePortU := binary.BigEndian.Uint16(sourcePort)
					destinationPortU := binary.BigEndian.Uint16(destinationPort)
					rawDataType.SrcPort = uint(sourcePortU)
					rawDataType.DstPort = uint(destinationPortU)
				}
			}
			// increasing of counters
			_, present := repository[rawDataType]
			if present == true {
				data := repository[rawDataType]
				newBytes := data.Bytes + uint(len(originalFrameX))
				actualTime := time.Now()
				newDataEntry := model.RawData{
					Bytes: newBytes,
					Time: actualTime,
					RawDataType: &rawDataType,
				}
				repository[rawDataType] = &newDataEntry
			} else {
				rawData := model.RawData{
					Bytes: uint(len(originalFrameX)),
					Time: time.Now(),
					RawDataType: &rawDataType,
				}
				repository[rawDataType] = &rawData
			}
		}
	}
	// if there are some entries, sent them to DB
	if len(repository) != 0 {
		// building of slice from map and sending of slice to DB
		slice := make([](*model.RawData), len(repository))
		i := uint(0)
		for  _, value := range repository {
			slice[i] = value
			i++
		}
		go FramesParser.statisticalData.WriteNewDataEntries(&slice)
	}
}

// Unwrapping of TZSP datagram.
// Parameter frame *[]byte - original frame.
// Returning *[]byte - unwrapped original frame.
func unwrapTzsp(frame *[]byte) *[]byte {
	startIndex := uint(0)
	framex := *frame
	length := uint(len(framex))
	if length > 14 {
		ethertype := []byte{framex[startIndex + 12], framex[startIndex + 13]}
		ethertypeU := binary.BigEndian.Uint16(ethertype)
		startIndex = uint(14)
		if length >= 34 && ethertypeU == ETHER_TYPE_IPV4 {
			ihl := framex[startIndex] & 0x0f
			ihlU := uint8(ihl) * 4
			protocol := framex[startIndex + 9]
			protocolU := uint8(protocol)
			startIndex += uint(ihlU)
			if length >= 14 + uint(ihlU) && protocolU == PROTOCOL_UDP {
				sourcePort := []byte{framex[startIndex], framex[startIndex+1]}
				destinationPort := []byte{framex[startIndex+2], framex[startIndex+3]}
				sourcePortU := binary.BigEndian.Uint16(sourcePort)
				destinationPortU := binary.BigEndian.Uint16(destinationPort)
				if length >= 22+uint(ihlU) && (sourcePortU == PORT_TZSP || destinationPortU == PORT_TZSP) {
					startIndex += uint(8)
					startIndex = getTzspPayloadIndex(frame, startIndex)
					cutFrameX := framex[startIndex:]
					return &cutFrameX
				}
			}
		} else if length >= 54 && ethertypeU == ETHER_TYPE_IPV6 {
			nextHeader := framex[startIndex + 6]
			nextHeaderU := uint8(nextHeader)
			startIndex += uint(40)
			if length >= 62 && nextHeaderU == PROTOCOL_UDP {
				sourcePort := []byte{framex[startIndex], framex[startIndex + 1]}
				destinationPort := []byte{framex[startIndex + 2], framex[startIndex + 3]}
				sourcePortU := binary.BigEndian.Uint16(sourcePort)
				destinationPortU := binary.BigEndian.Uint16(destinationPort)
				if length >= 67 && (sourcePortU == PORT_TZSP || destinationPortU == PORT_TZSP) {
					startIndex += uint(8)
					startIndex = getTzspPayloadIndex(frame, startIndex)
					cutFrameX := framex[startIndex:]
					return &cutFrameX
				}
			}
		}
	}
	return nil
}

// Reading of index of first data byte wrapped in TZSP datagram.
// Parameter tzspDatagram *[]byte - TZSP datagram.
// Parameter nextIndex uint - Index from which we would like to read TZSP tag type.
// Returning uint - Final index of first data byte.
func getTzspPayloadIndex(tzspDatagram *[]byte, nextIndex uint) uint {
	datagram := *tzspDatagram
	tagType := datagram[nextIndex]
	nextIndex = nextIndex + 4
	if tagType == TAG_TYPE_PADDING {
		index := nextIndex + 1
		getTzspPayloadIndex(tzspDatagram, index)
	} else if tagType == TAG_TYPE_END {
		index := nextIndex + 1
		return index
	} else {
		tagLength := uint(datagram[1])
		index := nextIndex + tagLength + 2
		return getTzspPayloadIndex(tzspDatagram, index)
	}
	return uint(0)
}