package machine

import (
	"model"
	"configuration"
	"github.com/google/gopacket/pcap"
	"time"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"bytes"
	"net"
	"fmt"
)

// Initial capacity of the frames buffer.
const BUFFER_ALLOCATION_SIZE uint = 50
const FILTER_TZSP string = "udp port 37008"
const PORT_TZSP uint = 37008

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
		true, readTimeout)
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
	handler := FramesParser.handler
	defer handler.Close()
	framesBuffer := make([](*gopacket.Packet), 0, BUFFER_ALLOCATION_SIZE)
	tickChannel := time.Tick(time.Millisecond * time.Duration(FramesParser.networkConfiguration.DataBuffer))
	framesSource := gopacket.NewPacketSource(handler, handler.LinkType())
	for frame := range framesSource.Packets() {
		frameData := frame
		select {
		case <- tickChannel:
			// after data buffer time (ms), buffer is sent to next processing on the way to the database ...
			framesBuffer = append(framesBuffer, &frameData)
			go FramesParser.processFramesBucket(framesBuffer)
			framesBuffer = [](*gopacket.Packet){}
		default:
			framesBuffer = append(framesBuffer, &frameData)
		}
	}
	configuration.Info.Println("Processing of the frames ended.")
}

// Processing of frames bucket by using aggregation on bytes over same raw data types.
// Parameter buffer []*gopacket.Packet - buffered network frames.
func (FramesParser *FramesParser) processFramesBucket(buffer []*gopacket.Packet) {
	repository := make(map[model.RawDataType](*model.RawData))
	for _, frame := range buffer {
		udpLayer := (*frame).Layer(layers.LayerTypeUDP)
		if udpLayer != nil {
			udp, _ := udpLayer.(*layers.UDP)
			dstPort := uint(udp.DstPort)
			if dstPort == PORT_TZSP {
				// declaration of final protocols entities
				var networkProtocol, transportProtocol, srcPort, dstPort, direction uint = 0, 0, 0, 0, 0
				// reading of TZSP header && dispatching of original frame
				payload := udp.LayerPayload()
				taggedFields := uint(payload[4])
				var firstDataIndex uint = 0
				if taggedFields == 0 || taggedFields == 1 {
					firstDataIndex = 5
				} else {
					additionalLength := uint(payload[5])
					firstDataIndex = additionalLength + 6
				}
				originalData := payload[firstDataIndex:]
				packet := gopacket.NewPacket(originalData, layers.LayerTypeEthernet, gopacket.Default)

				// reading of the ethertype (network protocol)
				ethernetLayer := (packet).Layer(layers.LayerTypeEthernet)
				if ethernetLayer != nil {
					ethernetPacket, _ := ethernetLayer.(*layers.Ethernet)
					networkProtocol = uint(ethernetPacket.EthernetType)
					srcAddress := []byte(ethernetPacket.SrcMAC)
					if bytes.Compare(srcAddress, *FramesParser.routerMacAddress) == 0 {
						direction = 1
					}
				}
				// reading of the transport protocol (from IPv4 header)
				ipv4Layer := (packet).Layer(layers.LayerTypeIPv4)
				if ipv4Layer != nil {
					ip, _ := ipv4Layer.(*layers.IPv4)
					transportProtocol = uint(ip.Protocol)
				}
				// reading of the transport protocol (from IPv6 header)
				ipv6Layer := (packet).Layer(layers.LayerTypeIPv6)
				if ipv6Layer != nil {
					ip, _ := ipv6Layer.(*layers.IPv6)
					transportProtocol = uint(ip.NextHeader)
				}
				// reading of port from TCP header
				tcpLayer := (packet).Layer(layers.LayerTypeTCP)
				if tcpLayer != nil {
					tcp, _ := tcpLayer.(*layers.TCP)
					srcPort = uint(tcp.SrcPort)
					dstPort = uint(tcp.DstPort)
				}
				// reading of port from UDP header
				udpLayer := (packet).Layer(layers.LayerTypeUDP)
				if udpLayer != nil {
					udp, _ := udpLayer.(*layers.UDP)
					srcPort = uint(udp.SrcPort)
					dstPort = uint(udp.DstPort)
				}
				// modelling of raw data element
				rawDataType := model.RawDataType{
					NetworkProtocol: networkProtocol,
					TransportProtocol: transportProtocol,
					SrcPort: srcPort,
					DstPort: dstPort,
					Direction: direction,
				}
				_, present := repository[rawDataType]
				if present == true {
					data := repository[rawDataType]
					data.Bytes += uint(len(originalData))
					data.Time = time.Now()
					repository[rawDataType] = data
				} else {
					rawData := model.RawData{
						Bytes: uint(len(originalData)),
						Time: time.Now(),
						RawDataType: &rawDataType,
					}
					repository[rawDataType] = &rawData
				}
			}
		}
	}
	// if there are some entries ...
	if len(repository) != 0 {
		// building of slice from map and sending of slice to DB
		slice := make([](*model.RawData), 0, len(repository))
		for  _, value := range repository {
			slice = append(slice, value)
		}
		FramesParser.statisticalData.WriteNewDataEntries(&slice)
		fmt.Println("OK")
	}
}