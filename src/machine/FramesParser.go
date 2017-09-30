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
)

// Initial capacity of the frames buffer.
const BUFFER_ALLOCATION_SIZE uint = 50
const FILTER_TZSP string = "udp port 37008"
const PORT_TZSP uint = 37008

// Attribute conf model.NetworkConfiguration - network configuration settings. See model.NetworkConfiguration.
// Attribute statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
type FramesParser struct {
	networkConfiguration 	*model.NetworkConfiguration
	statisticalData 		*model.StatisticalData
}

// Frame combined with the timestamp (when the frame was captured).
// Attribute Frame *gopacket.Packet - structure of network frame.
// Attribute Time time.Time - time of the frame capture.
type TimestampedFrame struct {
	Frame	*gopacket.Packet
	Time	time.Time
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
	handle := openNetworkAdapter(FramesParser.networkConfiguration)
	processFrames(FramesParser.networkConfiguration, FramesParser.statisticalData, handle)
}

// Opening of the network adapter and setting of TZSP filter.
// Parameter configuration model.NetworkConfiguration - network configuration settings. See model.NetworkConfiguration.
// Returning *pcap.Handle - frames handler. See pcap.Handle.
func openNetworkAdapter(conf *model.NetworkConfiguration) *pcap.Handle {
	configuration.Info.Println("Opening of the network adapter.")
	readTimeout := time.Duration(conf.ReadTimeout) * time.Millisecond
	handle, err01 := pcap.OpenLive(conf.AdapterName, int32(conf.MaximumFrameSize),
		true, readTimeout)
	if err01 != nil {
		configuration.Error.Panicf("Error opening device %s: %v", conf.AdapterName, err01)
	}
	configuration.Info.Println("Network adapter is open.")

	configuration.Info.Println("Setting of TZSP filter.")
	err02 := handle.SetBPFFilter(FILTER_TZSP)
	if err02 != nil {
		configuration.Error.Panicf("Error applying of TZSP filter: %v", err02)
	}
	configuration.Info.Println("TZSP filter is applied.")

	return handle
}

// Sequential processing of frames.
// Parameter configuration model.NetworkConfiguration - network configuration settings.
// Parameter statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
// Parameter handle *pcap.Handle - frames handler. See pcap.Handle.
func processFrames(conf *model.NetworkConfiguration, statisticalData *model.StatisticalData, handle *pcap.Handle) {
	configuration.Info.Println("Starting of frames processing.")
	defer handle.Close()
	tickChannel := time.Tick(time.Millisecond * time.Duration(conf.DataBuffer))
	routerMacAddress := parseStringToMacAddress(conf.RouterMacAddress)
	framesBuffer := make([](*TimestampedFrame), 0, BUFFER_ALLOCATION_SIZE)
	framesSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for frame := range framesSource.Packets() {
		frameData := frame
		select {
		case <- tickChannel:
			// after data buffer time (ms), buffer is sent to next processing on the way to the database ...
			framesBuffer = append(framesBuffer, &TimestampedFrame{Frame: &frameData, Time: time.Now()})
			go sendDataToDatabase(statisticalData, framesBuffer, routerMacAddress)
			framesBuffer = [](*TimestampedFrame){}
		default:
			framesBuffer = append(framesBuffer, &TimestampedFrame{Frame: &frameData, Time: time.Now()})
		}
	}
	configuration.Info.Println("Processing of the frames ended.")
}

// Forming of raw data and sending of frames collections to database manager.
// Parameter statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
// Parameter timestampedFrames [](*TimestampedFrame - slice of frames tagged with timestamp. See TimestampedFrame.
func sendDataToDatabase(statisticalData *model.StatisticalData, timestampedFrames [](*TimestampedFrame),
	routerMacAddress *([]byte)) {
	rawData := make([](*model.RawData), 0)
	for _, timeFrame := range timestampedFrames {
		udpLayer := (*timeFrame.Frame).Layer(layers.LayerTypeUDP)
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
					if bytes.Compare(srcAddress, *routerMacAddress) == 0 {
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
				// reading of ports from TCP header
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
				rawData = append(rawData, &model.RawData{
					Time: timeFrame.Time,
					SrcPort: srcPort,
					DstPort: dstPort,
					NetworkProtocol: networkProtocol,
					TransportProtocol: transportProtocol,
					Bytes: uint(len(originalData)),
					Direction: direction,
				})
			}
		}
	}
	if len(rawData) != 0 {
		statisticalData.WriteNewDataEntries(&rawData)
	}
}

// Converting of string to MAC address (byte array format).
// Parameter macAddress string - string representation of MAC address.
// Returning *([]byte) - byte array representation of MAC address.
func parseStringToMacAddress(macAddress string) *([]byte) {
	hw, err := net.ParseMAC(macAddress)
	if err != nil {
		configuration.Error.Panicf("Error reading of router's MAC address %s: %v", macAddress, err)
	}
	array := []byte(hw)
	return &array
}