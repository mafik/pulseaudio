// Package pulseaudio provides functions for interacting with the PulseAudio server over the native protocol.
//
// Sample usage:
//
// package main
//
// import (
// 	"fmt"
//
// 	"mrogalski.eu/go/pulseaudio"
// )
//
// func main() {
// 	c, err := pulseaudio.NewClient("golang")
// 	if err != nil {
// 		fmt.Println("Error when creating PulseAudio client:", err)
// 		return
// 	}
// 	fmt.Println(c.Sinks()[0].GetVolume())
// 	c.Close()
// }
package pulseaudio // import "mrogalski.eu/go/pulseaudio"

// #cgo pkg-config: libpulse
// #include <pulse/pulseaudio.h>
// #include "pulsego.h"
import "C"
import (
	"fmt"
	"unsafe"
)

// Client maintains a connection to the PulseAudio server.
type Client struct {
	mainloop    *C.struct_pa_mainloop
	mainloopAPI *C.struct_pa_mainloop_api
	context     *C.struct_pa_context
	updates     chan func()
}

// NewClient establishes a connection to the PulseAudio server.
func NewClient(name string) (*Client, error) {
	mainloop := C.pa_mainloop_new()
	mainloopAPI := C.pa_mainloop_get_api(mainloop)
	cname := C.CString(name)
	context := C.pa_context_new(mainloopAPI, cname)

	if C.pa_context_connect(context, nil, C.PA_CONTEXT_NOFLAGS, nil) < 0 {
		errno := C.pa_context_errno(context)
		desc := C.GoString(C.pa_strerror(errno))
		return nil, fmt.Errorf("Connection error: %s", desc)
	}
loop:
	for {
		switch C.pa_context_get_state(context) {
		case C.PA_CONTEXT_CONNECTING:
		case C.PA_CONTEXT_AUTHORIZING:
		case C.PA_CONTEXT_FAILED:
			return nil, fmt.Errorf("Connection error")
		case C.PA_CONTEXT_READY:
			break loop
		}
		if C.pa_mainloop_iterate(mainloop, 1, nil) < 0 {
			return nil, fmt.Errorf("Mainloop error")
		}
	}
	C.free(unsafe.Pointer(cname))
	client := Client{
		mainloop,
		mainloopAPI,
		context,
		make(chan func(), 10),
	}
	go client.loop()
	return &client, nil
}

func (c *Client) loop() {
	for {
		select {
		case call := <-c.updates:
			call()
		default:
			if C.pa_mainloop_iterate(c.mainloop, 1, nil) < 0 {
				fmt.Println("Exiting PulseAudio loop")
				return
			}
		}
	}
}

func (c *Client) post(f func()) {
	c.updates <- f
	C.pa_mainloop_wakeup(c.mainloop)
}

func (c *Client) run(op *C.pa_operation) {
	for C.pa_operation_get_state(op) == C.PA_OPERATION_RUNNING {
		C.pa_mainloop_iterate(c.mainloop, 1, nil)
	}
	if C.pa_operation_get_state(op) != C.PA_OPERATION_DONE {
		fmt.Println("Error when running PulseAudio operation!")
	}
	C.pa_operation_unref(op)
}

var callback func()

//export go_context_subscribe_cb
func go_context_subscribe_cb(event C.pa_subscription_event_type_t, idx C.uint, userdata unsafe.Pointer) {
	callback()
}

// Subscribe registers specified callback to be called on every PulseAudio update.
func (c *Client) Subscribe(cb func()) {
	c.post(func() {
		callback = cb
		C.pa_context_set_subscribe_callback(c.context, (*[0]byte)(C.context_subscribe_cb), nil)
		c.run(C.pa_context_subscribe(c.context, C.PA_SUBSCRIPTION_MASK_ALL, nil, nil))
	})
}

// ProfileInfo provides operations on PulseAudio profiles.
type ProfileInfo struct {
	Client      *Client
	Card        *Card
	Name        string
	Description string
	SinkCount   uint
	SourceCount uint
	Priority    uint
	Available   bool
}

// Activate sets this profile as the main one for the given card.
func (p *ProfileInfo) Activate() {
	p.Client.post(func() {
		cname := C.CString(p.Card.Name)
		pname := C.CString(p.Name)
		p.Client.run(C.pa_context_set_card_profile_by_name(p.Client.context, cname, pname, nil, nil))
		C.free(unsafe.Pointer(pname))
		C.free(unsafe.Pointer(cname))
	})
}

