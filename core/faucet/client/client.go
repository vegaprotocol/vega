// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
