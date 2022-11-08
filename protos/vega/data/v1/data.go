package v1

func (d Data) DeepClone() *Data {
	if d.Signers != nil && len(d.Signers) > 0 {
		sgns := d.Signers
		d.Signers = make([]*Signer, len(sgns))
		for i, s := range sgns {
			d.Signers[i] = s.DeepClone()
		}
	}

	if d.Data != nil && len(d.Data) > 0 {
		data := d.Data
		d.Data = make([]*Property, len(data))
		for i, dt := range data {
			d.Data[i] = dt.DeepClone()
		}
	}

	if d.MatchedSpecIds != nil && len(d.MatchedSpecIds) > 0 {
		ms := d.MatchedSpecIds
		d.MatchedSpecIds = make([]string, len(ms))
		for i, m := range ms {
			d.MatchedSpecIds[i] = m
		}
	}

	return &d
}

func (o ExternalData) DeepClone() ExternalData {
	if o.Data != nil {
		return ExternalData{
			Data: o.Data.DeepClone(),
		}
	}

	return ExternalData{
		Data: &Data{},
	}
}