func newProfileInfo(client *Client, card *Card, ptr *C.pa_card_profile_info2) *ProfileInfo {
	return &ProfileInfo{
		Client:      client,
		Card:        card,
		Name:        C.GoString(ptr.name),
		Description: C.GoString(ptr.description),
		SinkCount:   uint(ptr.n_sinks),
		SourceCount: uint(ptr.n_sources),
		Priority:    uint(ptr.priority),
		Available:   ptr.available != 0,
	}
}

// PortAvailable tells whether a port on the sound card is available.
type PortAvailable int

const (
	// PortAvailableUnknown indicates that a port cannot be queried for availability.
	PortAvailableUnknown = iota
	// PortAvailableNo indicates that a port is disconnected.
	PortAvailableNo
	// PortAvailableYes indicates that a port is connected.
	PortAvailableYes
)

func convertPortAvailable(paAvailable C.int) PortAvailable {
	switch paAvailable {
	case C.PA_PORT_AVAILABLE_UNKNOWN:
		return PortAvailableUnknown
	case C.PA_PORT_AVAILABLE_NO:
		return PortAvailableNo
	case C.PA_PORT_AVAILABLE_YES:
		return PortAvailableYes
	default:
		panic("Unknown availability: " + string(paAvailable))
	}
}

// Direction tells whether a port on the sound card is a source or a sink.
type Direction int

const (
	// DirectionOutput indicates that a port is an audio sink.
	DirectionOutput = iota
	// DirectionInput indicates that a port is an audio source.
	DirectionInput
)

func convertDirection(paDirection C.int) Direction {
	switch paDirection {
	case C.PA_DIRECTION_OUTPUT:
		return DirectionOutput
	case C.PA_DIRECTION_INPUT:
		return DirectionInput
	default:
		panic("Unknown direction: " + string(paDirection))
	}
}

func convertPropertyList(props *C.pa_proplist) map[string][]byte {
	m := make(map[string][]byte)
	var ptr unsafe.Pointer
	for {
		cName := C.pa_proplist_iterate(props, &ptr)
		if cName == nil {
			break
		}
		var data unsafe.Pointer
		var size C.size_t
		C.pa_proplist_get(props, cName, &data, &size)
		name := C.GoString(cName)
		bytes := C.GoBytes(data, C.int(size))
		m[name] = bytes
	}
	return m
}

// PortInfo provides information on audio ports.
type PortInfo struct {
	Name          string
	Description   string
	Priority      uint
	Available     PortAvailable
	Direction     Direction
	Properties    map[string][]byte
	LatencyOffset int64
	Profiles      []*ProfileInfo
}

func newPortInfo(client *Client, card *Card, i *C.pa_card_port_info) *PortInfo {
	port := PortInfo{
		Name:          C.GoString(i.name),
		Description:   C.GoString(i.description),
		Priority:      uint(i.priority),
		Available:     convertPortAvailable(i.available),
		Direction:     convertDirection(i.direction),
		LatencyOffset: int64(i.latency_offset),
	}
	for iter := i.profiles2; *iter != nil; iter = (**C.pa_card_profile_info2)(unsafe.Pointer(uintptr(unsafe.Pointer(iter)) + unsafe.Sizeof(iter))) {
		ptr := *iter
		profile := newProfileInfo(client, card, ptr)
		port.Profiles = append(port.Profiles, profile)
	}
	port.Properties = convertPropertyList(i.proplist)
	return &port
}

// Card provides information on sound cards.
type Card struct {
	Client        *Client
	Index         uint
	Name          string
	OwnerModule   uint
	Driver        string
	Profiles      []*ProfileInfo
	ActiveProfile *ProfileInfo
	Properties    map[string][]byte
	Ports         []*PortInfo
}

func newCardInfo(c *Client, i *C.pa_card_info) *Card {
	card := Card{
		Client:      c,
		Index:       uint(i.index),
		Name:        C.GoString(i.name),
		OwnerModule: uint(i.owner_module),
		Driver:      C.GoString(i.driver),
	}
	for iter := i.profiles2; *iter != nil; iter = (**C.pa_card_profile_info2)(unsafe.Pointer(uintptr(unsafe.Pointer(iter)) + unsafe.Sizeof(iter))) {
		ptr := *iter
		profile := newProfileInfo(c, &card, ptr)
		card.Profiles = append(card.Profiles, profile)
		if i.active_profile2 == ptr {
			card.ActiveProfile = profile
		}
	}
	for iter := i.ports; *iter != nil; iter = (**C.pa_card_port_info)(unsafe.Pointer(uintptr(unsafe.Pointer(iter)) + unsafe.Sizeof(iter))) {
		ptr := *iter
		port := newPortInfo(c, &card, ptr)
		card.Ports = append(card.Ports, port)
	}
	card.Properties = convertPropertyList(i.proplist)
	return &card
}

