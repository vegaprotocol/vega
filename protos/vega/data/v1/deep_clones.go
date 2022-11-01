package v1

func (p PropertyKey) DeepClone() *PropertyKey {
	pkey := p
	return &pkey
}

func (s Signer) DeepClone() *Signer {
	return &Signer{
		Signer: s.Signer,
	}
}

func (c Condition) DeepClone() *Condition {
	cond := c
	return &cond
}

func (f Filter) DeepClone() *Filter {
	if f.Key != nil {
		f.Key = f.Key.DeepClone()
	}

	if len(f.Conditions) > 0 {
		conditions := f.Conditions
		f.Conditions = make([]*Condition, len(conditions))
		for i, c := range conditions {
			f.Conditions[i] = c.DeepClone()
		}
	}
	return &f
}

func (s DataSourceSpecConfiguration) DeepClone() *DataSourceSpecConfiguration {
	if len(s.Signers) > 0 {
		sgns := s.Signers
		s.Signers = make([]*Signer, len(sgns))
		for i, sig := range sgns {
			s.Signers[i] = sig.DeepClone()
		}
	}

	if len(s.Filters) > 0 {
		filters := s.Filters
		s.Filters = make([]*Filter, len(filters))
		for i, f := range filters {
			s.Filters[i] = f.DeepClone()
		}
	}

	return &DataSourceSpecConfiguration{
		Signers: s.Signers,
		Filters: s.Filters,
	}
}

func (s DataSourceSpec) DeepClone() *DataSourceSpec {
	config := &DataSourceSpecConfiguration{}
	if s.Config != nil {
		config = s.Config.DeepClone()
	}
	return &DataSourceSpec{
		Id:        s.Id,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Config:    config,
		Status:    s.Status,
	}
}

func (s ExternalDataSourceSpec) DeepClone() *ExternalDataSourceSpec {
	if s.Spec != nil {
		spec := s.Spec.DeepClone()
		return &ExternalDataSourceSpec{
			Spec: spec,
		}
	}

	return &ExternalDataSourceSpec{
		Spec: &DataSourceSpec{},
	}
}

func (o OracleSpec) DeepClone() *OracleSpec {
	if o.ExternalDataSourceSpec != nil {
		return &OracleSpec{
			ExternalDataSourceSpec: o.ExternalDataSourceSpec.DeepClone(),
		}
	}

	return &OracleSpec{
		ExternalDataSourceSpec: &ExternalDataSourceSpec{
			Spec: &DataSourceSpec{},
		},
	}
}

func (p Property) DeepClone() *Property {
	return &p
}

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
