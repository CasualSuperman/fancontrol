package main

import "time"

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
		s     *sensor
		r     fanRelation
		speed uint8
	}
}

type fanSpeeds struct {
	Idle   uint8
	Active uint8
	Full   uint8
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
	err = file.WriteVal(1)
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

func (s *sensor) GetValue() uint8 {
	val, err := s.sensorFile.ReadVal()
	if err != nil {
		panic(err)
	}
	println(s.name + " temp:", val)
	return val
}

func (f fanRelation) Relate(temp uint8) uint8 {
	if temp < f.Target {
		return f.Idle
	}
	if temp > f.Max {
		return f.Max
	}

	scaled := float32(temp)
	scaled -= float32(f.Target)
	scaled *= float32(f.Full-f.Active)
	scaled /= float32(f.Max-f.Target)
	scaled += float32(f.Active)

	return uint8(scaled)
}

func (s *sensor) Update() {
	val := s.GetValue()
	for _, f := range s.watchers {
		f.Update(s, val)
	}
}

func (s *sensor) Watch() {
	for {
		s.Update()
		time.Sleep(s.updateFreq)
	}
}

func (f *fan) Update(s *sensor, val uint8) {
	found := false
	for i := range f.watching {
		watch := &f.watching[i]
		if watch.s.name == s.name {
			watch.speed = watch.r.Relate(val)
			found = true
			break
		}
	}
	if !found {
		panic("couldn't find sensor " + s.name + " for fan " + f.name)
	}
	speed := f.Speed()
	println(f.name + " pwm:", speed)
	f.SetSpeed(speed)
}

func (f *fan) SetSpeed(val uint8) {
	f.pwmFile.WriteVal(val)
}

func (f *fan) Speed() uint8 {
	max := f.speeds.Idle
	for _, s := range f.watching {
		if s.speed > max {
			max = s.speed
		}
	}
	return max
}

