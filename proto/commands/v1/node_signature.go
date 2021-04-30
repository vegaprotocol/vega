package v1

func (n NodeSignature) DeepClone() *NodeSignature {
	if len(n.Sig) > 0 {
		sigBytes := n.Sig
		n.Sig = make([]byte, len(sigBytes))
		for i, b := range sigBytes {
			n.Sig[i] = b
		}
	}
	return &n
}

// required for graphq event stream
func (NodeSignature) IsEvent() {}
