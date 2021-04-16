package v1

func (p PropertyKey) DeepClone() *PropertyKey {
	return &p
}

func (c Condition) DeepClone() *Condition {
	return &c
}

func (f Filter) DeepClone() *Filter {
	if f.Key != nil {
		f.Key = f.Key.DeepClone()
	}

	if len(f.Conditions) > 0 {
		conditions := f.Conditions
		f.Conditions = make([]*Condition, len(conditions))
		for i, c := range f.Conditions {
			f.Conditions[i] = c.DeepClone()
		}
	}
	return &f
}

func (o OracleSpecConfiguration) DeepClone() *OracleSpecConfiguration {
	if len(o.Filters) > 0 {
		filters := o.Filters
		o.Filters = make([]*Filter, len(filters))
		for i, f := range filters {
			o.Filters[i] = f.DeepClone()
		}
	}
	return &o
}
