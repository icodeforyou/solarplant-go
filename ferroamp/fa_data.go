package ferroamp

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
	clone := &FaData{Ehub: data.Ehub}

	clone.Sso = make(map[string]SsoMessage, len(data.Sso))
	for k, v := range data.Sso {
		clone.Sso[k] = v
	}

	clone.Eso = make(map[string]EsoMessage, len(data.Eso))
	for k, v := range data.Eso {
		clone.Eso[k] = v
	}

	clone.Esm = make(map[string]EsmMessage, len(data.Esm))
	for k, v := range data.Esm {
		clone.Esm[k] = v
	}

	return clone
}
