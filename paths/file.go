package paths

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"

	"github.com/BurntSushi/toml"
)

var (
	ErrEmptyResponse        = errors.New("empty response")
	ErrEmptyFile            = errors.New("empty file")
	ErrContentLooksLikeHTML = errors.New("the content looks like it contains HTML, be sure your file has TOML formatting")
	ErrContentLooksLikeJSON = errors.New("the content looks like it contains JSON, be sure your file has TOML formatting")
)

func FetchStructuredFile(url string, v interface{}) error {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("couldn't load file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("couldn't read HTTP response body: %w", err)
	}

	if len(body) == 0 {
		return ErrEmptyResponse
	}

	if err := decodeTOMLFile(body, v); err != nil {
		return err
	}

	return nil
}

func decodeTOMLFile(content []byte, v interface{}) error {
	if content[0] == '<' {
		return ErrContentLooksLikeHTML
	}

	if content[0] == '{' {
		return ErrContentLooksLikeJSON
	}

	if _, err := toml.Decode(string(content), v); err != nil {
		return fmt.Errorf("invalid TOML document: %w", err)
	}

	return nil
}

func ReadStructuredFile(path string, v interface{}) error {
	buf, err := vgfs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("couldn't read file: %w", err)
	}

	if len(buf) == 0 {
		return ErrEmptyFile
	}

	if err := decodeTOMLFile(buf, v); err != nil {
		return err
	}

	return nil
}

func WriteStructuredFile(path string, v interface{}) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(v); err != nil {
		return fmt.Errorf("couldn't encode to TOML: %w", err)
	}

	if err := vgfs.WriteFile(path, buf.Bytes()); err != nil {
		return fmt.Errorf("couldn't write file: %w", err)
	}

	return nil
}

func ReadEncryptedFile(path string, passphrase string, v interface{}) error {
	encryptedBuf, err := vgfs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("couldn't read secure file: %w", err)
	}

	if len(encryptedBuf) == 0 {
		return ErrEmptyFile
	}

	buf, err := vgcrypto.Decrypt(encryptedBuf, passphrase)
	if err != nil {
		return fmt.Errorf("couldn't decrypt content: %w", err)
	}

	err = json.Unmarshal(buf, v)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal content: %w", err)
	}

	return nil
}

func WriteEncryptedFile(path string, passphrase string, v interface{}) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("couldn't marshal content: %w", err)
	}

	encryptedBuf, err := vgcrypto.Encrypt(buf, passphrase)
	if err != nil {
		return fmt.Errorf("couldn't encrypt content: %w", err)
	}

	if err := vgfs.WriteFile(path, encryptedBuf); err != nil {
		return fmt.Errorf("couldn't write secure file: %w", err)
	}

	return nil
}
