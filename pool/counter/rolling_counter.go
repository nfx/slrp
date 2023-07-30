package counter

import (
	"bytes"
	"encoding/binary"
	"time"
)

type RollingCounter struct {
	buf      []int32
	updated  time.Time
	window   int16
	interval time.Duration
	sum      int32
	pos      int16
}

func NewRollingCounter(window int16, interval time.Duration) RollingCounter {
	now := time.Now()
	return RollingCounter{
		buf:      make([]int32, window),
		updated:  now.Round(interval),
		window:   window,
		interval: interval,
	}
}

func (r *RollingCounter) Series() []int32 {
	seq := append(r.buf[r.pos:], r.buf[:r.pos]...)
	return seq
}

func (r *RollingCounter) Add(what int32) {
	now := time.Now().Truncate(r.interval)
	clearIntervals := int16(now.Sub(r.updated) / r.interval)
	if clearIntervals > r.window {
		clearIntervals = r.window
	}
	for i := 0; i < int(clearIntervals); i++ {
		r.pos = (r.pos + 1) % r.window
		r.sum -= r.buf[r.pos]
		r.buf[r.pos] = 0
	}
	r.updated = now
	r.buf[r.pos] += what
	r.sum += what
}

func (r *RollingCounter) Increment() {
	r.Add(1)
}

func (r *RollingCounter) Sum() int {
	return int(r.sum)
}

// MarshalBinary converts the rollingCounter struct into a binary representation.
func (r *RollingCounter) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write each field of the struct into the buffer in binary format
	err := binary.Write(buf, binary.BigEndian, int64(len(r.buf)))
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, r.buf)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, r.updated.UnixNano())
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, int16(r.window))
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, r.interval.Nanoseconds())
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, r.sum)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, int16(r.pos))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary converts the binary data back into the original rollingCounter struct.
func (r *RollingCounter) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	// Read each field from the buffer and decode it into the struct
	var bufLen int64
	err := binary.Read(buf, binary.BigEndian, &bufLen)
	if err != nil {
		return err
	}
	r.buf = make([]int32, bufLen)
	err = binary.Read(buf, binary.BigEndian, &r.buf)
	if err != nil {
		return err
	}
	var updatedNano int64
	err = binary.Read(buf, binary.BigEndian, &updatedNano)
	if err != nil {
		return err
	}
	r.updated = time.Unix(0, updatedNano)
	err = binary.Read(buf, binary.BigEndian, &r.window)
	if err != nil {
		return err
	}
	var intervalNano int64
	err = binary.Read(buf, binary.BigEndian, &intervalNano)
	if err != nil {
		return err
	}
	r.interval = time.Duration(intervalNano)
	err = binary.Read(buf, binary.BigEndian, &r.sum)
	if err != nil {
		return err
	}
	err = binary.Read(buf, binary.BigEndian, &r.pos)
	if err != nil {
		return err
	}
	// reset previous intervals in the buffer after a downtime
	r.Add(0)
	return nil
}
