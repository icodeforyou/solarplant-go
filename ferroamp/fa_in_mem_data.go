package ferroamp

import (
	"sync"

	"github.com/angas/solarplant-go/convert"
)

type FaInMemData struct {
	mu   sync.RWMutex
	data *FaData
}

func NewFaInMemData() *FaInMemData {
	return &FaInMemData{data: NewFaData()}
}

func (d *FaInMemData) CurrentState() *FaData {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.data.Clone()
}

func (d *FaInMemData) SetEHub(ehub *EhubMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.Ehub = *ehub
}

func (d *FaInMemData) SetSso(sso *SsoMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.Sso[sso.ID.Value] = *sso
}

func (d *FaInMemData) SetEso(eso *EsoMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.Eso[eso.ID.Value] = *eso
}

func (d *FaInMemData) SetEsm(esm *EsmMessage) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data.Esm[esm.ID.Value] = *esm
}

/** Battery level in percent (stage of chanrge) */
func (d *FaInMemData) BatteryLevel() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.data.Esm) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, esm := range d.data.Esm {
		sum += esm.Soc.Value
	}

	return convert.TwoDecimals(sum / float64(len(d.data.Esm)))
}

func (d *FaInMemData) BatteryStatuses() []int16 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.data.Esm) == 0 {
		return []int16{}
	}

	var statuses []int16
	for _, esm := range d.data.Esm {
		statuses = append(statuses, int16(esm.Status.Value))
	}

	return statuses
}

/** Production lifetime in kWh */
func (d *FaInMemData) ProductionLifetime() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.data.Sso) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, sso := range d.data.Sso {
		sum += float64(sso.Wpv.Value)
	}

	return convert.TwoDecimals(convert.MJ2Kwh(sum))
}

/** Solar power in kW */
func (d *FaInMemData) SolarPower() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.data.Sso) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, sso := range d.data.Sso {
		sum += sso.Upv.Value * sso.Ipv.Value
	}

	return convert.TwoDecimals(sum / 1e3)
}

/** Grid power in kW */
func (d *FaInMemData) GridPower() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return convert.TwoDecimals((d.data.Ehub.Pext.L1 + d.data.Ehub.Pext.L2 + d.data.Ehub.Pext.L3) / 1e3)
}

/** Battery power in kW. Charging = negative value, discharging = positive value */
func (d *FaInMemData) BatteryPower() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return convert.TwoDecimals(d.data.Ehub.Pbat.Value / 1e3)
}

/** Production since given state in kWh */
func (d *FaInMemData) ProducedSince(since FaData) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	mj := d.data.Ehub.Wpv.Value - since.Ehub.Wpv.Value
	return convert.TwoDecimals(convert.MJ2Kwh(mj))
	// This should work but values (wloadprodq) are not updateing as expected
	// return convert.TwoDecimals(d.data.Ehub.LifetimeProduced() - from.Ehub.LifetimeProduced())
}

/** Consumption since given state in kWh */
func (d *FaInMemData) ConsumedSince(since FaData) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return convert.TwoDecimals(d.data.Ehub.LifetimeConsumed() - since.Ehub.LifetimeConsumed())
}

/** Battery net load since given state in kWh */
func (d *FaInMemData) BatteryNetLoadSince(since FaData) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	prod := d.data.Ehub.WbatProd.Value - since.Ehub.WbatProd.Value
	cons := d.data.Ehub.WbatCons.Value - since.Ehub.WbatCons.Value
	return convert.TwoDecimals(convert.MJ2Kwh(prod - cons))
}
