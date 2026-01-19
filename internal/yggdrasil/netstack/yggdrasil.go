package netstack

import (
	"log"
	"net"

	"github.com/yggdrasil-network/yggdrasil-go/src/core"
	"github.com/yggdrasil-network/yggdrasil-go/src/ipv6rwc"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
)

// YggdrasilNIC implements a network interface card for gVisor that connects to Yggdrasil
type YggdrasilNIC struct {
	stackRef   *YggdrasilNetstack
	ipv6rwc    *ipv6rwc.ReadWriteCloser
	dispatcher stack.NetworkDispatcher
	readBuf    []byte
	writeBuf   []byte
	rstPackets chan *stack.PacketBuffer
}

// NewYggdrasilNIC creates and registers a NIC with the gVisor stack
func (s *YggdrasilNetstack) NewYggdrasilNIC(ygg *core.Core) tcpip.Error {
	rwc := ipv6rwc.NewReadWriteCloser(ygg)
	mtu := rwc.MTU()
	nic := &YggdrasilNIC{
		stackRef:   s,
		ipv6rwc:    rwc,
		readBuf:    make([]byte, mtu),
		writeBuf:   make([]byte, mtu),
		rstPackets: make(chan *stack.PacketBuffer, 100),
	}
	if err := s.stack.CreateNIC(1, nic); err != nil {
		return err
	}

	// Start packet reader goroutine
	go func() {
		var rx int
		var err error
		for {
			rx, err = nic.ipv6rwc.Read(nic.readBuf)
			if err != nil {
				log.Println("YggdrasilNIC read error:", err)
				break
			}
			pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{
				Payload: buffer.MakeWithData(nic.readBuf[:rx]),
			})
			nic.dispatcher.DeliverNetworkPacket(ipv6.ProtocolNumber, pkb)
		}
	}()

	// Start RST packet handler goroutine
	go func() {
		for {
			pkt := <-nic.rstPackets
			if pkt == nil {
				continue
			}
			_ = nic.writePacket(pkt)
		}
	}()

	// Setup routing for Yggdrasil address space
	_, snet, err := net.ParseCIDR("0200::/7")
	if err != nil {
		return &tcpip.ErrBadAddress{}
	}
	subnet, err := tcpip.NewSubnet(
		tcpip.AddrFromSlice(snet.IP.To16()),
		tcpip.MaskFrom(string(snet.Mask)),
	)
	if err != nil {
		return &tcpip.ErrBadAddress{}
	}
	s.stack.AddRoute(tcpip.Route{
		Destination: subnet,
		NIC:         1,
	})

	// Add local address if HandleLocal is enabled
	if s.stack.HandleLocal() {
		ip := ygg.Address()
		if err := s.stack.AddProtocolAddress(
			1,
			tcpip.ProtocolAddress{
				Protocol:          ipv6.ProtocolNumber,
				AddressWithPrefix: tcpip.AddrFromSlice(ip.To16()).WithPrefix(),
			},
			stack.AddressProperties{},
		); err != nil {
			return err
		}
	}
	return nil
}

// Attach attaches the NIC to a network dispatcher
func (e *YggdrasilNIC) Attach(dispatcher stack.NetworkDispatcher) { e.dispatcher = dispatcher }

// IsAttached returns whether the NIC is attached to a dispatcher
func (e *YggdrasilNIC) IsAttached() bool { return e.dispatcher != nil }

// MTU returns the maximum transmission unit
func (e *YggdrasilNIC) MTU() uint32 { return uint32(e.ipv6rwc.MTU()) }

// SetMTU sets the MTU (not supported)
func (e *YggdrasilNIC) SetMTU(uint32) {}

// Capabilities returns the link endpoint capabilities
func (*YggdrasilNIC) Capabilities() stack.LinkEndpointCapabilities { return stack.CapabilityNone }

// MaxHeaderLength returns the maximum header length
func (*YggdrasilNIC) MaxHeaderLength() uint16 { return 40 }

// LinkAddress returns the link address
func (*YggdrasilNIC) LinkAddress() tcpip.LinkAddress { return "" }

// SetLinkAddress sets the link address (not supported)
func (*YggdrasilNIC) SetLinkAddress(tcpip.LinkAddress) {}

// Wait waits for the NIC to finish
func (*YggdrasilNIC) Wait() {}

// writePacket writes a single packet to Yggdrasil
func (e *YggdrasilNIC) writePacket(pkt *stack.PacketBuffer) tcpip.Error {
	// Recover from panic in ToView() on malformed packets
	defer func() {
		_ = recover()
	}()

	vv := pkt.ToView()
	n, err := vv.Read(e.writeBuf)
	if err != nil {
		return &tcpip.ErrAborted{}
	}
	_, err = e.ipv6rwc.Write(e.writeBuf[:n])
	if err != nil {
		return &tcpip.ErrAborted{}
	}
	return nil
}

// WritePackets writes multiple packets to Yggdrasil
func (e *YggdrasilNIC) WritePackets(list stack.PacketBufferList) (int, tcpip.Error) {
	var i int
	var err tcpip.Error
	for i, pkt := range list.AsSlice() {
		// Handle TCP RST packets asynchronously for performance
		if pkt.Data().Size() == 0 {
			if pkt.Network().TransportProtocol() == tcp.ProtocolNumber {
				tcpHeader := header.TCP(pkt.TransportHeader().Slice())
				if (tcpHeader.Flags() & header.TCPFlagRst) == header.TCPFlagRst {
					e.rstPackets <- pkt
					continue
				}
			}
		}
		err = e.writePacket(pkt)
		if err != nil {
			log.Println("YggdrasilNIC write error:", err)
			return i - 1, err
		}
	}
	return i, nil
}

// WriteRawPacket writes a raw packet (not implemented)
func (e *YggdrasilNIC) WriteRawPacket(*stack.PacketBuffer) tcpip.Error {
	panic("not implemented")
}

// ARPHardwareType returns the ARP hardware type
func (*YggdrasilNIC) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

// AddHeader adds headers to a packet
func (e *YggdrasilNIC) AddHeader(*stack.PacketBuffer) {}

// ParseHeader parses headers from a packet
func (e *YggdrasilNIC) ParseHeader(*stack.PacketBuffer) bool {
	return true
}

// Close closes the NIC
func (e *YggdrasilNIC) Close() {
	e.stackRef.stack.RemoveNIC(1)
	e.dispatcher = nil
}

// SetOnCloseAction sets a callback for when the NIC is closed
func (e *YggdrasilNIC) SetOnCloseAction(func()) {}
