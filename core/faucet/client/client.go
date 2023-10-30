// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"code.vegaprotocol.io/vega/core/faucet"
)

const (
	// use default address of faucet.
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

func (c *Client) Mint(party, asset, amount string) error {
	body := faucet.MintRequest{
		Party:  party,
		Asset:  asset,
		Amount: amount,
	}
	jbytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.mintURL, bytes.NewReader(jbytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
