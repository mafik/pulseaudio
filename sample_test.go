package pulseaudio

import (
	"fmt"
	"testing"
	"time"
)

// clientForTest creates a new client and ensures that there is an active output.
func clientForTest() *Client {
	c, err := NewClient()
	if err != nil {
		panic(err)
	}
	outs, active, err := c.Outputs()
	if err != nil {
		panic(err)
	}
	if active < 0 {
		for _, out := range outs {
			if !out.Available {
				continue
			}
			err = out.Activate()
			if err != nil {
				panic(err)
			}
			break
		}
	}
	err = c.SetVolume(0.5)
	if err != nil {
		panic(err)
	}
	return c
}

func Example() {
	client, err := NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()
	// Use `client` to interact with PulseAudio
}

func TestExampleClient_SetVolume(t *testing.T) {
	c := clientForTest()
	defer c.Close()

	err := c.SetVolume(1.5)
	if err != nil {
		panic(err)
	}

	vol, err := c.Volume()
	if err != nil{
		t.Errorf("%v", err)
	}
	if vol < 1.4999 {
		t.Errorf("Wrong volume value : %v", vol)
	}
}

func TestExampleClient_Updates(t *testing.T) {
	c := clientForTest()
	defer c.Close()

	updates, err := c.Updates()
	if err != nil {
		panic(err)
	}

	select {
	case _ = <-updates:
		//fmt.Println("Got update from PulseAudio")
		t.Errorf("Got update from PulseAudio")
	case _ = <-time.After(time.Millisecond * 10):
		fmt.Println("No update in 10 ms")
	}

	err = c.SetVolume(0.1)
	if err != nil {
		panic(err)
	}
	fmt.Println("Volume set to 0.1")

	select {
	case _ = <-updates:
		fmt.Println("Got update from PulseAudio")
	case _ = <-time.After(time.Millisecond * 10):
		t.Errorf("No update in 10 ms")
	}

	// Output:
	// No update in 10 ms
	// Volume set to 0.1
	// Got update from PulseAudio
}
