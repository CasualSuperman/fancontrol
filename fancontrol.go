package main

import (
	"encoding/json"
	"os"
	"time"
)

type jsonConfigFile struct {
	Sensors map[string]jsonSensor
	Fans    map[string]jsonFan
	Temps   map[string]jsonTemps
}

type jsonSensor struct {
	Location   tempLocation
	UpdateFreq int `json:"update"`
}

type jsonFan struct {
	PwmFile pwmLocation `json:"pwm"`
	FanFile fanLocation `json:"fan"`
	Speeds  fanSpeeds
}

type fanRelation struct {
	fanSpeeds
	Target uint8
	Max    uint8
}

type jsonTemps map[string]fanRelation

func main() {
	// Get to the command station
	err := os.Chdir("/sys/class/hwmon")
	if err != nil {
		panic(err)
	}

	f, err := os.Open("/etc/conf.d/fans")

	if err != nil {
		panic(err)
	}

	var config *jsonConfigFile

	// Load configuration
	dec := json.NewDecoder(f)
	err = dec.Decode(&config)

	if err != nil {
		panic(err)
	}

	var sensors []sensor
	var fans []fan

	for name, s := range config.Sensors {
		sensor, err := convertSensor(name, s)
		if err != nil {
			panic(err)
		}
		sensors = append(sensors, sensor)
	}

	for name, f := range config.Fans {
		fan, err := convertFan(name, f)
		if err != nil {
			panic(err)
		}
		fans = append(fans, fan)
	}

	for fanName, relations := range config.Temps {
		var fan *fan
		for f := range fans {
			if fans[f].name == fanName {
				fan = &fans[f]
				break
			}
		}

		if fan == nil {
			panic("unable to find fan named " + fanName)
		}

		for sensorName, relation := range relations {
			var sen *sensor
			for s := range sensors {
				if sensors[s].name == sensorName {
					sen = &sensors[s]
					break
				}
			}

			if sen == nil {
				panic("unable to find sensor named " + sensorName)
			}

			var watch struct {
				s     *sensor
				r     fanRelation
				speed uint8
			}
			watch.s = sen
			watch.r = relation
			watch.r.fanSpeeds = fan.speeds

			fan.watching = append(fan.watching, watch)
			sen.watchers = append(sen.watchers, fan)
		}
	}

	config = nil

	for i := range sensors {
		go sensors[i].Watch()
	}

	for {
		time.Sleep(1 * time.Hour)
	}
}
