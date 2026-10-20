package proxy

import (
	"errors"
	"strings"

	"github.com/tidwall/gjson"
)

// requestPolicyModel reads the same unambiguous model sent upstream. Checking
// one occurrence while forwarding a different duplicate would split policy
// enforcement from execution. Unrelated fields remain byte-for-byte intact.
func requestPolicyModel(body []byte) (string, error) {
	if !gjson.ValidBytes(body) || !gjson.ParseBytes(body).IsObject() {
		return "", errors.New("request must be a JSON object")
	}
	var model string
	var found bool
	var modelErr error
	gjson.ParseBytes(body).ForEach(func(key, value gjson.Result) bool {
		if !strings.EqualFold(key.String(), "model") {
			return true
		}
		if found || key.String() != "model" || value.Type != gjson.String {
			modelErr = errors.New("model must be a single string field spelled model")
			return false
		}
		found = true
		model = strings.TrimSpace(value.String())
		return true
	})
	return model, modelErr
}
