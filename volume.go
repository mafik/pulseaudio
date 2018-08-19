package pulseaudio

import (
	"fmt"
)

const pulseVolumeMax = 0xffff

// Volume returns current audio volume as a number from 0 to 1 (or more than 1 - if volume is boosted).
func (c *Client) Volume() (float32, error) {
	s, err := c.ServerInfo()
	if err != nil {
		return 0, err
	}
	sinks, err := c.sinks()
	for _, sink := range sinks {
		if sink.Name != s.DefaultSink {
			continue
		}
		return float32(sink.Cvolume[0]) / pulseVolumeMax, nil
	}
	return 0, fmt.Errorf("PulseAudio error: couldn't query volume - sink %s not found", s.DefaultSink)
}

// SetVolume changes the current volume to a specified value from 0 to 1 (or more than 1 - if volume should be boosted).
func (c *Client) SetVolume(volume float32) error {
	s, err := c.ServerInfo()
	if err != nil {
		return err
	}
	return c.setSinkVolume(s.DefaultSink, cvolume{uint32(volume * 0xffff)})
}

func (c *Client) SetSinkVolume(sinkName string, volume float32) error {
	return c.setSinkVolume(sinkName, cvolume{uint32(volume * 0xffff)})
}

func (c *Client) setSinkVolume(sinkName string, cvolume cvolume) error {
	_, err := c.request(commandSetSinkVolume, uint32Tag, uint32(0xffffffff), stringTag, []byte(sinkName), byte(0), cvolume)
	return err
}
