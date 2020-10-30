package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"code.vegaprotocol.io/vega/faucet"
)

const (
	// use default address of faucet
	defaultAddress = "http://0.0.0.0:1790"
)

type Client struct {
	clt     *http.Client
	addr    string
	mintURL string
}

func New(addr string) (*Client, error) {
	mintURL, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	mintURL.Path = path.Join(mintURL.Path, "/api/v1/mint")
	return &Client{
		clt:     &http.Client{},
		addr:    defaultAddress,
		mintURL: mintURL.String(),
	}, nil
}

func NewDefault() (*Client, error) {
	return New(defaultAddress)
}

func (c *Client) Mint(party, asset string, amount uint64) error {
	body := faucet.MintRequest{
		Party:  party,
		Asset:  asset,
		Amount: amount,
	}
	jbytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.mintURL, bytes.NewReader(jbytes))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := c.clt.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	resbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	faucetRes := &faucet.MintResponse{}
	err = json.Unmarshal(resbody, faucetRes)
	if err != nil {
		return err
	}
	if !faucetRes.Success {
		return errors.New("unable to allocate new funds")
	}
	return nil
}
