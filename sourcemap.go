package sourcemap

import (
	"encoding/json"
	"io"
	"path"
	"strings"
)

type Map struct {
	Version         int      `json:"version"`
	File            string   `json:"file"`
	SourceRoot      string   `json:"sourceRoot"`
	Sources         []string `json:"sources"`
	Names           []string `json:"names"`
	Mappings        string   `json:"mappings"`
	decodedMappings []*Mapping
}

type Mapping struct {
	OriginalFile    string
	OriginalLine    int
	OriginalColumn  int
	GeneratedLine   int
	GeneratedColumn int
}

func Read(r io.Reader) (*Map, error) {
	d := json.NewDecoder(r)
	var m Map
	if err := d.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

const base64encode = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var base64decode [256]int

func init() {
	for i := 0; i < len(base64decode); i++ {
		base64decode[i] = 0xff
	}
	for i := 0; i < len(base64encode); i++ {
		base64decode[base64encode[i]] = i
	}
}

func (m *Map) DecodedMappings() []*Mapping {
	if m.decodedMappings == nil {
		r := strings.NewReader(m.Mappings)
		readVLQ := func() int {
			v := 0
			s := uint(0)
			for {
				b, _ := r.ReadByte()
				o := base64decode[b]
				if o == 0xff {
					r.UnreadByte()
					return 0
				}
				v += (o &^ 32) << s
				if o&32 == 0 {
					break
				}
				s += 5
			}
			if v&1 != 0 {
				return -(v >> 1)
			}
			return v >> 1
		}
		var originalFile = 0
		var originalLine = 1
		var originalColumn = 0
		var generatedLine = 1
		var generatedColumn = 0
		for r.Len() != 0 {
			b, _ := r.ReadByte()
			if b == ',' {
				continue
			}
			if b == ';' {
				generatedLine++
				generatedColumn = 0
				continue
			}
			r.UnreadByte()
			generatedColumn += readVLQ()
			originalFile += readVLQ()
			originalLine += readVLQ()
			originalColumn += readVLQ()
			readVLQ()
			m.decodedMappings = append(m.decodedMappings, &Mapping{path.Join(m.SourceRoot, m.Sources[originalFile]), originalLine, originalColumn, generatedLine, generatedColumn})
		}
	}
	return m.decodedMappings
}