type cardList struct {
	client *Client
	cards  []*Card
}

// Cards queries PulseAudio for all available sound cards.
func (c *Client) Cards() []*Card {
	ret := make(chan []*Card)
	c.post(func() {
		var list cardList
		op := C.pa_context_get_card_info_list(c.context, (*[0]byte)(C.card_info_cb), unsafe.Pointer(&list))
		list.client = c
		c.run(op)
		ret <- list.cards
	})
	return <-ret
}

//export go_card_info_cb
func go_card_info_cb(i *C.pa_card_info, eol C.int, userdata unsafe.Pointer) {
	if eol != 0 {
		return
	}
	list := (*cardList)(userdata)
	card := newCardInfo(list.client, i)
	list.cards = append(list.cards, card)
}

// ChannelVolume contains volume values (0-65536) for audio channels.
type ChannelVolume []uint

// Sink provides operations on PulseAudio sinks.
type Sink struct {
	Client      *Client
	Name        string
	Index       uint
	Description string
	// TODO: sample_spec
	// TODO: channel_map
	// TODO: owner_module
	Volume ChannelVolume
	Mute   bool
	// TODO: monitor_source
	// TODO: monitor_source_name
	// TODO: latency
	// TODO: driver
	// TODO: flags
	Properties map[string][]byte
	// TODO: configured_latency
	// TODO: base_volume
	// TODO: state
	// TODO: n_volume_steps
	// TODO: card
	// TODO: ports
	// TODO: formats
}

// GetVolume returns the volume for this audio sink.
func (s *Sink) GetVolume() float32 {
	return float32(s.Volume[0]) / float32(C.PA_VOLUME_NORM)
}

// SetVolume sets the volume for this audio sink.
func (s *Sink) SetVolume(value float32) {
	var volume C.struct_pa_cvolume
	volume.channels = C.uchar(len(s.Volume))
	for i := range s.Volume {
		volume.values[i] = C.uint(value * C.PA_VOLUME_NORM)
		s.Volume[i] = uint(value * float32(C.PA_VOLUME_NORM))
	}
	s.Client.post(func() {
		s.Client.run(C.pa_context_set_sink_volume_by_index(s.Client.context, C.uint(s.Index), &volume, nil, nil))
	})
}

func newSinkInfo(c *Client, i *C.pa_sink_info) *Sink {
	sink := Sink{
		Client:      c,
		Name:        C.GoString(i.name),
		Index:       uint(i.index),
		Description: C.GoString(i.description),
		Mute:        i.mute != 0,
		Properties:  convertPropertyList(i.proplist),
	}
	for iter := 0; iter < int(i.volume.channels); iter++ {
		sink.Volume = append(sink.Volume, uint(i.volume.values[iter]))
	}
	return &sink
}

type sinkList struct {
	c     *Client
	sinks []*Sink
}

// Sinks queries PulseAudio for all audio sinks.
func (c *Client) Sinks() []*Sink {
	ret := make(chan []*Sink)
	c.post(func() {
		list := sinkList{}
		op := C.pa_context_get_sink_info_list(c.context, (*[0]byte)(C.sink_info_cb), unsafe.Pointer(&list))
		list.c = c
		c.run(op)
		ret <- list.sinks
	})
	return <-ret
}

//export go_sink_info_cb
func go_sink_info_cb(i *C.pa_sink_info, eol C.int, userdata unsafe.Pointer) {
	if eol != 0 {
		return
	}
	list := (*sinkList)(userdata)
	sink := newSinkInfo((*Client)(list.c), i)
	list.sinks = append(list.sinks, sink)
}

// Close disconnects Client from the PulseAudio server.
func (c *Client) Close() {
	c.post(func() {
		if C.pa_context_get_state(c.context) == C.PA_CONTEXT_READY {
			C.pa_context_disconnect(c.context)
		}
		C.pa_mainloop_free(c.mainloop)
	})
}
