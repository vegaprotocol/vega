package verify

import (
	"bytes"
	"io/ioutil"
	"os"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/protobuf/jsonpb"
)

type AssetCmd struct{}

func (opts *AssetCmd) Execute(params []string) error {
	return verifier(params, verifyAsset)
}

func readFile(r *reporter, path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		r.Err("%v, no such file or directory", path)
		return nil
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		r.Err("unable to read file: %v", err)
		return nil
	}

	return bytes
}

func verifyAsset(r *reporter, bs []byte) string {
	prop := &types.Proposal{}
	u := jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}

	err := u.Unmarshal(bytes.NewBuffer(bs), prop)
	if err != nil {
		r.Err("unable to unmarshal file: %v", err)
		return ""
	}

	if len(prop.Reference) <= 0 {
		r.Warn("no proposal.reference specified")
	}

	if len(prop.PartyID) <= 0 {
		r.Err("proposal.partyID is missing")
	} else {
		if !isValidParty(prop.PartyID) {
			r.Warn("proposal.partyID does not seems to be a valid party ID")
		}
	}

	if prop.Terms == nil {
		r.Err("missing proposal.Terms")
	} else {
		verifyAssetTerms(r, prop)
	}

	m := jsonpb.Marshaler{
		Indent:       " ",
		EmitDefaults: true,
	}
	buf, _ := m.MarshalToString(prop)
	return string(buf)
}

func verifyAssetTerms(r *reporter, prop *types.Proposal) {
	if prop.Terms.ClosingTimestamp == 0 {
		r.Err("prop.terms.closingTimestamp is missing or 0")
	} else if time.Unix(prop.Terms.ClosingTimestamp, 0).Before(time.Now()) {
		r.Warn("prop.terms.closingTimestamp may be in the past")
	}
	if prop.Terms.ValidationTimestamp == 0 {
		r.Err("prop.terms.validationTimestamp is missing or 0")
	} else if time.Unix(prop.Terms.ValidationTimestamp, 0).Before(time.Now()) {
		r.Warn("prop.terms.validationTimestamp may be in the past")
	}
	if prop.Terms.EnactmentTimestamp == 0 {
		r.Err("prop.terms.enactmentTimestamp is missing or 0")
	} else if time.Unix(prop.Terms.EnactmentTimestamp, 0).Before(time.Now()) {
		r.Warn("prop.terms.enactmentTimestamp may be in the past")
	}

	newAsset := prop.Terms.GetNewAsset()
	if newAsset == nil {
		r.Err("prop.terms.newAsset is missing or null")
		return
	}
	if newAsset.Changes == nil {
		r.Err("prop.terms.newAsset.changes is missing or null")
		return
	}

	switch source := newAsset.Changes.Source.(type) {
	case *types.AssetSource_Erc20:
		contractAddress := source.Erc20.GetContractAddress()
		if len(contractAddress) <= 0 {
			r.Err("prop.terms.newAsset.changes.erc20.contractAddress is missing")
		} else if !isValidEthereumAddress(contractAddress) {
			r.Warn("prop.terms.newAsset.changes.erc20.contractAddress may not be a valid ethereum address")
		}
	default:
		r.Err("unsupported prop.terms.newAsset.changes")
	}
}
