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

package encoding

import (
	"encoding/base64"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

// Duration is a wrapper over an actual duration so we can represent
// them as string in the toml configuration.
type Duration struct {
	time.Duration
}

// Get returns the stored duration.
func (d *Duration) Get() time.Duration {
	return d.Duration
}

// UnmarshalText unmarshal a duration from bytes.
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func (d *Duration) UnmarshalFlag(s string) error {
	return d.UnmarshalText([]byte(s))
}

// MarshalText marshal a duraton into bytes.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d Duration) MarshalFlag() (string, error) {
	bz, err := d.MarshalText()
	return string(bz), err
}

// LogLevel is wrapper over the actual log level
// so they can be specified as strings in the toml configuration.
type LogLevel struct {
	logging.Level
}

// Get return the store value.
func (l *LogLevel) Get() logging.Level {
	return l.Level
}

// UnmarshalText unmarshal a loglevel from bytes.
func (l *LogLevel) UnmarshalText(text []byte) error {
	var err error
	l.Level, err = logging.ParseLevel(string(text))
	return err
}

func (l *LogLevel) UnmarshalFlag(s string) error {
	return l.UnmarshalText([]byte(s))
}

// MarshalText marshal a loglevel into bytes.
func (l LogLevel) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

type Bool bool

func (b *Bool) UnmarshalFlag(s string) error {
	if s == "true" {
		*b = true
	} else if s == "false" {
		*b = false
	} else {
		return fmt.Errorf("only `true' and `false' are valid values, not `%s'", s)
	}
	return nil
}

type Base64 []byte

func (b *Base64) UnmarshalFlag(s string) error {
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = dec
	return nil
}

func (b Base64) MarshalFlag() (string, error) {
	return base64.StdEncoding.EncodeToString(b), nil
}

type NodeMode string

const (
	NodeModeValidator   NodeMode = "validator"
	NodeModeFull        NodeMode = "full"
	NodeModeSeed        NodeMode = "seed"
	NodeModeUnsupported NodeMode = "unsupported"
)

func NodeModeFromString(s string) (NodeMode, error) {
	switch NodeMode(s) {
	case NodeModeValidator:
		return NodeModeValidator, nil
	case NodeModeFull:
		return NodeModeFull, nil
	case NodeModeSeed:
		return NodeModeSeed, nil
	default:
		return NodeModeUnsupported, fmt.Errorf("%s is not a valid node mode, expected [validator, full, seed]", s)
	}
}

// UnmarshalText unmarshal a loglevel from bytes.
func (n *NodeMode) UnmarshalText(text []byte) error {
	var err error
	*n, err = NodeModeFromString(string(text))
	return err
}

// MarshalText marshal a loglevel into bytes.
func (n NodeMode) MarshalText() ([]byte, error) {
	return []byte(n), nil
}

type DataNodeRetentionMode string

const (
	DataNodeRetentionModeStd         DataNodeRetentionMode = "standard"
	DataNodeRetentionModeArchive     DataNodeRetentionMode = "archive"
	DataNodeRetentionModeLite        DataNodeRetentionMode = "lite"
	DataNodeRetentionModeUnsupported DataNodeRetentionMode = "unsupported"
)

func DataNodeRetentionModeFromString(s string) (DataNodeRetentionMode, error) {
	switch DataNodeRetentionMode(s) {
	case DataNodeRetentionModeStd:
		return DataNodeRetentionModeStd, nil
	case DataNodeRetentionModeArchive:
		return DataNodeRetentionModeArchive, nil
	case DataNodeRetentionModeLite:
		return DataNodeRetentionModeLite, nil
	default:
		return DataNodeRetentionModeUnsupported, fmt.Errorf("%s is not a valid data node retention mode, expected [standard, archive, lite]", s)
	}
}
