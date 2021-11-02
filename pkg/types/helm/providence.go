//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm

import (
	"bytes"
	"io"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"golang.org/x/crypto/openpgp/clearsign"
)

type Provenance struct {
	ChartMetadata map[string]string
	SumCollection *SumCollection
	Block         *clearsign.Block
}

type SumCollection struct {
	Files  map[string]string `json:"files"`
	Images map[string]string `json:"images,omitempty"`
}

func (p *Provenance) Unmarshal(reader io.Reader) error {

	buf := &bytes.Buffer{}
	read, err := buf.ReadFrom(reader)

	if err != nil {
		return errors.New("Failed to read from buffer")
	}

	if read == 0 {
		return errors.New("Provenance file contains no content")
	}

	rawProvenanceFile := buf.Bytes()

	block, _ := clearsign.Decode(rawProvenanceFile)

	if block == nil {
		return errors.New("Unable to decode provenance file")
	}

	if block.ArmoredSignature == nil {
		return errors.New("Unable to locate armored signature in provenance file")
	}

	p.Block = block

	err = p.parseMessageBlock(block.Plaintext)

	if err != nil {
		return errors.Wrap(err, "Error occurred parsing message block")
	}

	return nil

}

func (p *Provenance) parseMessageBlock(data []byte) error {

	parts := bytes.Split(data, []byte("\n...\n"))
	if len(parts) < 2 {
		return errors.New("message block must have at least two parts")
	}

	sc := &SumCollection{}

	err := yaml.Unmarshal(parts[1], sc)

	if err != nil {
		return errors.Wrap(err, "Error occurred parsing SumCollection")
	}

	p.SumCollection = sc

	return nil
}

func (p *Provenance) GetChartAlgorithmHash() (string, string, error) {

	if p.SumCollection == nil || p.SumCollection.Files == nil {
		return "", "", errors.New("Unable to locate chart hash")

	}

	files := p.SumCollection.Files

	for _, value := range files {

		parts := strings.Split(value, ":")

		if len(parts) != 2 {
			return "", "", errors.New("Invalid hash found in Provenance file")
		}

		return parts[0], parts[1], nil
	}

	// Return error if no keys found
	return "", "", errors.New("No checksums found")

}
