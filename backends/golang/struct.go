package golang

import (
	"fmt"

	"github.com/andyleap/gencode/schema"
)

func WalkStruct(s *schema.Struct) (parts *StringBuilder, err error) {
	parts = &StringBuilder{}
	intHandler := &schema.IntType{
		Bits:   64,
		Signed: false,
		VarInt: true,
	}
	parts.Append(fmt.Sprintf(`type %s struct {
	`, s.Name))
	for _, f := range s.Fields {
		p, err := WalkFieldDef(f)
		if err != nil {
			return nil, err
		}
		parts.Join(p)
		parts.Append(`
	`)
	}
	if !s.Framed {
		parts.Append(fmt.Sprintf(`}
	
func (d *%s) Size() (s uint64) {
	`, s.Name))
	} else {
		parts.Append(fmt.Sprintf(`}
	
func (d *%s) FramedSize() (s uint64, us uint64) {
	`, s.Name))
	}
	for _, f := range s.Fields {
		p, err := WalkFieldSize(f)
		if err != nil {
			return nil, err
		}
		parts.Join(p)
	}
	if s.Framed {
		intcode, err := WalkIntSize(intHandler, "l")
		if err != nil {
			return nil, err
		}
		parts.Append(`
	l := s
	us = s
	`)
		parts.Join(intcode)
	}
	parts.Append(fmt.Sprintf(`
	return 
}`))

	if s.Framed {
		parts.Append(fmt.Sprintf(`
func (d *%s) Size() (s uint64) {
	s, _ = d.FramedSize()
	return
}
`, s.Name))
	}

	parts.Append(fmt.Sprintf(`
func (d *%s) Marshal(buf []byte) ([]byte, error) {`, s.Name))
	if s.Framed {
		parts.Append(`
	size, usize := d.FramedSize()`)
	} else {
		parts.Append(`
	size := d.Size()`)
	}
	parts.Append(`
	{
		if uint64(cap(buf)) >= d.Size() {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)
	`)
	if s.Framed {
		intcode, err := WalkIntMarshal(intHandler, "usize")
		if err != nil {
			return nil, err
		}
		parts.Join(intcode)
	}
	for _, f := range s.Fields {
		p, err := WalkFieldMarshal(f)
		if err != nil {
			return nil, err
		}
		parts.Join(p)
	}
	parts.Append(fmt.Sprintf(`
	return buf[:i], nil
}
	
func (d *%s) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)
	`, s.Name))
	if s.Framed {
		parts.Append(`usize := uint64(0)
	`)
		intcode, err := WalkIntUnmarshal(intHandler, "usize")
		if err != nil {
			return nil, err
		}
		parts.Join(intcode)
		parts.Append(`
	if usize > uint64(len(buf))+i {
		return 0, io.EOF
	}`)
	}
	for _, f := range s.Fields {
		p, err := WalkFieldUnmarshal(f)
		if err != nil {
			return nil, err
		}
		parts.Join(p)
	}
	parts.Append(`
	return i, nil
}
`)
	if s.Framed {
		parts.Append(fmt.Sprintf(`
func (d *%s) Serialize(w io.Writer) (error) {
	buf, err := d.Marshal(nil)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	if err != nil {
		return err
	}
	return nil
}
`, s.Name))
		parts.Append(fmt.Sprintf(`
func (d *%s) Deserialize(r io.Reader) (error) {
	size := uint64(0)
	sbuf := []byte{0,0,0,0,0,0,0,0,0,0}
	bs := uint8(0)
	i := uint64(0)
	for sbuf[i] & 0x80 == 0x80 {
		_, err := r.Read(sbuf[i:i+1])
		if err != nil {
			return err
		}
		size |= uint64(sbuf[i]&0x7F) << bs
		bs += 7
		i++
	}
	buf := make([]byte, size + i)
	copy(buf, sbuf[0:i])
	n := uint64(i)
	size += i
	var err error
	for n < size && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += uint64(nn)
	}
	if err != nil {
		return err
	}
	_, err = d.Unmarshal(buf)
	if err != nil {
		return err
	}
	return nil
}
`, s.Name))
	}
	return
}

func WalkFieldDef(s *schema.Field) (parts *StringBuilder, err error) {
	parts = &StringBuilder{}
	parts.Append(fmt.Sprintf(`%s `, s.Name))
	subp, err := WalkTypeDef(s.Type)
	if err != nil {
		return nil, err
	}
	parts.Join(subp)
	return
}

func WalkFieldSize(s *schema.Field) (parts *StringBuilder, err error) {
	return WalkTypeSize(s.Type, "d."+s.Name)
}

func WalkFieldMarshal(s *schema.Field) (parts *StringBuilder, err error) {
	return WalkTypeMarshal(s.Type, "d."+s.Name)
}

func WalkFieldUnmarshal(s *schema.Field) (parts *StringBuilder, err error) {
	return WalkTypeUnmarshal(s.Type, "d."+s.Name)
}