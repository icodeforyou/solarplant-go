package ferroamp

import "maps"

type FaData struct {
	Ehub EhubMessage
	Sso  map[string]SsoMessage
	Eso  map[string]EsoMessage
	Esm  map[string]EsmMessage
}

func NewFaData() *FaData {
	return &FaData{
		Sso: make(map[string]SsoMessage),
		Eso: make(map[string]EsoMessage),
		Esm: make(map[string]EsmMessage),
	}
}

func (data *FaData) Clone() *FaData {
	return &FaData{
		Ehub: data.Ehub,
		Sso:  maps.Clone(data.Sso),
		Eso:  maps.Clone(data.Eso),
		Esm:  maps.Clone(data.Esm),
	}
}
