package schema

import "github.com/storacha-network/go-ucanto/core/result/failure"

type mapped[I, O, O2 any] struct {
	reader    Reader[I, O]
	converter func(O) (O2, failure.Failure)
}

func (m mapped[I, O, O2]) Read(i I) (O2, failure.Failure) {
	o, err := m.reader.Read(i)
	if err != nil {
		var o2 O2
		return o2, err
	}
	return m.converter(o)
}

func Mapped[I, O, O2 any](reader Reader[I, O], converter func(O) (O2, failure.Failure)) Reader[I, O2] {
	return mapped[I, O, O2]{reader, converter}
}
