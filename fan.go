package main

import (
	"fmt"
	"log/syslog"
	"time"
)

type (
	temp uint8
	pwm  uint8
)

func (t temp) String() string {
	return fmt.Sprint(uint8(t))
}
func (p pwm) String() string {
	return fmt.Sprint(uint8(p))
}

type sensor struct {
	name       string
	sensorFile readableSysFile
	updateFreq time.Duration
	watchers   []*fan
}

type fan struct {
	name    string
	pwmFile writableSysFile
	fanFile readableSysFile
	speeds  fanSpeeds

	watching []struct {
		s   *sensor
		r   fanRelation
		pwm pwm
	}
}

type fanSpeeds struct {
	Idle   pwm
	Active pwm
	Full   pwm
}

func convertSensor(name string, config jsonSensor) (s sensor, err error) {
	s.name = name
	f, err := readableSysPath(config.Location).Open()
	if err != nil {
		return
	}
	s.sensorFile = f
	s.updateFreq = time.Second * time.Duration(config.UpdateFreq)
	return
}

func convertFan(name string, config jsonFan) (f fan, err error) {
	f.name = name
	// Enable pwm on the fan
	file, err := writableSysPath(config.PwmFile + "_enable").Open()
	if err != nil {
		return
	}
	err = file.WriteString("1")
	if err != nil {
		return
	}
	file.Close()

	file, err = writableSysPath(config.PwmFile).Open()
	if err != nil {
		return
	}
	f.pwmFile = file

	rFile, err := readableSysPath(config.FanFile + "_input").Open()
	if err != nil {
		return
	}
	f.fanFile = rFile
	f.speeds = config.Speeds

	return
}

func (s *sensor) Temp() temp {
	val, err := s.sensorFile.ReadVal()
	if err != nil {
		panic(err)
	}
	return temp(val)
}

func (f fanRelation) Relate(t temp) (pwm, bool) {
	if t < f.Target {
		return f.Idle, false
	}
	if t > f.Max {
		return f.Full, true
	}

	scaled := float32(t)
	scaled -= float32(f.Target)
	scaled *= float32(f.Full-f.Active)
	scaled /= float32(f.Max-f.Target)
	scaled += float32(f.Active)

	return pwm(scaled), false
}

func (s *sensor) Update(l *syslog.Writer) {
	val := s.Temp()
	for _, f := range s.watchers {
		f.Update(s, val, l)
	}
}

func (s *sensor) Watch(l *syslog.Writer) {
	for {
		s.Update(l)
		time.Sleep(s.updateFreq)
	}
}

func (f *fan) Update(s *sensor, val temp, l *syslog.Writer) {
	found := false
	for i := range f.watching {
		watch := &f.watching[i]
		if watch.s.name == s.name {
			var critical bool
			watch.pwm, critical = watch.r.Relate(val)
			if critical {
				l.Crit("Sensor '" + s.name + "' above maximum temperature of " + watch.r.Max.String() + ". Running fan '" + f.name + "' at full speed.")
			}
			found = true
			break
		}
	}
	if !found {
		panic("couldn't find sensor " + s.name + " for fan " + f.name)
	}
	pwm := f.Pwm()
	f.SetPwm(pwm)
}

func (f *fan) SetPwm(val pwm) {
	f.pwmFile.WriteString(val.String())
}

func (f *fan) Pwm() pwm {
	max := f.speeds.Idle
	for _, s := range f.watching {
		if s.pwm > max {
			max = s.pwm
		}
	}
	return max
}

