package main

import "time"

type (
	temp uint8
	pwm  uint8
)

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

func (s *sensor) Temp() temp {
	val, err := s.sensorFile.ReadVal()
	if err != nil {
		panic(err)
	}
	println(s.name + " temp:", val)
	return temp(val)
}

func (f fanRelation) Relate(t temp) pwm {
	if t < f.Target {
		return f.Idle
	}
	if t > f.Max {
		return f.Full
	}

	scaled := float32(t)
	scaled -= float32(f.Target)
	scaled *= float32(f.Full-f.Active)
	scaled /= float32(f.Max-f.Target)
	scaled += float32(f.Active)

	return pwm(scaled)
}

func (s *sensor) Update() {
	val := s.Temp()
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

func (f *fan) Update(s *sensor, val temp) {
	found := false
	for i := range f.watching {
		watch := &f.watching[i]
		if watch.s.name == s.name {
			watch.pwm = watch.r.Relate(val)
			found = true
			break
		}
	}
	if !found {
		panic("couldn't find sensor " + s.name + " for fan " + f.name)
	}
	pwm := f.Pwm()
	println(f.name + " pwm:", pwm)
	f.SetPwm(pwm)
}

func (f *fan) SetPwm(val pwm) {
	f.pwmFile.WriteVal(uint8(val))
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

