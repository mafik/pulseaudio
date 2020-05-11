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
	if err != nil {
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

func TestExampleClient_SetMute(t *testing.T) {
	c := clientForTest()
	defer c.Close()

	err := c.SetMute(true)
	if err != nil {
		t.Errorf("Can't mute : %v", err)
	}
	b, err := c.Mute()
	if err != nil || !b {
		t.Errorf("Can't mute : %v", err)
	}

	err = c.SetMute(false)
	if err != nil {
		t.Errorf("Can't unmute : %v", err)
	}
	b, err = c.Mute()
	if err != nil || b {
		t.Errorf("Wrong value : %v", err)
	}

}

func TestExampleClient_ToggleMute(t *testing.T) {
	c := clientForTest()
	defer c.Close()

	b1, err := c.ToggleMute()
	if err != nil {
		t.Errorf("Can't toggle mute : %v", err)
	}
	b2, err := c.ToggleMute()
	if err != nil {
		t.Errorf("Can't toggle mute : %v", err)
	}

	if b1 == b2 {
		t.Errorf("Wrong value : %v", err)
	}
}
